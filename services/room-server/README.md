# 实时联机服务

独立 Go module，计划使用 cmd/server/main.go 入口和 internal/transport、protocol、session、room、config、app 目录。当前已实现 internal/protocol 中的时钟 Ping/Pong 编解码。可运行入口、传输和会话层待实现。该模块目前使用 Go 标准库，无第三方依赖。

此服务独立构建和部署，不导入 api-server 的 internal 包，不直接读写其 SQLite。ENet/UDP 兼容验证和实现按项目开发计划推进。
