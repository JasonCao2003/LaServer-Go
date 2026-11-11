package main

import (
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/app"
	"log/slog"
	"os"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api-server stopped", "error", err)
		os.Exit(1)
	}
}
func run() error { return app.Run(os.Args[1:]) }
