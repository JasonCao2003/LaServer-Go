# Go API 接口测试手册

更新日期：2026-11-23。适用于 `services/api-server` 的 HTTP 接口。测试进度见[路由验证台账](api-route-verification.md)，未完成任务见[测试覆盖计划](superpowers/plans/2026-11-12-api-test-coverage.md)。

完整清单见[201 条固定路径逐路由测试用例](api-test-cases.md)。下文 21 条是需要多个接口配合的游戏场景用例。

## 测试目标与通过口径

固定路径共 201 条（200 条注册 API 路径，加独立的 `GET /healthz`）；这个数量只代表入口，不代表全部协议分支或真实客户端通过。动态 `GET /ugc/...` 单独测试。每条路由分别记录四个状态：**代码接入、自动化合同、运行实例冒烟、客户端验收**。别名入口也要逐一发请求；旧服务本身为占位的功能只标“占位兼容”。

一次接口测试必须核对 HTTP 方法和状态码、响应字段及类型、认证、状态或文件副作用。写入接口还要核对重复请求、失败回滚和重启后读取；涉及多个账号时检查越权访问。不能仅凭 HTTP 200、客户端进入游戏或房间地址下发判定业务通过。WebSocket、Socket.IO、MQTT 和 `room-server` 的 ENet/UDP 不计入 201 条静态 HTTP 路由。

## 环境与数据隔离

1. 进入 `services/api-server`；仓库根目录没有统一的 `go.mod`。优先复用已有 Go 工具链。本机曾使用 Windows Go 1.27.4；WSL 有可用环境时按项目约定优先在 WSL 测试。
2. 从 `config/api-server.example.json` 复制一份本地配置到 `config/`，设定**新的** `database` 路径。运行 `migrate` 前确认目标文件不存在或确实是本项目的测试库；不要指向旧 LaServer 实例、真实账号库或另一轮测试库。
3. 需要测试资源时，将 `game_config_dir` 指向经过确认的旧游戏配置只读目录。需要客户端连接时，设置客户端可达的 `listen`、TLS 证书与包内 API 地址；`room_host`/`room_port` 只是下发地址，不会启动实时服务。
4. 测试文件和数据库放在 `services/api-server/data/`，日志放仓库 `logs/`。配置、数据库、凭据和日志不得提交。使用已签名 APK 覆盖安装时保留应用数据；不得为测试直接清除设备数据。

当前 MuMu 隔离环境的本地配置为 `services/api-server/config/api-server.mumu-0108-20261123.json`，数据库为 `services/api-server/data/mumu-0108-20261123/api.db`。地址 `10.17.43.177:18443` 只对应当时开发机网络；重跑前重新核对 IP、端口、证书和 APK，不沿用过期地址。

## 自动化执行

以下在 `services/api-server` 目录执行，输出按日期保存到仓库 `logs/`。命令会运行测试；仅在本轮测试已获授权时执行。

```powershell
# Windows PowerShell；先确认 go 在 PATH，或使用现有工具链的绝对路径。
go test -json ./... > ..\..\logs\api-test-YYYYMMDD.jsonl 2>&1
go vet ./... > ..\..\logs\api-vet-YYYYMMDD.log 2>&1
go build ./... > ..\..\logs\api-build-YYYYMMDD.log 2>&1
```

```sh
# Linux/WSL；在仓库根目录执行现有综合检查脚本。
sh scripts/verify/api-server.sh
```

检查进程退出码和日志中的失败事件。`go test -json` 的 `pass` 事件包含包和子测试，不能直接当作通过的路由数。定向重跑使用 `go test ./internal/handler -run '^Test名称$' -count=1`；修改业务逻辑后再跑完整 `go test ./...`。`go vet` 与 `go build` 是静态检查，不代替接口请求。

已有自动化覆盖：A 批基础、资源与状态，B 批交易和任务闭环，C 批好友、消息、内容、留影与 UGC，D 批舞台与管理。B 批尚缺逐入口合同，所有批次的逐分支矩阵仍未完成。具体测试名可在 `internal/handler/batch_{a,b,c,d}*_test.go` 和 `json_compat_test.go` 查询。

## 运行实例冒烟

先在**隔离**数据库执行迁移，再启动 API；同一测试端口只能有一个已确认的服务实例。

```powershell
cd services\api-server
Copy-Item config/api-server.example.json config/api-server.test.json
# 编辑 config/api-server.test.json：使用新的 database 路径并按测试需求设置 TLS、资源目录和监听地址。
go run ./cmd/server migrate --config config/api-server.test.json
go run ./cmd/server --config config/api-server.test.json
```

从另一个终端检查健康状态，再运行现有冒烟脚本：

```powershell
Invoke-WebRequest https://127.0.0.1:18443/healthz -SkipCertificateCheck
python scripts/verify/local_smoke.py https://127.0.0.1:18443
```

上述 Python 脚本从**仓库根目录**运行，使用测试环境 TLS 跳过证书验证，并会创建测试账号、写入测试数据。仅对隔离开发实例运行；其结果是有限的本地链路检查，不是 201 条路由完整验收。若配置监听 HTTP 或其他端口，按实际地址调整命令。记录脚本退出状态、通过/失败项、服务访问日志及数据库前后状态；不得把失败后的重试 200 隐去。

## 接口测试矩阵

### 用例使用说明

下面的字段来自当前 Go 实现及现有合同测试。`A` 表示当前登录账号的 `user`（字符串）和 `session`（字符串）；除注册、登录和启动配置外，请求体均带这两个字段。所有示例均为 `POST`、JSON 请求体。表中只列**关键**响应字段，实际响应可含版本兼容字段；数值、具体列表和资源内容以隔离测试库及配置为准。先在游戏内操作，再按同一时间窗口关联 API 访问日志、响应体和数据库前后值。尚未取得真实客户端证据的用例状态均为**待执行**，以下“预期”不是已通过结论。

最后一列按 `logs/mumu-0108-api-fixed-20261123.log` 中 `method=POST path=... status=200` 的记录填写：**✅ 后面逐一列出有 200 记录的路径，— 表示没有找到**。同一用例包含多个接口时，仅勾选确实出现的接口；通配路径不勾选。200 只证明请求到达并得到该状态码，不证明返回内容正确或游戏操作完成；此前出现过 500 后重试 200 的接口仍需排查。

| ID / 接口名称 | 传入参数（JSON 字段） | 传回参数（关键字段） | 游戏中如何触发 | 正确结果与核对点 | MuMu 0.1.8 HTTP 200 记录 |
| --- | --- | --- | --- | --- | --- |
| A01 创建旧版账号 `/account/create` | `device_name` 字符串、`language` 字符串；新版签名字段按实际客户端抓取 | `user`、`device`、`key`、`session`、`resource_list` | 全新安装或新测试资料首次进入登录流程 | 返回 200；账号 ID 和会话非空，后续资源请求使用同一账号；重复启动不能无意创建第二账号。缺 `device_name` 返回 400。 | — |
| A02 旧版登录 `/account/login` | `device`、`key`、`language`；也支持 `user` 分支 | `user`、`session`、`resource_list`，旧客户端还需版本兼容字段 | 用已有测试账号重新打开游戏并登录 | 返回 200，账号 ID 不变、会话轮换；旧会话再请求受保护接口应失败。未知账号按当前分支返回 404。 | — |
| A03 新版注册与登录 `/account/auth/create`、`/account/auth/login` | 注册：`type`、`device_name`、`device_token`、`production`、`tos_version`、`device_key`、`sig_ts`、`sig`、`hashes`；登录参数以客户端实际请求记录 | `authinfo.user/device/key/recovery_token`、`session`、`resource_list`；登录还含 `websocket_url`、`websocket_auth` | 在使用新版认证流程的客户端首次注册、重登 | 成功返回 200 并完成资源加载；缺签名必需字段返回 400；未授权/封禁账号不能进入游戏。0.1.8 实际走哪条入口须以日志确认。 | ✅ `/account/auth/login` |
| A04 存档读取与同步 `/account/userdata`、`/account/sync_user_data` | `user`、`session`；写入另带 `data` 字符串、`time` 数值、`version` 数值 | `userdata.data/time/version` | 改变装扮或位置，触发保存；完全关闭后重开 | 写入后读取内容和时间一致；重新启动服务后仍一致；空 `data` 不能清空既有存档；另一账号会话读取返回 401。 | ✅ `/account/userdata` |
| A05 货币 `/account/get_currency` | A | `currency` 对象，含 `candles`、`heart`、`wax`、`season_candle` 等数值 | 打开货币面板，获取蜡烛或爱心后再打开 | 余额与变动一致，不能为负；同日重复请求不会重复发放每日 VIP 奖励；重登后保持。 | ✅ `/account/get_currency` |
| A06 解锁列表 `/account/get_unlocks` | A | `unlocks` 数组 | 登录资源同步、打开表情或先祖解锁界面 | 返回 200，列表能被客户端解析，不出现 `Unrecognized JSON escape character`；购买/解锁后再次读取包含相应记录。需检查原始 JSON 中旧客户端不兼容的 `\\uXXXX`。 | ✅ `/account/get_unlocks` |
| A07 解锁确认 `/account/ack_unlock` | A、`name` 字符串 | `result` | 获得新解锁并关闭提示 | 确认后提示不反复出现；无效账号会话不得确认其他账号解锁。注意此旧接口可能以响应内 `result` 表示失败，不能只看 HTTP 200。 | — |
| A08 检查点读取与写入 `/account/get_checkpoints`、`/account/set_checkpoint` | A；写入加 `level_id`、`level_hash`、`bst_id`、`states` | `checkpoints` 数组，元素含 `level_id`、`bst_id`、`arc_order`、`is_furthest` | 进入一张地图，激活检查点，再进入下一张地图并重登 | 同地图更新不能产生重复记录；最远进度指向正确地图；重登后记录保持。缺写入字段或无效 `level_id` 返回 400。 | ✅ `/account/get_checkpoints`；✅ `/account/set_checkpoint` |
| A09 房间发现 `/account/find_previous_or_empty` | A | `uri`、`move_ts`、`signature`、`sig`、`ps_only`、`platform_filter` | 客户端尝试进入场景或切换房间 | 返回 200 且 `uri` 与测试配置的实时服地址一致；仅表示地址下发，UDP 连接须单独验证。已观察过偶发 500，出现一次即记失败并保留请求日志。 | ✅ `/account/find_previous_or_empty` |
| A10 收藏品 `/account/get_collectibles`、`/account/collect_collectible` | 读取：A；收集：A、`name` 字符串 | `collectibles` | 拾取一个可收集物，退出后重新进入 | 收集前后列表变化与游戏提示一致；重复拾取不重复计数或奖励，另一账号不受影响。 | ✅ `/account/get_collectibles` |
| A11 魔法库存 `/account/consumable/get_consumables1`、`/account/consumable/consume_item1` | 读取：A；使用：A、`consumable_id` 数值、`consumable_count` 数值 | `get_consumables`、`consume_item` | 在游戏中打开魔法栏并使用一个已拥有魔法 | 库存减去使用数量，效果出现；库存不足不得继续扣减或产生效果；另一账号库存不变。 | ✅ `/account/consumable/get_consumables1` |
| A12 光翼状态 `/account/wing_buffs/get`、`/account/wing_buffs/collect`、`/account/wing_buffs/drop` | A；收集/掉落加 `name` 字符串 | `wing_buffs`、`update_wing_buffs` | 收集光翼、在可复现条件下掉落光翼并重登 | 读回数量和状态与游戏一致；重复收集/掉落不应产生额外光翼；其他账号数据隔离。 | ✅ `/account/wing_buffs/get` |
| A13 商店目录 `/account/get_shop` | A | `get_shop`（商品列表，含 `name`、`unlockid`、`unlock_cost`、`currency_type` 等） | 打开先祖商店或物品节点 | 商品显示、消耗货币和解锁依赖与配置一致；缺资源配置时按实际错误处理，不能把空白界面当通过。 | ✅ `/account/get_shop` |
| B01 购买解锁 `/account/purchase_unlock` | A、`unlock_id` 数值、`cost` 数值 | `result` | 准备足够蜡烛，在商店购买一个未解锁节点 | 成功后只扣一次货币并新增解锁；重复提交返回 `already` 且不二次扣款；余额不足不能获得解锁。 | — |
| B02 地图拾取 `/account/collect_pickup_batch` | A、`level_id`、`pickup_ids` 数组、`global_pickup_ids` 数组 | 响应关键字段按客户端本次请求记录，重点核对货币/拾取状态 | 在地图拾取烛火，返回大厅后重进地图 | 首次拾取增加对应烛火或季节奖励；重复上报同一 ID 不重复发放；重新登录后状态保持。 | — |
| B03 季节任务 `/account/get_season_quests`、`/account/activate_season_quest`、`/account/claim_season_quest_reward` | A；激活/领取加 `quest_id`、`in_season` | 任务列表、领取结果；具体字段按本次客户端响应记录 | 打开季节任务、激活、完成目标并领取奖励 | 未完成不能领取（当前测试预期 409）；完成后奖励只发一次，重复领取不增加余额；跨账号不可领。 | — |
| B04 支付回执 `/account/commerce/receipt` | A、`platform`、`product_id`、`transaction_id`，真实平台另带回执数据 | `result`、`transactions` | 仅在隔离配置和模拟交易中做接口测试；真实购买需另行授权 | 假平台默认拒绝；未配置真实票据验证返回 501，不应发货；隔离模式的同交易 ID 重放不重复发货，跨账号重放冲突。**不能以游戏内真实支付做本用例。** | — |
| C01 邀请好友 `/account/create_invite`、`/account/check_invite`、`/account/accept_invite`、`/account/get_invites` | 创建：A、`invite_nickname`；检查/接受：B 的 `user/session`、`token_id` | `invite_token_id`、`invite_exists`、`invite_accepted`、`invites` | 准备测试账号 A/B，A 创建邀请，B 输入邀请并接受，双方重登 | B 能接受且双方好友关系可见；同邀请码重复接受失败；非创建者不能删除邀请；各账号请求需分别关联日志。 | ✅ `/account/get_invites` |
| C02 好友列表 `/account/get_friends`、`/account/get_online_friends` | A | 好友列表及在线列表；具体对象字段以本次响应记录 | 完成 C01 后打开星盘或好友界面，B 上线和下线 | 双方关系可见；在线状态变化按当前实现反映，不能仅凭空列表或 HTTP 200 判通过。实时在线依赖需单独记录。 | ✅ `/account/get_friends`；✅ `/account/get_online_friends` |
| C03 消息 `/service/message/api/v2/get_all`、`/service/message/api/v1/get_all`、`/service/message/api/v1/get` | A，分页/消息标识按实际请求记录 | 消息数组、未读状态等按实际响应记录 | A/B 互发消息或赠礼，重新打开收件箱 | 发送者和接收者一致，仅目标账号可见，重登后仍可读取；具体字段必须先由日志确认，未触发时标未执行。 | ✅ `/service/message/api/v1/get_all` |
| D01 舞台与管理接口 | 参见路由台账中的 `/service/stage/api/v1/*`、`/admin/api/*`，字段按相应合同测试记录 | 舞台对象、管理结果和错误响应 | 仅在有对应游戏入口或隔离管理端时操作 | 核对所有权、顺序、权限和持久化；没有可操作 UI 的路径标“仅接口测试”，不虚构游戏步骤。 | — |

**每条用例还需执行失败分支**：无效 `session`、缺字段、账号 A 的 `user` 搭配账号 B 的 `session`。写入类再测重复请求、重登读回；支付、任务、赠礼等再测事务回滚。HTTP 状态可能因旧协议而为 200 且业务字段表示失败，验收时同时看响应体和数据库。填写结果时记录具体请求体（脱敏）、完整响应字段结构、服务日志时间和游戏界面文字；上述表格不代替实际取证。

| 范围 | 主要接口与场景 | 必查断言 |
| --- | --- | --- |
| A：身份、资源、状态 | create/login、userdata、解锁、魔法、资源定义 | 正常/缺字段/无效或过期会话；0.1.8/0.3.11 版本字段；跨账号读取；配置缺失返回明确错误；JSON 原始转义兼容 |
| B：货币、奖励、任务 | purchase、receipt、lootboxes、quest、season | 余额不足、重复交易 ID、并发领取、日界、重启幂等、事务回滚；真实支付凭据未经验证不得发货 |
| C：关系、消息、内容 | friends、message、gift、geonote、feed、recording、UGC | 双账号可见性与所有权、重复礼物、过期、分页、上传失败、分块续传、Range 与路径穿越；跨实例聊天另记限制 |
| D：舞台与管理 | stage、reservation、`/admin/api/*` | 舞台所有权和顺序、存储故障；管理 token/来源限制、双向好友状态、审计事务；预约和公寓旧占位单独标注 |

对每个具体路径在[路由验证台账](api-route-verification.md)记录：方法、版本、请求字段、成功/失败响应关键字段、认证、数据库/文件前后值、自动化测试名、冒烟日志、客户端证据及差异。最少覆盖正确请求、缺字段、失效会话、跨账号访问；写入再测重复、回滚和重启。`GET /ugc/...` 另测完整读取、有效/无效 Range、越界和 `..` 路径。管理路径分别测 GET 与 POST。

## 0.1.8 / 0.3.11 客户端验收

1. 记录 APK 文件名和 SHA-256、包名、实际版本与客户端上报版本、设备 ABI、ADB 目标、开发机 IP、API 配置和时间。更新地址后要重新签名并验证 APK；原包保留。
2. 按冷启动 → TLS → 创建或登录 → `resource_list` 拉取 → 进入游戏 → 存档写入与重启读取 → 任务/好友/内容 → 房间发现的顺序记录首个失败点。对每一步关联服务端请求路径、状态码和客户端结果。
3. 没有 `room-server` 时只能验证房间地址下发，不能记录 UDP 进房、双端同步或自然过图通过。界面状态优先由测试者文字告知；确需截图时先取得同意，集中放 `pic/`。

当前证据：2026-11-23 MuMu 的 0.1.8 网络地址测试包已成功创建账号、登录并进入游戏；Go 服务记录 136 次 POST、47 个唯一路径。`/account/get_unlocks` 的 `\\u` 转义曾导致客户端崩溃，修复后通过完整 Go 测试并成功进入游戏。相同运行日志有 7 次 HTTP 500，涉及世界任务、邀请、房间发现和状态解锁，均见随后重试 200；**根因未定位，这些路径不能标为无错误通过**。0.3.11 本轮未重测，存档重启恢复和 UDP 未验。详情见[客户端兼容矩阵](client-compatibility.md)和 `logs/mumu-0108-api-fixed-20261123.log`。

## 失败记录与结果模板

失败先保留最小请求条件、服务端状态码、错误响应形状、数据库前后值和客户端首个可见现象；检查现有日志后再决定是否加脱敏诊断。不要把真实 session、设备标识、支付凭据或完整存档写入文档、Git 或普通日志。现有 Go 日志把未分类错误写作 `internal`，遇到 500 时需另定位实际错误原因；不能直接归因于 SQLite 锁。

```text
测试 ID / 日期：
环境：Go commit、配置文件名、测试库路径、APK SHA-256、设备/ABI
接口：HTTP 方法 + 路径、测试账号 A/B（脱敏）
场景：正常 / 缺字段 / 无效会话 / 越权 / 重复 / 并发 / 故障注入
预期：状态码、关键字段、状态变化
实际：状态码、关键字段、状态变化、客户端界面
证据：测试名、logs/ 文件及时间、数据库核对结果
结论：通过 / 失败 / 占位兼容 / 未执行；缺陷或差异链接
```

只有自动化、运行实例和客户端对应场景都有可追溯证据，且没有未解释的 500 或数据不一致时，才标记该场景通过。未执行项保持“未执行”。
