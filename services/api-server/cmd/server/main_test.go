package main

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
)

func TestGameAPIProcess(t *testing.T) {
	if os.Getenv("LASERVER_TEST_CHILD") != "1" {
		t.Skip("subprocess helper; exercised by startup test")
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("LASERVER_TEST_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	os.Args = append([]string{"api-server"}, args...)
	if err := run(); err != nil {
		t.Fatal(err)
	}
}

func process(t *testing.T, ctx context.Context, args ...string) *exec.Cmd {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGameAPIProcess$")
	cmd.Env = append(os.Environ(), "LASERVER_TEST_CHILD=1", "LASERVER_TEST_ARGS="+string(b))
	return cmd
}

func TestMigrateStartAndShutdownWithoutRealtime(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM subprocess test runs on Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
	if out, err := process(t, ctx, "migrate", "--config", config).CombinedOutput(); err != nil {
		t.Fatalf("migrate: %v %s", err, out)
	}
	if out, err := process(t, ctx, "migrate", "--config", config).CombinedOutput(); err == nil {
		t.Fatalf("second migrate accepted: %s", out)
	}
	cmd := process(t, ctx, "--config", config)
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
