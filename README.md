# LaServer-Go

LaServer 的 Go 重构项目，逐步迁移旧客户端游戏接口与 ENet/UDP 联机逻辑，理清业务边界并简化部署。

> 当前处于开发阶段，尚不能完整替代旧 LaServer。已接入扫描清单中的 93/201 条路由，其余 108 条尚未迁移；本轮新增接口已完成 Go 静态构建检查，测试与真实客户端验收尚未完成。

## 服务与技术栈

| 服务 | 职责 | 当前状态 |
| --- | --- | --- |
| `api-server` | 游戏 HTTP 接口、账号和存档等持久化业务；后续承载管理 API | 可独立运行，已实现首批接口 |
| `room-server` | ENet/UDP、连接会话、房间与玩家实时同步 | 仅建立独立 Go module，暂无运行入口 |

两个服务位于同一仓库，各自维护 Go module、入口、配置和发布流程，不要求同时启动。服务之间不直接导入对方的 internal 包，也不共享数据库作为通信通道。

- Go 最低版本：1.26.0；当前验证工具链：Go 1.27.4。
- HTTP：标准库 `net/http`；日志：`log/slog`。
- 数据库：SQLite，使用 `database/sql` 和 `modernc.org/sqlite v1.62.0`，不引入 Redis 或 MySQL。
- 实时房间计划使用内存状态；ENet 库仍需兼容验证后确定。

## 目录

```text
LaServer-Go/
├── services/
│   ├── api-server/
│   │   ├── cmd/server/          API 启动入口
│   │   ├── internal/
│   │   │   ├── handler/         HTTP 接口
│   │   │   ├── service/         业务逻辑
│   │   │   ├── repository/      SQLite 数据访问
│   │   │   ├── model/           旧表持久化模型
│   │   │   ├── middleware/      请求体与访问日志
│   │   │   ├── config/          配置加载
│   │   │   └── app/             启动与生命周期
│   │   ├── migrations/          SQLite schema v1–v8
│   │   ├── config/             无凭据配置示例
│   │   ├── go.mod
│   │   └── go.sum
│   └── room-server/
│       ├── README.md
│       └── go.mod              尚无实现或第三方依赖
├── docs/                       设计、开发计划和兼容说明
├── json/                       实体映射及接口进度报告
├── scripts/                    生成与验证工具
└── AGENTS.md                   项目约束
```

API 按 handler、service、repository、model、middleware、config、app 分层；room-server 保持独立 Go module 和进程边界。

## 启动 API

需要 Go 1.26.0 或更新版本。以下在 Ubuntu/WSL 的仓库根目录执行；无需启动 room-server 或数据库容器。

```sh
cd services/api-server
cp config/api-server.example.json config/api-server.json
# 编辑配置后，为全新的开发数据库显式初始化。
go run ./cmd/server migrate --config config/api-server.json
# 正常启动或重启。
go run ./cmd/server --config config/api-server.json
```

Windows PowerShell 使用 `Copy-Item config/api-server.example.json config/api-server.json` 复制配置，其余 Go 命令相同；Windows 实机运行尚未验收。

默认监听 `127.0.0.1:8443`，未配置证书时使用 HTTP，健康检查为 `GET /healthz`。`tls_cert` 与 `tls_key` 成对设置后启用 HTTPS。路径相对于配置文件目录解析。

| 配置 | 用途 |
| --- | --- |
| `database` | SQLite 文件路径，默认 `../data/api-server.db` |
| `game_config_dir` | 只读游戏配置目录；收藏品需要 collectibles.json，地图定义需要 map.json |
| `room_host` / `room_port` | 下发给客户端的实时服务 IPv4 和端口；下发地址不代表联机服务已就绪 |
| `auto_authorize` | 是否自动授权新账号，默认 false |
| `recreate_missing_account` | 是否按原 ID 补建缺失账号，默认 false |
| `allow_fake_commerce` | 启用本地模拟购买，默认 false；仅接受唯一交易 ID，真实凭据不发货 |
| `default_currency` / `current_season` | 新账号初始货币与当前赛季 |
| `resource_auth_token` | 本地配置有效资源 CDN token；缺失时对应接口返回 503 |
| `starwatch_jwt_secret` / `starwatch_broker_url` | 本地配置 MQTT JWT 签名密钥与 broker 地址；缺失时对应接口返回 503 |

在当前与旧 LaServer 并列的工作区中，`game_config_dir` 可设为 `../../../../LaServer/services/game-api/config`；其他部署需自行提供经过确认的配置文件。本仓库不自动复制旧客户端资源或真实账号数据，缺少必需资源的对应接口返回 503。

`migrate` 支持初始化空数据库并逐版升级本项目 schema v1–v10，拒绝无法识别的非空数据库；正常启动要求 schema v10，不自动改表。已是 v10 时重跑 migrate 不会重复建表。不要将旧 Python 实例数据库直接复制到新服务路径，迁移前应备份并单独验证。

## 接口进度与边界

已实现账号创建/登录、存档、货币、排名、房间发现、检查点、构建版本、设备能力、基础变量、公告、支持状态，以及收藏品、成就、角标、成就定义、地图定义。

游戏接口使用 POST，支持 text/plain 中的 JSON，请求体上限 2 MiB。清单内未迁移接口返回 501，未知路径返回 404；WebSocket、MQTT、自动全解锁与部分静态响应版本分流尚未完成。

登录仍继承旧服务可仅凭 user ID 换取会话的行为，尚未建立可靠的设备凭据校验。当前版本用于隔离本地开发，不应直接开放公网。存档接口要求有效会话，防止跨账号读取或写入。

完整字段、行为差异和验收范围见[接口开发说明](docs/api-development.md)与[路由清单](json/api/legacy-routes.json)。模型来源见[实体映射](docs/entities.md)。

## 检查与测试

在仓库根目录运行：

```sh
sh scripts/verify/api-server.sh
```

脚本优先使用现有 `~/.local/opt/go1.27.4/bin`，也可使用 PATH 中的 Go；在 API module 内执行测试、vet、Linux 构建和无 CGO 的 Windows amd64 交叉构建，日志集中写入根 logs/。

手动命令须进入服务目录，根目录没有统一 go.mod：

```sh
cd services/api-server
go test ./...
go vet ./...
go build ./...
```

最近一轮已有 17 项测试通过，竞态检查与双平台编译通过；这些结果不代表真实客户端、Windows 实机、TLS 互通或真实数据迁移已经验收。room-server 尚无代码，暂不提供构建与启动命令。

## 文档与协作

- [项目约束](AGENTS.md)
- [重构设计](docs/superpowers/specs/2026-11-11-go-rewrite-design.md)
- [开发计划](docs/superpowers/plans/2026-11-11-go-rewrite.md)
- [复刻待办与剩余接口](docs/TODO.md)
- [开发与修复记录](docs/TodoList.md)

运行数据、真实配置、证书私钥、日志、抓包和构建产物不提交。示例配置、源码、依赖锁文件及脱敏报告纳入版本控制。旧 LaServer 仅作为兼容参考，不覆盖其未提交改动或真实数据。
