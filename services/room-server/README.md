# 实时联机服务

独立 Go module，计划使用 cmd/server/main.go 入口和 internal/transport、protocol、session、room、config、app 目录。当前尚未实现，无可运行入口或第三方依赖，因此暂无 go.sum。

此服务独立构建和部署，不导入 api-server 的 internal 包，不直接读写其 SQLite。ENet/UDP 兼容验证和实现按项目开发计划推进。
