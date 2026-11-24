// 服务依赖组装、监听与生命周期管理

package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/JasonCao2003/LaServer-Go/services/room-server/internal/config"
)

// Transport is the lifecycle of a bound room transport. Serve must stop after
// its context is canceled. It calls ready after its event loop can accept
// connections. Close runs after Serve has returned.
type Transport interface {
	Serve(ctx context.Context, ready func()) error
	Close() error
}

// 运行房间服务并管理监听生命周期
func Run(ctx context.Context, cfg config.Config, openTransport func(config.Config) (Transport, error)) error {
	transport, err := openTransport(cfg)
	if err != nil {
		return fmt.Errorf("open room transport: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.ManagementListen)
	if err != nil {
		return errors.Join(fmt.Errorf("open room management: %w", err), transport.Close())
	}

	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var ready atomic.Bool
	management := &http.Server{
		Handler:           healthHandler(func() bool { return serveCtx.Err() == nil && ready.Load() }),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       20 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}
	transportDone := make(chan error, 1)
	managementDone := make(chan error, 1)
	go func() {
		err := transport.Serve(serveCtx, func() { ready.Store(true) })
		ready.Store(false)
		transportDone <- err
	}()
	go func() { managementDone <- management.Serve(listener) }()

	var serveErr error
	select {
	case err := <-transportDone:
		transportDone = nil
		if ctx.Err() == nil {
			serveErr = unexpectedExit("room transport", err)
		}
	case err := <-managementDone:
		managementDone = nil
		if ctx.Err() == nil {
			serveErr = unexpectedExit("room management", err)
		}
	case <-ctx.Done():
	}

	ready.Store(false)
	cancel()
	managementErr := management.Close()
	// Close the listener even if Serve has not registered it yet.
	listenerErr := listener.Close()
	if errors.Is(listenerErr, net.ErrClosed) {
		listenerErr = nil
	}
	if managementDone != nil {
		<-managementDone
	}
	if transportDone != nil {
		<-transportDone
	}
	return errors.Join(serveErr, managementErr, listenerErr, transport.Close())
}

// 构造非预期退出错误
func unexpectedExit(name string, err error) error {
	if err == nil {
		return fmt.Errorf("%s stopped unexpectedly", name)
	}
	return fmt.Errorf("%s stopped: %w", name, err)
}

// 提供健康检查 HTTP 处理器
func healthHandler(ready func() bool) http.Handler {
	mux := http.NewServeMux()
	alive := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}
	mux.HandleFunc("GET /healthz", alive)
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		alive(w, r)
	})
	return mux
}
