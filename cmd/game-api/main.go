package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/JasonCao2003/LaServer-Go/internal/gameapi"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("game-api stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	args := os.Args[1:]
	migrate := len(args) > 0 && args[0] == "migrate"
	if migrate {
		args = args[1:]
	}
	fs := flag.NewFlagSet("game-api", flag.ContinueOnError)
	path := fs.String("config", "config/game-api.json", "JSON config file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	cfg, err := gameapi.LoadConfig(*path)
	if err != nil {
		return err
	}
	if migrate {
		if err = os.MkdirAll(filepath.Dir(cfg.Database), 0700); err != nil {
			return err
		}
	}
	db, err := gameapi.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	if migrate {
		return gameapi.Migrate(context.Background(), db)
	}
	api, err := gameapi.New(db, cfg)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: cfg.Listen, Handler: api, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	done := make(chan error, 1)
	go func() {
		if cfg.TLSCert != "" {
			done <- server.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			done <- server.ListenAndServe()
		}
	}()
	select {
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
		err = <-done
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
