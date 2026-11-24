// api-server 启动入口

package main

import (
	"log/slog"
	"os"

	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/app"
	"github.com/gin-gonic/gin"
)

// 程序入口
func main() {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	if err := run(); err != nil {
		slog.Error("api-server stopped", "error", err)
		os.Exit(1)
	}
}

// 启动服务主流程
func run() error { return app.Run(os.Args[1:]) }
