# LaServer-Go

LaServer 的 Go 重构项目，逐步迁移旧客户端游戏接口与 ENet/UDP 联机逻辑，理清业务边界并简化部署。

> 当前处于开发阶段，尚不能完整替代旧 LaServer。扫描清单中的 201 条静态 HTTP 路由已有 Go 入口，但实时协议、双端同步和客户端全链路仍未完成。新实例允许从空 SQLite 启动；旧账号数据迁移按需进行，不作为继续开发的前提。

## 服务与技术栈

| 服务 | 职责 | 当前状态 |
| --- | --- | --- |
| `api-server` | 游戏 HTTP 接口、账号和存档等持久化业务；后续承载管理 API | 可独立运行；201 条静态路由已接入，API↔实时服务联调和客户端验收仍待完成 |
| `room-server` | ENet/UDP、连接会话、房间与玩家实时同步 | Windows amd64 已有单客户端入房证据；Linux 已增加 cgo 原生适配并通过构建及 ABI 测试，真实联机待验收 |
| `admin-panel` | 本机 Web 管理面板 | 独立启动，提供服务状态和好友管理 |

三个服务位于同一仓库，各自维护 Go module、入口、配置和发布流程，不要求同时启动。服务之间不直接导入对方的 internal 包，也不共享数据库作为通信通道。

- Go 最低版本：1.26.0；各服务按各自的 `go.mod` 构建。
- API 路由：Gin；HTTP 服务与 TLS：标准库 `net/http`；日志：`log/slog`。
- 数据库：SQLite，使用 `database/sql` 和 `modernc.org/sqlite v1.62.0`，不引入 Redis 或 MySQL。
- 房间状态保存在 `room-server` 内存中；Windows/Linux 原生传输使用单独构建的 SkyENet 桥接库（DLL / .so）；Linux 需要 cgo 和 C 编译器，真实联机仍待验收。

## 目录

```text
LaServer-Go/
├── services/
│   ├── api-server/
│   │   ├── cmd/server/          API 启动入口
│   │   ├── cmd/legacy-import/   旧库只读预检与显式导入
│   │   ├── internal/
│   │   │   ├── handler/         HTTP 接口
│   │   │   ├── service/         业务逻辑
│   │   │   ├── repository/      SQLite 数据访问
│   │   │   ├── model/           旧表持久化模型
│   │   │   ├── middleware/      请求体与访问日志
│   │   │   ├── config/          配置加载
│   │   │   └── app/             启动与生命周期
│   │   ├── migrations/          SQLite schema v1–v18
│   │   ├── config/             无凭据配置示例
│   │   ├── go.mod
│   │   └── go.sum
│   ├── room-server/
│   │   ├── cmd/server/         Windows/Linux 启动入口
│   │   ├── internal/           transport、protocol、session、room、config、app
│   │   ├── config/             示例配置
│   │   ├── README.md
│   │   └── go.mod
│   └── admin-panel/            独立管理面板
├── docs/                       设计、开发计划和兼容说明
├── json/                       实体映射及接口进度报告
├── scripts/                    生成与验证工具
└── AGENTS.md                   项目约束
```

API 按 handler、service、repository、model、middleware、config、app 分层；room-server 保持独立 Go module 和进程边界。

## 启动 API

需要 Go 1.26.0 或更新版本。以下从仓库根目录进入 API module 执行；仅使用 HTTP 功能时无需启动 room-server 或数据库容器。

```sh
cd services/api-server
cp config/api-server.example.json config/api-server.json
# 编辑配置后，初始化新库或将本项目旧 schema 升级至 v18。
go run ./cmd/server migrate --config config/api-server.json
# 正常启动或重启。
go run ./cmd/server --config config/api-server.json
```

Windows PowerShell 使用 `Copy-Item config/api-server.example.json config/api-server.json` 复制配置，其余 Go 命令相同。当前有 Windows 隔离实例和 MuMu 客户端的部分验证记录，生产部署仍未验收。

默认监听 `127.0.0.1:8443`，未配置证书时使用 HTTP，健康检查为 `GET /healthz`。`tls_cert` 与 `tls_key` 成对设置后启用 HTTPS。路径相对于配置文件目录解析。

| 配置 | 用途 |
| --- | --- |
| `database` | SQLite 文件路径，默认 `../data/api-server.db` |
| `game_config_dir` | 只读游戏配置目录；收藏品需要 collectibles.json，地图定义需要 map.json |
| `room_host` / `room_port` | 下发给客户端的实时服务 IPv4 和端口；下发地址不代表联机服务已就绪 |
| `auto_authorize` | 是否自动授权新账号，默认 false |
| `login_mode` | `credential`（默认）仅开放 `auth/create` 和使用 `recovery_token` 的 `auth/login`；`legacy_dev` 供隔离测试旧客户端，要求监听具体的回环或私有 IP |
| `recreate_missing_account` | 是否按原 ID 补建缺失账号，默认 false；只允许与 `legacy_dev` 同用 |
| `allow_fake_commerce` | 启用本地模拟购买，默认 false；仅接受唯一交易 ID，真实凭据不发货 |
| `default_currency` / `current_season` | 新账号初始货币与当前赛季 |
| `resource_auth_token` | 本地配置有效资源 CDN token；缺失时对应接口返回 503 |
| `starwatch_jwt_secret` / `starwatch_broker_url` | 本地配置 MQTT JWT 签名密钥与 broker 地址；缺失时对应接口返回 503 |

在当前与旧 LaServer 并列的工作区中，`game_config_dir` 可设为 `../../../../LaServer/services/game-api/config`；其他部署需自行提供经过确认的配置文件。本仓库不自动复制旧客户端资源或真实账号数据，缺少必需资源的对应接口返回 503。

`migrate` 可初始化空数据库，或把本项目 schema v1–v17 升级到 v18；拒绝无法识别的非空库。正常启动要求 v18，不自动改表。旧 Python 数据库不能直接当作 Go 数据库使用。

**旧数据按需迁移。** 开发和新实例可直接使用新建 SQLite；如需保留指定旧账号或存档，先用 `go run ./cmd/legacy-import -mode preflight -source <旧 users.db>` 只读盘点，再在隔离目标验证。导入器要求全新目标路径；当前旧实例报告 `partial=true`，默认拒绝部分导入，显式 `-allow-partial` 也不能保证旧账号完整保留。详见 [Task 4 持久化报告](docs/archive/records/api-task4-persistence.md)。

## 启动实时房间服务

当前运行入口支持 Windows amd64 和启用 cgo 的 Linux；需先按 [room-server 说明](services/room-server/README.md) 构建 SkyENet 桥接 DLL 或 .so，并配置 API 的房间入场密钥及服务间凭据。房间进程可独立启动；客户端实际入房时需要 API 验证会话。示例配置只监听回环地址，向局域网设备测试时须按实际网络设置监听和公告地址。Linux 原生适配已有自动化测试，真实客户端联机与多版本兼容仍待验收。Linux 环境准备、命令及验证范围见 [Linux 开发说明](docs/linux-development.md)。

## 接口进度与边界

201 条静态 HTTP 路由已有 Go 入口，账号、存档、货币、任务、购买、礼物、社交内容和部分资源装配具有本地行为测试。路由接入不等于每个分支、真实 APK 或正式环境通过。

游戏接口主要使用 POST，支持 text/plain 中的 JSON，请求体上限 2 MiB。未知路径返回 404；尚未实现的能力可能返回 501。WebSocket、Socket.IO、MQTT over WebSocket、自动全解锁及 API↔房间在线状态/推送仍待完成。

旧客户端的 ID／设备登录仅在显式 `login_mode: legacy_dev` 下开放；该模式不可监听通配或公网地址，并须另行限制局域网访问。默认 `credential` 模式要求 `auth/login` 提交在凭据模式签发的 `recovery_token`，成功后轮换令牌和会话。旧模式使用过的账号不能直接切到凭据模式登录。现有 APK 请求记录尚未证明 0.1.8／0.3.11 会提交此令牌，因此两版客户端联调需使用隔离开发模式。部署限制及四条登录路径的协议证据见[认证与兼容说明](docs/api-authentication.md)。

完整字段、行为差异和验收范围见[接口开发说明](docs/api-development.md)与[路由清单](json/api/legacy-routes.json)。当前工作顺序与接手命令见[开发交接](docs/development-handoff.md)。

## 启动管理面板

管理面板独立构建和启动，默认使用本机回环地址。登录后可查看 API 与房间服务状态，并通过 API 查询和管理好友。配置面板密码及与 API 配对的管理令牌后，从 `services/admin-panel` 执行 `go run ./cmd/admin-panel -config config/admin-panel.json`。配置示例和联调方法见[管理面板说明](services/admin-panel/README.md)。

## 检查与测试

在仓库根目录运行：

```sh
sh scripts/verify/api-server.sh
```

脚本使用 PATH 中的 Go，在 API module 内执行测试、vet 和构建，日志集中写入根 `logs/`，默认构建缓存写入根 `tmp/`。检查脚本禁止自动下载依赖和工具链；缺少 API 依赖时先在该 module 内自行执行 `go mod download`。Linux 房间服务独立检查：`sh scripts/verify/room-server.sh`。管理面板独立检查：`sh scripts/verify/admin-panel.sh`。

手动命令须进入服务目录，根目录没有统一 go.mod：

```sh
cd services/api-server
go test ./...
go vet ./...
go build ./...
```

`room-server` 是另一个 Go module，Windows 原生传输还需要桥接 DLL。各服务分别运行 `go test ./...` 和 `go vet ./...`；原生传输与真实客户端结果需单独记录。当前已验证 0.3.11 单客户端入房和重连，双端同步、自然过图及 0.1.8 全链路仍待验收。证据见[客户端兼容记录](docs/client-compatibility.md)。

## 文档与协作

- [项目约束](AGENTS.md)
- [重构设计](docs/superpowers/specs/2026-11-11-go-rewrite-design.md)
- [开发计划](docs/archive/plans/2026-11-11-go-rewrite.md)
- [API 接口测试手册](docs/api-testing.md)
- [开发交接](docs/development-handoff.md)
- [实时服务说明](services/room-server/README.md)
- [复刻待办与剩余接口](docs/TODO.md)
- [开发过程归档与审查索引](docs/archive/README.md)
- [开发与修复记录](docs/archive/records/TodoList.md)

运行数据、真实配置、证书私钥、日志、抓包和构建产物不提交。示例配置、源码、依赖锁文件及脱敏报告纳入版本控制。旧 LaServer 仅作为兼容参考，不覆盖其未提交改动或真实数据。
