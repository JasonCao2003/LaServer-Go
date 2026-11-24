//go:build (windows && amd64) || (linux && cgo)

// 房间服务启动与退出测试

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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
	binary := filepath.Join(t.TempDir(), "room-server")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/server")
	cmd.Dir = moduleRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build server: %v %s", err, out)
	}
	return binary
}

func TestRunHelpAndInvalidArguments(t *testing.T) {
	binary := buildServer(t)
	if out, err := exec.Command(binary, "-help").CombinedOutput(); err != nil {
		t.Fatalf("-help failed: %v %s", err, out)
	}
	for _, args := range [][]string{nil, {"unexpected"}, {"-unknown"}} {
		if out, err := exec.Command(binary, args...).CombinedOutput(); err == nil {
			t.Fatalf("invalid arguments accepted: %v %s", args, out)
		}
	}
}
