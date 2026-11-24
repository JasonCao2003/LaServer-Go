# LaServer-Go 开发交接

更新日期：2026-11-24。本文面向接手 Go 版 API 与实时房间服务的开发者。当前目标是让既有客户端逐步完成游戏与联机闭环；**不要求完整搬迁旧 LaServer 的账号和历史数据**。旧服务仍作为只读协议与行为参考。

## 项目边界与当前进度

| 部分 | 已有代码和证据 | 尚未完成 |
| --- | --- | --- |
| `services/api-server` | 独立 Go module；201 条静态 HTTP 路由有入口；SQLite schema v18；默认凭据登录与隔离 `legacy_dev`；任务、交易、消息、社交和留影有自动化用例；Task 4 增加只读预检和显式导入工具 | 两版 APK 的完整登录与资源合同；在线状态、聊天/管理推送和房间实时状态集成；旧 WebSocket、Socket.IO、MQTT over WebSocket；正式部署验收 |
| `services/room-server` | 独立 Go module；Windows amd64 / Linux cgo 入口及 SkyENet 桥接（Linux 已通过 Go/ABI 测试，真实传输待验收）；房间入场凭据向 API 验证；`284/542` 协议的部分消息和快照；0.3.11 隔离客户端单人入房、重连已记录 | 双端同步、自然过图、多房间流转、其他协议族；Linux 原生库构建与真实联机验收；完整客户端回归 |

两个服务各自启动、停止和构建。API 拥有 SQLite，房间服务拥有内存房间状态；不要跨 module 导入 `internal` 包或让房间服务直接读写 API 数据库。API 可以单独提供 HTTP 功能，房间服务也能单独监听；客户端**实际入房**需要 API 的会话验证。项目约束见[根 AGENTS.md](../AGENTS.md)，两服务运行细节见 [README](../README.md) 和 [room-server README](../services/room-server/README.md)。

## 旧数据处理决定

新实例可以从空 SQLite 初始化，旧账号、旧存档和历史内容默认不保留。不要把“旧库完整导入”作为 Task 5、房间服务开发或新实例技术验收的前置条件。若具体账号或存档需要保留，再明确范围、获取一致性源库快照，并先运行只读预检；只迁移经核对的必要数据。旧库和新库的文件不能直接互换。

现有旧实例预检显示大量孤儿行；显式导入报告为 `partial=true`，仅 4,102 行进入隔离新库，另有 81,010 行未导入，动态附件和配置旁路文件也未完整迁移。导入器默认拒绝部分结果；`-allow-partial` 是审阅报告后的显式选项，**不表示旧账号可完整恢复**。本轮没有修改旧实例。行数、字段映射、命令和限制见 [Task 4 持久化报告](archive/records/api-task4-persistence.md)。

## 接手后的最短启动与核对路径

1. 在仓库根目录先读 `AGENTS.md`、本文件、[API 下一阶段计划](superpowers/plans/2026-11-13-api-server-next-stage.md)及[客户端兼容记录](client-compatibility.md)。根目录没有统一 `go.mod`；Go 命令要进入各自服务目录。
2. API：在 `services/api-server` 复制 `config/api-server.example.json` 为未跟踪的本地配置，检查 `database`、`game_config_dir`、`listen`、TLS 和 `login_mode`；执行 `go run ./cmd/server migrate --config config/api-server.json`，再执行 `go run ./cmd/server --config config/api-server.json`。新库可为空。`legacy_dev` 仅用于隔离旧 APK 测试，不能监听公网。
3. 房间：Windows amd64 或 Linux cgo 按 [原生桥接说明](../services/room-server/README.md)构建 DLL 或 .so，再配置 `room_admission_key`、`room_admission_bearer`、API 验证地址、UDP 监听与客户端可达的公告地址。不要把真实凭据提交。桥接库、实际客户端联机均不能以普通 Go 编译成功代替。Linux 当前环境与待补依赖见 [Linux 开发说明](linux-development.md)。
4. 验证：在两个服务目录分别运行 `go test ./...`、`go vet ./...`。API 的 Task 4 全量测试与 vet 记录见 `logs/api-task4-full-test-final-20261124.log`、`logs/api-task4-vet-final-20261124.log`；Windows 无创建符号链接权限，相关导入路径专项用例被跳过。0.3.11 单客户端入房与重连证据见[客户端兼容记录](client-compatibility.md)。新的测试应记录命令、环境、结果和日志路径，不把 HTTP 200 或房间地址下发等同于联机通过。

## 下一阶段工作与完成条件

| 顺序 | 工作 | 可核对的完成条件 |
| --- | --- | --- |
| 1 | Task 5：定义 API↔房间服务的在线状态、好友进房、预约、公寓状态及推送合同 | 明确认证、超时、失联响应；两个服务仍能分别启动，API 不把离线玩家报在线 |
| 2 | Task 5：按客户端实际使用情况盘点并实现 WebSocket、Socket.IO、MQTT over WebSocket | 分协议验证握手、认证、事件/主题和消息封装；不把三者统一当普通 WebSocket |
| 3 | room-server：完成同房双端同步、断线回收、自然过图、多房间隔离及需要的协议族 | 两个隔离账号互相可见并同步，过图后状态正确，跨房间状态不串；逐版本记录首个失败包 |
| 4 | Task 6：0.1.8、0.3.11 客户端全链路和 Windows/Linux 部署验收 | 分别记录冷启动、登录、资源、存档重启、任务、好友、内容、入房、双端同步和过图；未支持平台如实标记 |
| 5 | 交付准备 | 更新 README、差异清单、配置示例、备份/回退步骤；正式实例切换另行安排，旧数据是否保留按本文件决定 |

API 的任务边界见[阶段计划](superpowers/plans/2026-11-13-api-server-next-stage.md)，逐路由证据见[路由台账](api-route-verification.md)。旧[UDP 交接文档](archive/records/room-server-handoff.md)记录的是 room-server 开发前的历史基线，阅读时以本文件和当前 room-server README 为准。

## 已知限制

- 两版原版 APK 是否能提交默认 `credential` 模式所需凭据仍无完整证据；隔离测试使用 `legacy_dev`。不要把用户 ID 登录当作公网身份校验。
- 普通聊天消息保存在单个 API 进程的内存队列，重启丢失，多实例之间也不共享；礼物等持久状态在 SQLite。
- 真实支付凭据未实现验证和发货；本地模拟收据只在显式开启时可用。
- 旧数据的部分导入不能证明客户端兼容或正式切换可行。当前开发不以完整旧数据迁移作为验收条件。
