# 游戏 API 迁移进度与运行

## 当前范围

本阶段已有可运行的 api-server，使用 Go net/http 和 modernc.org/sqlite。不是完整旧服务替代品：静态扫描找到 business 下 201 条 add_url_rule 注册，当前已全部接入 Go handler（`/ugc` 为前缀路由）。本轮新增接口已完成 Go 静态构建检查，测试尚未执行。扫描不包含装饰器路由、动态注册、Socket.IO 或 MQTT/WebSocket 协议，不能把这份数量视为整个旧服务的接口总数。

| 业务 | 已实现路径 |
| --- | --- |
| 账号创建 | /account/create、/account/auth/create |
| 登录 | /account/login、/account/auth/login |
| 存档 | /account/userdata、/account/sync_user_data |
| 货币、排名 | /account/get_currency、/account/get_rank |
| 房间发现 | /account/find_previous_or_empty |
| 神坛检查点 | /account/get_checkpoints、/account/set_checkpoint |
| 启动配置 | /account/get_latest_build_version、/account/device_capabilities/lookup、/account/get_vars |
| 账号资源 | /account/get_collectibles、/account/get_achievements、/account/get_app_badge_number |
| 定义数据 | /account/get_achievement_defs、/account/get_map_defs |
| 玩家定义与状态 | /account/get_forge_rates、/account/get_outfit、/account/get_achievement_stats、/account/get_achievement_stats_tracking、/account/get_questionnaires、/account/get_event_currency_defs、/account/get_relationship_abilities、/account/get_invites、/account/get_questionnaire_groups、/account/quest/get_quest_defs、/account/iaplist、邀请码 create/check/accept/delete |
| 公告与支持 | /account/get_motd、/account/support/has_reply |
| 第一批资源与认证 | /account/get_surveys、/account/get_event_schedule、/account/get_spirit_shops、/account/get_currency_defs、/account/trust/get_trust_tier_defs、/account/get_video_defs、/account/get_radiance_spawn_defs、/account/get_level_pickups、/account/trust/get_trust_group_defs、/account/remote_config/upload、/account/trust/get_trust_activity_defs、/account/get_generic_shop_item_tracking_info、/service/auth/api/v1/token、/service/infraction/api/v1/get_infractions_status_decorated、/starwatch/auth/api/v1/auth |
| 第五批好友与关系 | /account/get_blocked_friends、set_friend_name、set_friend_block、get_external_friends、serendipity/get、star/get、join_friend_game、set_outfit、set_joinable、get_remote_outfit、remove_relationship_hint；/service/relationship/api/v1/get_all_friend_gist、get_follower_count、update_friend_constellation_pages、add_follow、set_follow_name、delete_follow；/service/trust/api/v1/get、get_capped |
| 第六批消息与赠礼 | /account/get_pending_messages、chat/send、send、send_message、translate、set_device_token、give_candle、claim_message_gift、claim_message_gifts；/service/message/api/v2/get_all、api/v1/get_all、api/v1/get、api/v1/mark_received、api/v1/mark_seen；/service/relationship/api/v1/free_gifts/get_pending、send、claim |
| 第七批地理笔记 | /account/get_user_geonotes、geonote/get_user_geonotes、geonote/request_geonotes_for_n_friends、geonote/create_geonote2、create_geonote2、geonote/create_bug_report_geonote、create_bug_report_geonote、geonote/social_geonote2、social_geonote2、geonote/delete_geonote、delete_geonote、geonote/delete_bug_report_geonote、delete_bug_report_geonote、geonote/edit_message_geonote、edit_message_geonote、geonote/request_n_bug_report_geonotes、request_n_bug_report_geonotes |
| 第八批社交与留影 | /account/social_feed/create、get_user_feeds、get_curated_feeds、get_reactions、like、delete；/service/recording/api/v1/create、get、v2/create、v2/update、v2/get；GET /ugc/*；/service/social-feed/api/v2/get_curated_feeds、v1/get_user_feeds、v1/get_by_ids、v1/create、v1/hide、v1/like、v1/super_like、v1/delete、v1/edit、v1/get_reactions、v2/comments、v1/report、v2/get_penpals、v2/get_penpal_by_id |
| 第九批舞台与预约 | /service/reservation/api/v1/reserve；/service/stage/api/v1/get、set、friends_in_apartments、leave_apartment、update_apartment_status |
| 第十批管理好友 | /admin/api/friends_list、friends_set、friends_clear（POST/GET） |

所有游戏路径使用 POST，支持 Content-Type: text/plain 的 JSON，最大请求体 2 MiB；GET /healthz 为健康检查。JSON 错误不返回原始请求、会话或数据库细节。已注册旧路径未实现返回 501，未知路径返回 404。不能因为登录接口返回 200 就认为真实客户端已能进游戏：resource_list 的其余资源接口仍需迁移，WebSocket 和实时服务未实现。

## 环境与启动

项目 Go 最低版本为 1.26.0，驱动锁定 modernc.org/sqlite v1.62.0。原 Ubuntu WSL 工具链位于 `~/.local/opt/go1.27.4/`；本轮环境未提供 WSL 发行版，因此复用此前安装授权，在 Windows 用户目录放置便携 Go 1.27.4：`C:\Users\Jason\.local\opt\go1.27.4\go\bin`，下载缓存为 `C:\Users\Jason\.cache\laserver-go\go1.27.4.windows-amd64.zip`。未修改系统 PATH；本轮静态构建由该工具链完成。

以下在 services/api-server/ 目录、Ubuntu WSL 执行。只为新开发实例操作，不指向旧数据库：

```sh
export PATH="$HOME/.local/opt/go1.27.4/bin:$PATH"
cp config/api-server.example.json config/api-server.json
# 修改 room_host 为客户端可达 IPv4，并按需设置 TLS 和自动授权。
go run ./cmd/server migrate --config config/api-server.json
go run ./cmd/server --config config/api-server.json
```

相对数据库和证书路径相对于配置文件目录解析。migrate 显式创建开发数据库目录，可初始化空数据库或将本项目 schema v1–v16 升级到 v17，拒绝无法识别的旧实例数据库。升级在事务中创建新增表并保留账号，不自动导入旧项目数据。正常启动要求 schema v17，不自动建表；已是 v17 时重跑 migrate 不会重复建表。当前 schema 只覆盖已迁移功能所需表，不能直接导入、复制或替换旧实例数据；旧 instance 和 legacy 数据必须分别迁移并验收。

示例仅监听 127.0.0.1，默认不自动授权、不自动补建丢失账号，未授权登录返回 406。在隔离开发实例中可将 auto_authorize 设为 true 创建可登录测试账号；已有账号不会因此被批量授权。TLS 证书和私钥必须成对配置；旧客户端 TLS 互通尚未验证。API 独立启动，不检查实时服务是否在线，room_host/room_port 只是下发配置，不代表 ENet 已实现或可达。

## 资源配置与 schema v17

第四批增加玩家任务、季节任务和领取状态、世界任务冷却及每日拾取记录。任务创建与统计上报通过 SQLite 保存进度，领奖标记与货币变更在一个事务中提交；只有当日下发且已完成的任务可以领奖，客户端直接提交完成状态会被拒绝。旧服务的每日任务池会在重启或跨日时随机刷新并写回配置；Go 当前只读取现有配置池，跨日自动刷新仍待对齐；任务及领取标记统一按太平洋日界，未来日期排期使用任务目录按日期确定性生成。旧存量任务及领取标记尚无导入工具，不能直接切换真实账号。

`game_config_dir` 是只读旧游戏配置目录，路径相对于本服务配置文件解析。例如当前并列工作区可设置为 `../../../../LaServer/services/game-api/config`，复用既有配置，不复制数据库或静态响应资源。收藏品需要 collectibles.json，地图定义需要 map.json，任务定义需要 quest.json，IAP 清单需要 iaplist.json；未配置、文件缺失或格式错误返回 503。第一批的兑换树优先读取配置目录同级 `resources/static_responses/account__get_spirit_shops.json`，缺失时读 `spirit_shops.json`；活动日程按客户端版本读 `event_schedule_low.json` 或 `event_schedule_high.json`。成就、成就定义和角标从 Go 开发库读取，不会自动导入旧数据库记录。

第一批新增的调查、视频、信任、违规与关卡拾取表在 schema v8 建立，不自动导入旧数据；空表返回对应的空列表或空状态。`resource_auth_token` 必须配置有效资源 CDN token，`starwatch_jwt_secret` 和 `starwatch_broker_url` 必须由部署方本地配置；缺失时认证接口返回 503。旧服务写死的 token 和签名密钥未复制进公开仓库。`remote_config/upload` 沿用旧实现：校验会话和封禁后确认请求，旧代码本身不保存上传内容。Starwatch 与外部 MQTT broker 的实际互通尚未验证。

活动日程已按版本选择低/高资源，处理每日重置、当前赛季、活动关闭与烛火分组。自动烛火组采用按日期稳定选取，尚未核对旧 Python 随机序列的逐项一致性；客户端版本纠偏配置也待复核。兑换树按客户端版本裁成完整先祖子树并支持分页，配置缺失时明确报错。上述边界在客户端验收前均保留待验证状态。

schema v12 增加第五批所需的好友关系表：外部好友、机缘匹配、星标定义与关联、服务器分配、在线状态、关注关系和船友关注（boat_follow）。`/account/find_previous_or_empty` 现在会把本次下发的房间写入 `server_assignment`，`join_friend_game` 按旧逻辑复制好友最近一次分配；room-server 尚未实现，返回的地址不代表 ENet 已可达。外部好友、机缘匹配与星标表初始为空，需导入旧数据或由后续写入路径补充；`/account/get_external_friends`、`/account/serendipity/get` 在空表上返回空列表是预期行为。`/account/star/get` 对 0.26.0 及以上版本返回内置静态星标定义，低版本读取数据库。`/service/trust/api/v1/get_capped` 保留旧固定响应，但不再像旧服务那样为任意 user id 自动建号。

schema v13 增加消息与赠礼表：`message` 保存邮件已读时间，`free_gift` 保存免费礼物及其 30 天有效期。邮件内容不写入数据库，按 `game_config_dir/mailbox.json` 读取；文件缺失或格式错误时返回空邮件列表。聊天消息只存在于 api-server 进程内队列（每人最多 1000 条，重启丢失），由 `/account/get_pending_messages` 轮询排空；WebSocket 未实现，因此没有实时推送。免费礼物支持扣费、幂等领取和领取后角标计数，重复领取不重复发放。旧 `/account/chat/send` 的调试斜杠命令（改身高、货币、光翼）尚未迁移，会按普通聊天处理；`give_candle` 和礼物接口也不再自动创建占位账号，目标必须已存在。

schema v14 增加 `geonote` 表，地理笔记改由 SQLite 持久化；旧服务的 `instance/geonotes.json` 旁路文件尚未提供导入工具，切换真实实例前需单独迁移。读取按 `bug` 分区和 `level_id` 过滤，返回同关卡全部留言；`request_geonotes_for_n_friends` 按旧实现不查询好友关系。点赞、可见、举报和表情是留言级全局状态，不按查看者区分；删除和编辑校验作者，删除不存在的留言仍返回 `result: ok`。留言正文截断到 500 个字符，不设过期时间。

schema v15 增加社交动态表：`social_feed_post`、`social_feed_like`、`social_feed_extra`。动态附件数据从旧服务的 `instance/feed_extra/*.json` 旁路文件改为 SQLite，留影仍按旧布局写入 `instance_dir/recordings`（`<id>.json` 元数据 + v2 的 `<id>.ndjson` 分块），`GET /ugc/...` 支持字节 Range。旧 `feed_extra` JSON 与 recordings 文件尚未提供导入工具，切换真实实例前需单独迁移。账号层删除校验作者，服务层删除按旧实现不校验所有权；评论、举报和笔友接口保持旧服务的空响应，点赞计数仍是全局计数。

schema v16 增加 `stage_state` 表，按 `stage_id + owner_id` 保存共享空间摆放与 sequence；旧服务的 `instance/stage/*.json` 旁路文件尚未提供导入工具。`stage/get` 优先按目标 `user_id` 读取，`user_id` 为全零时回退到调用者；`stage/set` 以 `user` 为 owner，并把空 props 列表视为清空。预约接口返回配置中的 `room_host/room_port`，旧服务写死 `127.0.0.1:8443`；公寓好友发现、离开和状态更新保持旧服务的空列表/成功回执，不写持久化状态。

schema v17 增加 `admin_audit` 表，记录管理好友修改的来源 IP、动作、目标账号和结果。`/admin/api/*` 默认只允许回环地址访问；配置 `admin_token` 后必须提供匹配的 `X-Admin-Token` 或请求体 `admin_token`。保留旧管理路径的 GET/POST：GET 从查询参数读取 `user`/`friend` 等字段，POST 读取 JSON。删除写入 `friendship.status='removed'` 墓碑并置 `local_soft_deleted`，添加/恢复强制双向 active；`pushed` 固定为 false，因为尚未实现 WebSocket 推送。

schema v1–v7 开发库升级前保留备份，然后显式执行 migrate 命令；本轮没有对任何真实实例执行升级。迁移完成后应使用 v17 代码；v16 及更早版本二进制不识别新 schema，不能直接作为回退程序。

## 已知协议差异与边界

- 登录沿用旧服务可仅凭 user ID 换取会话的行为，尚未实现可验证的设备签名或恢复凭据认证。该行为不是安全认证，当前版本仅供隔离本地开发，不应据此开放公网。
- userdata 与 sync_user_data 均要求当前有效会话，并校验授权和封禁；旧 userdata 无严格校验、sync 仅警告会话不匹配的路径没有保留。
- 默认关闭丢失账号补建，需要显式设置 recreate_missing_account；尚未迁移旧面板的自动授权开始时间、全解锁和账号默认数据配置，初始货币改为显式 default_currency 配置。
- 季卡每日季蜡按 America/Los_Angeles 日期在 SQLite 事务中与领取标记一起提交，不再依赖旧旁路 JSON；重复和并发请求同日只发一次。旧每日标记尚未导入，不可直接切换真实账号。
- 货币表新增 user_id 唯一约束，避免新库同账号重复记录；旧重复行需在迁移时单独处理。
- 检查点采用旧 checkpoint_defs 的首条关卡顺序，保留数值 ID、更新已有关卡而不重复插入，并维护最远标记。
- 收藏品当前实现基础 collectibles.json 与本账号持有条目合并，尚未迁移旧面板基于创建时间的自动全解锁配置和全局静态响应分流；不能把普通账号读取测试视为这些特殊模式已兼容。
- 收藏品保留 user_id 别名；缺少账号或会话时只返回基础配置，不查询账号持有条目。成就定义当前只查询动态表，尚未迁移旧服务针对高版本的静态响应覆盖。
- get_vars 来自旧源码公开协议常量，不代表其中开关指向的功能已经实现。MOTD 使用无外部链接的开发默认公告，尚未接入旧公告配置。设备描述写入等附带行为尚未完全迁移。
- 登录 resource_list 和 websocket_url 保留旧协议字段，清单中其余资源及 websocket 实现仍待迁移；此处不保证完整客户端登录链路。
- 错误响应的详细文案、auth/login 特殊错误形状和部分表单兼容尚未逐分支对齐；目前测试验证核心状态、数据副作用及首批响应字段。
- 好友写接口只接受已存在的目标账号，不复制旧服务的占位账号创建行为；set_friend_name 会补建双向 friendship/friend_status，set_friend_block 只改当前账号方向，反向拉黑由读取端根据反向行判断。
- 关注接口按旧实现写入 boat_follow，`get_follower_count` 读取 follow_relationship；旧服务同样没有写入 follow_relationship 的业务路径，因此该计数在导入或后续写入前保持 0。
- `update_friend_constellation_pages` 按旧协议始终返回 HTTP 200，结果放在 result/status 字段；OPTIONS 返回 CORS 预检，非法分页只影响响应内容而不改状态码。
- 好友在线状态未接入 UDP/room-server 在线列表，列表接口统一返回离线；`join_friend_game` 只依据数据库中的服务器分配，不检查好友实时在线。
- 聊天没有持久化、WebSocket 或跨进程投递；同一 api-server 进程内广播给全部账号，客户端断开后只能依赖下一次轮询。多实例部署时聊天不互通。
- 邮件已读按旧语义逐条提交，不保证整个请求原子；`mark_seen` 只回显，不落库。
- 免费礼物 `gift_id` 由进程内单调计数生成，不设唯一约束；旧服务的同款行为保留。礼物领取不校验有效期，过期礼物在已知 `gift_id` 时仍可领取。
- `give_candle` 对零费用能力不写 `free_gift` 行，也不发送旧服务的临时礼物消息；需要客户端通过好友状态刷新感知能力变化。
- geonote 的读写都没有按查看者的可见性过滤；被标记 `reported` 或 `visible=false` 的留言仍会出现在列表和 bug 列表中，这与旧实现一致。`likes` 只在 `liked` 从 false→true 或 true→false 时增减，且是全局计数。
- `/account/geonote/request_geonotes_for_n_friends` 只按关卡返回留言，不读取好友关系；旧服务也没有注册无前缀的 `/account/request_geonotes_for_n_friends`，Go 保持同样缺口。
- 旧 geonote 返回体同时包含新、旧两套字段（`geonote_id`/`id`、`created_at`/`created_time`、`pos_x` 与 `pos` 等），Go 保留这一双写形状以便各版本客户端读取。
- 社交动态存在三套不同的 `reactions` 形状：服务读取为 8 键、账号 `get_reactions` 为 9 键、账号列表条目为 11 键；Go 按路由分别保留，未统一。
- `get_by_ids` 的 NIL 哨兵 id 回退（按最近搜索提示取本用户动态）尚未实现；未知 id 直接返回空列表。`feed_extra` 的 `private` 字段只写不读，旧服务也没有可见性过滤。
- 留影接口按旧实现不校验会话：任意调用者可创建、覆盖或读取任意 recording id；`/ugc` 只取路径最后一段并限制在 recordings 目录内，不校验 URL 中的用户段。
- 舞台接口同样不校验会话，调用者可用任意 `user`/`user_id` 读写对应 owner 的摆放；`friends_in_apartments` 不查询房间或在线状态，始终返回空好友表。
- `stage/set` 在无法识别 props 时不改写已有摆放；旧服务会额外保存 `last_request` 诊断字段，Go 未保留该字段。
- 预约接口不再返回写死的 `127.0.0.1:8443`，改用 `room_host/room_port`；`hmac` 仍为 64 个零，event id 固定。
- 管理好友接口没有实现旧服务“整段 /admin 由隧道中间件拦截”的部署逻辑，改为进程内回环或 `admin_token` 校验；跨机管理必须配置 token。
- 管理接口的 `pushed` 始终为 false，尚未接入 WebSocket `friends_updated`；在线客户端要等下一次好友列表轮询才能看到变化。

## 验证

2026-11-12 跨批次验收结果：`scripts/verify/inventory_routes.py` 重新扫描旧服务 201 条注册、201 个唯一路径，Go 侧 `pending=0`；`/ugc/<path:subpath>` 由 `/ugc/` 前缀路由处理。`go test -json ./...` 31 个测试全部通过，`go vet ./...`、Linux 构建和 Windows amd64 交叉构建通过；日志见 `logs/api-server-tests.jsonl`、`logs/api-server-vet.log`、`logs/api-server-build-linux.log`、`logs/api-server-build-windows.log`。测试使用临时数据库，不接触旧实例。

本机运行环境：配置 `services/api-server/config/api-server.local-test.json`，数据库 `services/api-server/data/local-test-20261112/api.db`，监听 `https://127.0.0.1:18443`，复用旧服务的游戏配置与 TLS 证书。`scripts/verify/local_smoke.py https://127.0.0.1:18443` 覆盖账号、资源、聊天、动态、留言、舞台、留影与 UGC Range、免费礼物、管理好友、预约和进房入口，**51 项全部通过**。

客户端验证据 `Tools/platform-tools/adb.exe devices` 检查当前无已连接设备，尚未执行 0.1.8 / 0.3.11 真机冷启动、认证、资源拉取、存档、进房与双端同步；不能把路由标记为客户端通过。API 冒烟和交叉编译也不等于 Windows 实机运行、TLS 与实际客户端互通。

## 后续迁移

依赖启动资源完整性继续实现装扮、基础定义、解锁、好友与邀请，然后完成购买/奖励事务、任务和社交内容接口。十批共 159 条路由已全部接入，路由复扫与 Go 测试已完成；下一步是真实客户端验收，并继续处理 TODO 中 room-server、数据导入、管理页面与部署切换等未完成项。完整迁移未完成，不把 501 路由计为成功接口。
