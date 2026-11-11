# LaServer-Go

LaServer 的 Go 重构项目。目标是理清业务边界、统一数据写入规则和简化启动，按游戏 API、管理功能、实时联机的顺序迁移。最终业务 API 和实时服务均使用 Go，分别构建、启动与部署，不要求同时启动。

当前状态：已有可独立运行的游戏 API，首批实现 16 个旧接口；完整接口迁移、实时服务和真实客户端验收尚未完成。见[接口进度与启动说明](docs/api-development.md)。

数据库使用 SQLite，不引入 Redis 或 MySQL。业务 API 管理持久化数据，实时服务在内存中维护房间状态。

- [架构设计（待评审）](docs/superpowers/specs/2026-11-11-go-rewrite-design.md)
- [开发计划草案](docs/superpowers/plans/2026-11-11-go-rewrite.md)

仓库仅收录经检查可公开的源码和文档，不收录真实账号数据、证书私钥、原始抓包或未经许可的客户端资源。
