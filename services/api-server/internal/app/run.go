package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/handler"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/middleware"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func Run(args []string) error {
	migrate := len(args) > 0 && args[0] == "migrate"
	if migrate {
		args = args[1:]
	}
	fs := flag.NewFlagSet("api-server", flag.ContinueOnError)
	path := fs.String("config", "config/api-server.json", "JSON config file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	if migrate {
		if err = os.MkdirAll(filepath.Dir(cfg.Database), 0700); err != nil {
			return err
		}
	}
	db, err := repository.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	if migrate {
		return repository.Migrate(context.Background(), db)
	}
	if err = config.ValidateRuntime(cfg); err != nil {
		return err
	}
	if err = repository.CheckSchema(db); err != nil {
		return err
	}
	api := handler.NewAPI(service.New(repository.New(db), &cfg), &cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: cfg.Listen, Handler: middleware.Access(api), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
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
