// 服务启动与健康检查测试

package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/config"
)

type testTransport struct {
	started chan struct{}
	closed  chan struct{}
	failure error
}

func (transport *testTransport) Serve(ctx context.Context, ready func()) error {
	ready()
	close(transport.started)
	if transport.failure != nil {
		return transport.failure
	}
	<-ctx.Done()
	return ctx.Err()
}

func (transport *testTransport) Close() error {
	close(transport.closed)
	return nil
}

func TestRunClosesTransportWhenManagementListenFails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	tr := &testTransport{started: make(chan struct{}), closed: make(chan struct{})}
	err = Run(context.Background(), config.Config{ManagementListen: listener.Addr().String()},
		func(config.Config) (Transport, error) { return tr, nil })
	if err == nil {
		t.Fatal("Run accepted a port already in use")
	}
	select {
	case <-tr.closed:
	default:
		t.Fatal("transport not closed after partial startup failure")
	}
	select {
	case <-tr.started:
		t.Fatal("transport loop started after partial startup failure")
	default:
	}
}

func TestRunStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr := &testTransport{started: make(chan struct{}), closed: make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		result <- Run(ctx, config.Config{ManagementListen: "127.0.0.1:0"},
			func(config.Config) (Transport, error) { return tr, nil })
	}()
	select {
	case <-tr.started:
	case err := <-result:
		t.Fatalf("Run exited before startup: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("transport did not start")
	}
	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop")
	}
	select {
	case <-tr.closed:
	default:
		t.Fatal("transport not closed during shutdown")
	}
}

func TestRunReturnsTransportFailure(t *testing.T) {
	failure := errors.New("transport failed")
	tr := &testTransport{started: make(chan struct{}), closed: make(chan struct{}), failure: failure}
	err := Run(context.Background(), config.Config{ManagementListen: "127.0.0.1:0"},
		func(config.Config) (Transport, error) { return tr, nil })
	if !errors.Is(err, failure) {
		t.Fatalf("Run error = %v, want transport failure", err)
	}
}

func TestHealthHandler(t *testing.T) {
	for _, test := range []struct {
		path   string
		method string
		ready  bool
		want   int
	}{
		{"/healthz", http.MethodGet, false, http.StatusNoContent},
		{"/readyz", http.MethodGet, true, http.StatusNoContent},
		{"/readyz", http.MethodGet, false, http.StatusServiceUnavailable},
		{"/healthz", http.MethodPost, false, http.StatusMethodNotAllowed},
		{"/unknown", http.MethodGet, false, http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		request, err := http.NewRequest(test.method, test.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		healthHandler(func() bool { return test.ready }).ServeHTTP(response, request)
		if response.Code != test.want {
			t.Errorf("%s %s = %d, want %d", test.method, test.path, response.Code, test.want)
		}
	}
}
