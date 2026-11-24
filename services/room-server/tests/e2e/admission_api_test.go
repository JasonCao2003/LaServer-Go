package e2e

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/session"
)

// Supply the separately built API executable to exercise both modules across HTTP.
func TestAdmissionAgainstRealAPI(t *testing.T) {
	binary := os.Getenv("SKYLA_TEST_API_BINARY")
	if binary == "" {
		t.Skip("set SKYLA_TEST_API_BINARY to the API executable for cross-service verification")
	}
	dir := t.TempDir()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	secret := func() string { var b [32]byte; rand.Read(b[:]); return hex.EncodeToString(b[:]) }
	bearer := secret()
	cfg, err := json.Marshal(map[string]any{"listen": addr, "database": "data/test.db", "room_host": "127.0.0.1", "room_port": 25565, "login_mode": "legacy_dev", "auto_authorize": true, "room_admission_key": secret(), "room_admission_bearer": bearer})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "api.json")
	if err := os.WriteFile(file, cfg, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, binary, "migrate", "--config", file).Run(); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, binary, "--config", file)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cancel(); _ = cmd.Wait() }()
	base := "http://" + addr
	client := &http.Client{Timeout: time.Second}
	for {
		resp, err := client.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		select {
		case <-ctx.Done():
			t.Fatal("API health check timed out")
		case <-time.After(20 * time.Millisecond):
		}
	}
	post := func(path string, body any, dest any) {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Post(base+path, "text/plain", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("%s: HTTP %d", path, resp.StatusCode)
		}
		if err = json.NewDecoder(resp.Body).Decode(dest); err != nil {
			t.Fatal(err)
		}
	}
	var player struct{ User, Session string }
	post("/account/create", map[string]string{"device_name": "phase2", "language": "zh"}, &player)
	var discovery struct{ URI, Signature string }
	post("/account/find_previous_or_empty", map[string]string{"user": player.User, "session": player.Session}, &discovery)
	sig, err := hex.DecodeString(discovery.Signature)
	if err != nil || len(sig) != 32 {
		t.Fatal("invalid discovery signature")
	}
	proof := protocol.AdmissionProof{AccountID: player.User}
	copy(proof.Signature[:], sig)
	v, err := session.NewAdmissionVerifier(base, bearer)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := v.Verify(ctx, discovery.URI, proof)
	if err != nil || identity.AccountID != player.User || !identity.ExpiresAt.After(time.Now()) {
		t.Fatalf("verified admission: %v", err)
	}
	proof.Signature[0] ^= 1
	if _, err = v.Verify(ctx, discovery.URI, proof); !errors.Is(err, session.ErrAdmissionRejected) {
		t.Fatalf("invalid signature: %v", err)
	}
}
