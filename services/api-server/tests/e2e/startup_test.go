// 启动、迁移与退出流程测试

package e2e

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}

func buildServer(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "api-server")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/server")
	cmd.Dir = moduleRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build server: %v %s", err, out)
	}
	return binary
}

func process(ctx context.Context, binary string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, binary, args...)
}

func TestMigrateStartAndShutdownWithoutRealtime(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM subprocess test runs on Linux")
	}
	binary := buildServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := t.TempDir()
	config := filepath.Join(dir, "server.json")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	data, err := json.Marshal(map[string]any{"listen": addr, "database": "data/test.db", "room_host": "127.0.0.1", "room_port": 25565})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(config, data, 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := process(ctx, binary, "migrate", "--config", config).CombinedOutput(); err != nil {
		t.Fatalf("migrate: %v %s", err, out)
	}
	db, err := repository.Open(filepath.Join(dir, "data", "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("INSERT INTO user(id,session,recovery_token) VALUES('startup-test','test-session','test-recovery')"); err != nil {
		t.Fatal(err)
	}
	if out, err := process(ctx, binary, "migrate", "--config", config).CombinedOutput(); err != nil {
		t.Fatalf("repeat migrate: %v %s", err, out)
	}
	if err := repository.CheckSchema(db); err != nil {
		t.Fatal(err)
	}
	var accounts int
	if err := db.QueryRow("SELECT count(*) FROM user WHERE id='startup-test' AND session='test-session' AND recovery_token='test-recovery'").Scan(&accounts); err != nil || accounts != 1 {
		t.Fatalf("account after repeat migrate: count=%d, err=%v", accounts, err)
	}
	cmd := process(ctx, binary, "--config", config)
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	client := http.Client{Timeout: 200 * time.Millisecond}
	ready := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		resp, err := client.Get("http://" + addr + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				ready = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatal("API did not start without realtime server")
	}
	if err = cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err = cmd.Wait(); err != nil {
		t.Fatalf("graceful shutdown failed: %v", err)
	}
	listener, err = net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listener leaked: %v", err)
	}
	listener.Close()
}
