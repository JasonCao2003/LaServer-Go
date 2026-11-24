// 进房准入校验测试

package session_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/protocol"
	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/session"
)

func TestVerifyAdmission(t *testing.T) {
	bearer := strings.Repeat("z", 32)
	proof := protocol.AdmissionProof{AccountID: "account", Signature: [32]byte{0xab}}
	until := time.Now().Add(time.Minute).Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/room/admissions/verify" || r.Method != http.MethodPost ||
			r.Header.Get("Authorization") != "Bearer "+bearer {
			t.Error("incorrect verification request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil ||
			request["account_id"] != "account" || request["room_uri"] != "127.0.0.1:25565" ||
			request["signature"] != "ab"+strings.Repeat("00", 31) {
			t.Error("incorrect proof body")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"account_id": "account", "expires_at": until})
	}))
	defer server.Close()
	verify, err := session.NewAdmissionVerifier(server.URL, bearer)
	if err != nil {
		t.Fatal(err)
	}
	got, err := verify.Verify(context.Background(), "127.0.0.1:25565", proof)
	if err != nil || got.AccountID != "account" || got.ExpiresAt.Unix() != until {
		t.Fatalf("verification returned %+v, %v", got, err)
	}
}

func TestVerifyAdmissionRejectsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	verify, err := session.NewAdmissionVerifier(server.URL, strings.Repeat("z", 32))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verify.Verify(context.Background(), "127.0.0.1:25565", protocol.AdmissionProof{}); !errors.Is(err, session.ErrAdmissionRejected) {
		t.Fatalf("verification error = %v", err)
	}
}

func TestAdmissionRejectsExpiredIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"account_id": "account", "expires_at": time.Now().Add(-time.Minute).Unix()})
	}))
	defer server.Close()
	v, err := session.NewAdmissionVerifier(server.URL, strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.Verify(context.Background(), "127.0.0.1:25565", protocol.AdmissionProof{AccountID: "account"})
	if err == nil || got.AccountID != "" {
		t.Fatal("expired identity accepted")
	}
}

func TestAdmissionHonorsDeadline(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-release }))
	defer server.Close()
	defer close(release)
	v, err := session.NewAdmissionVerifier(server.URL, strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := v.Verify(ctx, "127.0.0.1:25565", protocol.AdmissionProof{AccountID: "account"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline error=%v", err)
	}
}
