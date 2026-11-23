# Go UDP 房间服务开发交接

更新日期：2026-11-23。本文供接手 `services/room-server` 的开发者使用。旧实现 `../LaServer/services/realtime-server` 只读参考；项目约束以仓库根目录 [AGENTS.md](../AGENTS.md) 和 [Go 重构设计](superpowers/specs/2026-11-11-go-rewrite-design.md) 为准。

## 目标、现状与边界

目标是用独立 Go 进程接管旧实时服务的 ENet/UDP 连接、入房、玩家状态、广播、移动、过图和必要的管理观测。`api-server` 与 `room-server` 各有 Go module、入口、配置和生命周期；两者可以分别启动、停止和部署。实时房间状态只在 `room-server` 内存中维护；持久化账号和业务数据由 API 的 SQLite 管理。禁止跨服务导入 `internal` 包，禁止让实时服务直接读写 API 数据库，也不引入 Redis 或 MySQL。

当前 `services/room-server` **只有 `go.mod` 和 README**，无 `cmd/server/main.go`、Go 源码、可运行入口或 `go.sum`。现有 Go API 的 `/account/find_previous_or_empty` 会返回配置的 `room_host:room_port` 并记录服务器分配；`/account/join_friend_game` 读取好友最近的分配。它们都没有证明 UDP 端口可达、玩家已认证或已入房。2026-11-23 MuMu 0.1.8 客户端已进入游戏，API 日志见房间发现 200，但 UDP 握手、进房、双端同步、自然过图均**未验证**。API 日志曾有房间发现偶发 500 后重试 200，根因未定位，不能归因于 UDP。

## 旧实现阅读顺序

旧代码位于 `../LaServer/services/realtime-server/src/`。先读 [旧服务说明](../../LaServer/services/realtime-server/README.md) 和 [版本适配说明](../../LaServer/services/realtime-server/docs/版本适配说明.md)，再按下表追调用链。旧代码中硬编码、运行开关和历史注释都只是迁移线索，必须用测试和实际客户端复核。

| 顺序 | 旧路径（均相对旧服务 `src/`） | 需要提取的合同 |
| --- | --- | --- |
| 1 | `main.js`、`config/env.js`、`api/httpServer.js` | 启动/退出顺序、UDP 与管理 HTTP 地址、配置校验、部分启动失败清理 |
| 2 | `transport/EnetServer.js` | ENet checksum、channel、可靠/不可靠发送、peer 连接/断开事件、发送与销毁语义 |
| 3 | `protocol/PacketCodec.js`、`PacketIds.js`、`binary/*`、`packets/*` | 包头、字节序、长度限制、包 ID、各包字段和错误处理 |
| 4 | `protocol/VersionProfile.js`、`versionMatrix.js`、`versions/*` | 版本族选择、0.1.8 与 0.3.11 对应 codec 的证据、版本误报时的选择策略 |
| 5 | `session/RoomServer.js`、`ConnectionSession.js`、`ConnectionManager.js`、`PacketDispatcher.js`、`TickLoop.js` | 连接超时、重复连接、调度顺序、断线回收、周期广播 |
| 6 | `services/JoinService.js`、`BroadcastService.js`、`MoveService.js`、`LevelService.js`、`GameMessageService.js` | 进房握手、同房/同关可见性、移动、过图、消息与快照 ACK |
| 7 | `domain/*`、`observability/*`、`protocol/snapshot/*` | 房间/玩家所有权、关卡数据、快照窗口、日志与敏感抓包策略 |

旧 Node 服务依赖 `sky-enet`。`EnetServer.js` 的创建参数含 `checksum: true`、`channelLimit`；默认配置见 `config/env.js`（UDP 端口 25565、2 channel、最大 32 peers、入房超时 6000 ms、最大包 16384 字节）。**这些是旧实现默认值，不是 Go 方案已验证的最终值。** 旧项目还包含 `tests/sky0134-regression.mjs` 与 `protocol/PacketCodec.test.js`；版本说明称一条构建号断言与实现不一致，使用时先核对基线，不把旧测试全绿作为既成事实。

## 协议与客户端证据

- 旧版本说明把 0.6.5–0.12.3、0.13.4–0.21.5、0.22.0–0.34.3 分为 `legacy`、`mqtt`、`modern` 三族。说明中的版本扫描和部分抓包不等于每版都已实机验收。0.1.8 适配包的实际资源/引擎/上报版本可能不同；0.3.11 也曾上报 0.13.4.170388。**不可仅按 APK 文件名或 User-Agent 选择 UDP codec**；先核对实际包头、JoinGame 负载和构建信息。
- 先保留旧协议字节级样本：ENet connect、JoinGame、EnterGame、NetTimePing/Pong、LevelUpdate、PlayerJoined/Left、Move、GameMsg。每个样本记来源客户端、时间、方向、channel、可靠标记、原始长度、脱敏后的十六进制、预期解析结果。抓包只对隔离测试账号开启，日志放本项目 `logs/`，按项目规则不得散放或提交原始标识与会话。
- 旧说明提到不同族在包头、JoinGame/LevelUpdate、NetLevelData 封装以及聊天路径上有差异；部分 `mqtt` 族社交消息走 `/account/ws`。UDP 迁移不能顺手把 MQTT/WebSocket 假定为普通 UDP 消息。
- 若没有可复现字节样本，先做旧服务的只读源码合同和独立 codec 测试；真实客户端兼容状态保持“未验证”。

## 拟建 Go 目录与职责

```
services/room-server/
  cmd/server/main.go          # 独立入口、信号处理
  internal/config/            # UDP/管理监听、限额、超时与地址校验
  internal/app/               # 组装与启动/停止清理
  internal/transport/         # ENet 适配接口与选定实现
  internal/protocol/          # 帧、包、版本 codec、快照
  internal/session/           # peer 与已认证会话、超时
  internal/room/              # 房间、玩家、关卡状态单一所有者
  internal/api/               # 本机管理与状态 HTTP
  config/                     # 无凭据示例配置
```

传输层只负责 ENet 收发和 peer 事件；协议层只做字节编解码；会话层负责 peer 生命周期及认证状态；房间层串行更新玩家/关卡状态并产生广播事件。不得由多个 goroutine 直接改同一房间 map。管理 HTTP 只观察或执行明确授权的操作，不直接暴露原始会话、抓包或密钥。

## 与 API 的接口合同（开始编码前定稿）

| 需求 | 当前事实 | 交接后的设计任务 |
| --- | --- | --- |
| 地址下发 | API 配置 `room_host`/`room_port`，房间发现返回 `uri`、时间与签名；不探测房间存活 | 明确 UDP 监听地址和**客户端可达的公告地址**，测试 API 停机/房间停机时各自仍可启动 |
| 入房身份 | 旧服务存在 JoinGame/会话逻辑；Go 两服务尚无认证协议 | 定义 room→API 的验证请求/响应、超时和失联策略；拒绝未知/过期/被禁账号，不因 API 暂时失联放行；跨进程调用不得直接访问 SQLite |
| 在线/好友房间 | Go API 的在线好友目前不能反映真实 UDP 在线状态；好友进房依赖最近 `server_assignment` | 定义 room 的只读在线/房间查询和 API 调用边界、TTL、断线清理；避免把旧分配当在线事实 |
| 版本选择 | API 的 User-Agent/资源版本与实际 UDP 协议不一定一致 | 定义可信的版本传递或握手识别方式；协议族不明时安全拒绝并记录脱敏诊断 |

此表是**待实现合同**，不是已经存在的端点。正式定义前先写双服务合同测试：正常验证、无效/过期会话、API 不可用、超时、重连、同账号重复连接，以及一个服务停止不终止另一个服务。

## 实施顺序与每阶段交付

1. **技术验证**：隔离比较 Go ENet 候选与旧 `sky-enet`、实际客户端。记录许可证、维护状态、Windows/Linux amd64、CGO/动态库、checksum、2 channel、可靠/不可靠包、分片、重传、断线回调。用最小握手证明兼容后锁定依赖；未证明前不承诺纯 Go 单文件发布。
2. **可独立运行的空房间服务**：配置、端口冲突报错、ENet 监听、本机健康接口、信号退出、部分启动失败清理。独立启动时不要求 API 在线；需要认证的实际入房操作按合同失败关闭。
3. **协议最小闭环**：按实测客户端优先实现包头、JoinGame、EnterGame、Ping/Pong 和玩家加入/离开。以黄金字节样本做编码/解码往返和错误包拒绝；一台客户端真实连接并保持会话，不以 API 房间地址 200 代替。
4. **房间状态与双端同步**：实现玩家上限、重复连接、同房/同关可见性、快照或 GameMsg 的必要子集、移动广播、ACK、断线回收；两个隔离测试账号互相可见且状态不串房。
5. **过图与版本扩展**：实现 LevelUpdate/MoveResult/CancelMove 及自然过图；按实测 0.1.8、0.3.11 分别验证 codec，再扩充其他协议族。每个版本记录首个失败包和行为差异，不以版本矩阵条目作为通过证明。
6. **API 联调与交付**：接入会话验证/在线查询合同，补管理 HTTP 和必要指标；单独构建两服务，写启动、停止、回滚步骤。API 在线但 room 关闭、room 在线但 API 关闭两种状态都要有明确、可观测的结果。

## 验收记录与限制

每阶段留存：Go commit/工作区状态、客户端 APK 文件及 SHA-256、设备/ABI、API 公告地址、UDP 监听与可达地址、配置文件名、端口占用、测试账号脱敏 ID、包样本来源、测试命令、日志时间、实际现象及结论。测试必须区分：`API 下发地址` → `ENet connect` → `JoinGame 被解析并认证` → `进入场景` → `双端可见与同步` → `自然过图`；任何一步失败都不能把后续标通过。

现有 MuMu 0.1.8 APK 使用当时开发机地址 `10.17.43.177:18443`，这是历史测试地址，重跑必须重新核对本机网络和 API 配置。旧服务示例端口 25565 也不能代替当前监听事实。测试日志、抓包和临时 APK 的生命周期依 AGENTS.md 管理；不清理用户产物，不提交凭据或原始抓包。真实客户端测试及资源消耗按用户当前授权范围执行；本文编写未进行 UDP 测试。
