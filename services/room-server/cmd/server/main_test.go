//go:build (windows && amd64) || (linux && cgo)

// 房间服务启动与退出测试

package main

import (
	"context"
	"testing"
)

func TestRunHelpAndInvalidArguments(t *testing.T) {
	if err := run(context.Background(), []string{"-help"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"unexpected"}, {"-unknown"}} {
		if err := run(context.Background(), args); err == nil {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
}
