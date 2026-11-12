# 剩余游戏 API 路由分批实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 分 10 个可独立审阅的批次迁移当前剩余 159 条已扫描游戏 API 路由。

**Architecture:** 保持 api-server 的 handler → service → repository 分层和 SQLite 迁移链；同一业务的账号路径、service 路径与别名共用业务实现。room-server 独立启动，通过明确的服务接口协作，不能把内存状态当作跨服务数据。

**Tech Stack:** Go、net/http、SQLite（modernc.org/sqlite）；无需 Redis 或 MySQL。

**Spec:** [复刻 TODO](../../TODO.md)、[旧路由清单](../../../json/api/legacy-routes.json)、[客户端兼容矩阵](../../client-compatibility.md)。

## 口径与边界

- 基线为 2026-11-11 的 `docs/TODO.md` 未勾选路由：159 条；已接入 Go 路由 42 条。以下 10 批互斥，数量合计 159。分批只代表开发顺序，不代表测试或客户端验收通过。
- 前八批代码接入后，当前机器清单为 192 条已接入、9 条待迁移；本计划各批的原始数量保留为启动时快照，测试与客户端验收仍待执行。
- `json/api/legacy-routes.json` 已按当前 Go route map 刷新；之后每批继续同步更新 TODO、路由报告及开发进度。
- `/account/get_unlocks` 的全量自动解锁按客户端版本表合并，状态解锁和删除标记由 `/service/status/api/v1/get_unlocks` 独立读取；响应覆盖的管理员配置机制尚待迁移。此前 42 条已接入路由的兼容差异不计入 159，但应随所属业务批次复核。
- 旧注册方法包含 GET、POST 和 OPTIONS；扩展现有只允许 POST 的路由分发时按具体路径约束方法。未知路径仍为 404，未迁移旧路径仍为 501。
- 每批先核对旧请求、响应、版本条件、认证和数据库副作用；再实现实体或迁移、业务逻辑与路由。用户此前要求接口迁移期间暂缓测试，执行验证时以其最新指令为准，未执行不得标为通过。
- 本计划只覆盖静态扫描的 159 条；装饰器、动态注册、WebSocket、MQTT、Socket.IO、管理面板和 room-server 继续按 TODO 独立盘点，不计入本表。

## 批次总览

| 批次 | 范围 | 路由数 | 前置条件与交付重点 |
| --- | --- | ---: | --- |
| 1 | 启动资源、定义与认证 | 15 | 优先补齐登录后资源拉取需要的定义、活动配置与业务 token。核对版本分流及只读配置；远程配置上传不得用空成功响应代替。 |
| 2 | 解锁、魔法背包与增益 | 20 | 在 SQLite 中统一解锁、魔法、光翼和增益状态；补齐 get_unlocks 的全量模式、状态覆盖和删除标记，确保读取与写入一致。 |
| 3 | 商店、兑换与支付 | 16 | 先建立商品、库存和货币规则，再实现扣费与发放的单事务和幂等。Commerce 和兑换码依旧版真实能力处理外部依赖，不能虚构到账。 |
| 4 | 任务、赛季与进度 | 20 | 对齐任务定义、玩家进度、跨日与赛季边界；领奖与事件货币写入须防重复。 |
| 5 | 好友、关系与联机入口 | 19 | 在现有好友与邀请数据上补齐昵称、拉黑、关注、星标和远程装扮。加入好友房间需与独立 room-server 的状态接口约定一致。 |
| 6 | 消息、聊天与赠礼 | 17 | 消息投递、已读、聊天与免费礼物共用持久化语义；明确 API 与实时服务间的队列和断线重试行为。 |
| 7 | 地理笔记与留言 | 17 | 把 geonote 别名绑定到同一业务实现，补齐创建、查询、社交操作、编辑和删除的所有权校验。 |
| 8 | 社交动态、留影与 UGC | 26 | 先持久化留影和动态，再处理反应、评论、举报及 GET /ugc 文件读取；核对分页、路径和大小限制。 |
| 9 | 舞台、公寓与预约 | 6 | 对齐舞台状态、同公寓好友发现、离开和预约；房间状态不可只保存在单次 HTTP 请求内。 |
| 10 | 管理好友接口 | 3 | 隔离管理身份与普通玩家身份；保留旧管理路径的 GET/POST 方法，同时记录修改和审计。 |

## 每批共同完成标准

- [ ] 逐条记录旧源码路径、HTTP 方法、请求/响应字段、版本门控、错误状态和副作用；确认别名是否共用实现。
- [ ] 新 SQLite 表只通过递增迁移加入；写入需有会话与账号隔离，扣费、领奖和重复请求需有事务与幂等设计。
- [ ] 更新 Go route map、`docs/TODO.md`、`json/api/legacy-routes.json`、`docs/api-development.md`；记录已接入、静态检查、测试和客户端验证的不同状态。
- [ ] 可用工具链时执行格式、编译与相关静态检查；测试遵从用户的阶段安排。未验证行为保留在 TODO，不以 HTTP 200 空响应计作已迁移。
- [ ] 每批完成后检查 `git status` 与相关 diff，形成可单独审阅的修改单元；提交和推送须获得用户授权。

## 分批任务

### 第 1 批：启动资源、定义与认证（15 条，代码已接入）

优先补齐登录后资源拉取需要的定义、活动配置与业务 token。核对版本分流及只读配置；远程配置上传不得用空成功响应代替。

- [x] 核对并迁移以下入口：

  - `POST /account/get_surveys`
  - `POST /account/get_event_schedule`
  - `POST /account/get_spirit_shops`
  - `POST /account/get_currency_defs`
  - `POST /account/trust/get_trust_tier_defs`
  - `POST /account/get_video_defs`
  - `POST /account/get_radiance_spawn_defs`
  - `POST /account/get_level_pickups`
  - `POST /account/trust/get_trust_group_defs`
  - `POST /account/remote_config/upload`
  - `POST /account/trust/get_trust_activity_defs`
  - `POST /account/get_generic_shop_item_tracking_info`
  - `POST /service/auth/api/v1/token`
  - `POST /service/infraction/api/v1/get_infractions_status_decorated`
  - `POST /starwatch/auth/api/v1/auth`

- [x] 按共同完成标准更新清单与验证状态（静态构建通过；测试及客户端验收未执行）。

### 第 2 批：解锁、魔法背包与增益（20 条，代码已接入）

在 SQLite 中统一解锁、魔法、光翼和增益状态；补齐 get_unlocks 的全量模式、状态覆盖和删除标记，确保读取与写入一致。

- [x] 核对并迁移以下入口（静态构建通过，测试与客户端验收未执行）：

  - `POST /account/consumable/get_consumable_defs`
  - `POST /account/consumable/get_consumables`
  - `POST /account/consumable/consume_item1`
  - `POST /account/wing_buffs/get`
  - `POST /account/wing_buffs/collect`
  - `POST /account/collect_collectible`
  - `POST /account/wing_buffs/drop`
  - `POST /account/wing_buffs/deposit`
  - `POST /account/wing_buffs/convert`
  - `POST /account/buff/get_buffs1`
  - `POST /account/buff/get_buffs`
  - `POST /account/buff/remove_buff1`
  - `POST /account/buff/remove_buff`
  - `POST /account/consumable/get_consumables1`
  - `POST /account/ack_collectible`
  - `POST /account/buff/get_buff_defs`
  - `POST /service/status/api/v1/add_unlock`
  - `POST /service/status/api/v1/delete_unlock`
  - `POST /service/status/api/v1/get_unlocks`
  - `POST /service/inventory/api/v1/unlocks/remove_dye`

- [x] 按共同完成标准更新清单与验证状态（测试及客户端验收未执行）。

### 第 3 批：商店、兑换与支付（16 条，代码已接入）

先建立商品、库存和货币规则，再实现扣费与发放的单事务和幂等。Commerce 和兑换码依旧版真实能力处理外部依赖，不能虚构到账。

进度：16 条路由代码已接入；测试和客户端验收未执行。扣费与发放由 SQLite 单事务处理。旧 `commerce/receipt` 对凭据未做商店验证且会直接发货；Go 默认禁用模拟购买，仅在 `allow_fake_commerce=true` 时接受 `platform=fake` 且提供唯一 `transaction_id` 的请求，重复 ID 只返回已处理结果。旧客户端只发固定 `receipt` 时无法区分重试与再次购买，因此返回 `unique_transaction_id_required`，不发货；真实支付凭据返回 `receipt_verification_unavailable`，不假称验签成功。`commerce/fulfill` 仅确认已有的模拟订单。

- [x] 核对并迁移以下入口（静态构建通过，测试与客户端验收未执行）：

  - `POST /account/get_shop`
  - `POST /account/lootboxes/get`
  - `POST /account/lootboxes/redeem_box`
  - `POST /account/redeem_reward`
  - `POST /account/commerce/receipt`
  - `POST /account/commerce/fulfill`
  - `POST /account/commerce/update`
  - `POST,GET /account/commerce/product_info`
  - `POST /account/purchase_spirit_shop_item`
  - `POST /account/purchase_event_buff`
  - `POST /account/purchase_generic_shop_item`
  - `POST /account/purchase_unlock`
  - `POST /account/ack_forge_rate_reset`
  - `POST /account/buy_candle_wax`
  - `POST /account/code/verify`
  - `POST /account/purchase_dye`

- [x] 按共同完成标准更新清单与验证状态（测试及客户端验收未执行）。

### 第 4 批：任务、赛季与进度（20 条，代码已接入）

对齐任务定义、玩家进度、跨日与赛季边界；领奖与事件货币写入须防重复。

- [x] 核对并迁移以下入口（静态构建通过，测试与客户端验收未执行）：

  - `POST /account/set_achievement_stats`
  - `POST /account/get_daily_quests`
  - `POST /account/get_weekly_quests`
  - `POST /account/get_season_quests_v2`
  - `POST /account/get_season_pass_defs`
  - `POST /account/get_event_currency`
  - `POST /account/season_finalize`
  - `POST /account/season_start`
  - `POST /account/activate_season_quest`
  - `POST /account/claim_season_quest_reward`
  - `POST /account/get_season_quests`
  - `POST /account/get_account_world_quests`
  - `POST /account/claim_quest_reward`
  - `POST /account/collect_pickup_batch`
  - `POST /service/quest/api/v1/check_for_previous_completion`
  - `POST /service/quest/api/v1/get_player_quests`
  - `POST /service/quest/api/v1/get_player_quests_by_name`
  - `POST /service/quest/api/v1/get_quest_schedule`
  - `POST /service/quest/api/v1/create`
  - `POST /service/quest/api/v1/update_quest`

- [x] 按共同完成标准更新清单与验证状态（测试与客户端验收未执行）。

### 第 5 批：好友、关系与联机入口（19 条，代码已接入）

在现有好友与邀请数据上补齐昵称、拉黑、关注、星标和远程装扮。加入好友房间需与独立 room-server 的状态接口约定一致。

进度：19 条路由代码已接入；测试和客户端验收未执行。新增 schema v12 的好友关系、外部好友、机缘匹配、星标、服务器分配和关注表。`find_previous_or_empty` 会记录本次房间分配，`join_friend_game` 仍按旧实现读取好友最近一次 `server_assignment`；room-server 尚未实现，因此不能视为联机链路已通过。`get_capped` 保留旧固定响应，但不再通过无认证接口自动建号。

- [x] 核对并迁移以下入口（静态构建通过，测试与客户端验收未执行）：

  - `POST /account/get_blocked_friends`
  - `POST /account/set_friend_name`
  - `POST /account/set_friend_block`
  - `POST /account/get_external_friends`
  - `POST /account/serendipity/get`
  - `POST /account/star/get`
  - `POST /account/join_friend_game`
  - `POST /account/set_outfit`
  - `POST /account/set_joinable`
  - `POST /account/get_remote_outfit`
  - `POST /account/remove_relationship_hint`
  - `POST /service/relationship/api/v1/get_all_friend_gist`
  - `POST /service/relationship/api/v1/get_follower_count`
  - `POST,OPTIONS /service/relationship/api/v1/update_friend_constellation_pages`
  - `POST /service/trust/api/v1/get_capped`
  - `POST /service/trust/api/v1/get`
  - `POST /service/relationship/api/v1/add_follow`
  - `POST /service/relationship/api/v1/set_follow_name`
  - `POST /service/relationship/api/v1/delete_follow`

- [x] 按共同完成标准更新清单与验证状态（测试与客户端验收未执行）。

### 第 6 批：消息、聊天与赠礼（17 条，代码已接入）

消息投递、已读、聊天与免费礼物共用持久化语义；明确 API 与实时服务间的队列和断线重试行为。

进度：17 条路由代码已接入；测试和客户端验收未执行。新增 schema v13 的 `message` 与 `free_gift` 表。聊天消息按旧服务语义保存在进程内队列（每人最多 1000 条，重启即丢失），由 `/account/get_pending_messages` 轮询排空；未实现 WebSocket，因此没有实时推送。免费礼物持久化 30 天并支持幂等领取；`/account/chat/send` 的旧调试斜杠命令（改身高/货币/光翼）尚未迁移，会按普通聊天广播。

- [x] 核对并迁移以下入口（静态构建通过，测试与客户端验收未执行）：

  - `POST /account/get_pending_messages`
  - `POST /account/chat/send`
  - `POST /account/send`
  - `POST /account/send_message`
  - `POST /account/translate`
  - `POST /account/set_device_token`
  - `POST /account/give_candle`
  - `POST /account/claim_message_gift`
  - `POST /account/claim_message_gifts`
  - `POST /service/message/api/v2/get_all`
  - `POST /service/message/api/v1/get_all`
  - `POST /service/message/api/v1/get`
  - `POST /service/message/api/v1/mark_received`
  - `POST /service/message/api/v1/mark_seen`
  - `POST /service/relationship/api/v1/free_gifts/get_pending`
  - `POST,OPTIONS /service/relationship/api/v1/free_gifts/send`
  - `POST /service/relationship/api/v1/free_gifts/claim`

- [x] 按共同完成标准更新清单与验证状态（测试与客户端验收未执行）。

### 第 7 批：地理笔记与留言（17 条，代码已接入）

把 geonote 别名绑定到同一业务实现，补齐创建、查询、社交操作、编辑和删除的所有权校验。

进度：17 条路由代码已接入；测试和客户端验收未执行。新增 schema v14 的 `geonote` 表，替代旧服务的 `instance/geonotes.json` 旁路文件；旧 JSON 尚未提供导入工具，切换真实实例前需单独迁移。别名共用同一 handler，删除与编辑按作者校验所有权；读取仍按旧实现返回同关卡全部留言，`request_geonotes_for_n_friends` 不按好友过滤。

- [x] 核对并迁移以下入口（静态构建通过，测试与客户端验收未执行）：

  - `POST /account/get_user_geonotes`
  - `POST /account/geonote/get_user_geonotes`
  - `POST /account/geonote/request_geonotes_for_n_friends`
  - `POST /account/geonote/create_geonote2`
  - `POST /account/create_geonote2`
  - `POST /account/geonote/create_bug_report_geonote`
  - `POST /account/create_bug_report_geonote`
  - `POST /account/geonote/social_geonote2`
  - `POST /account/social_geonote2`
  - `POST /account/geonote/delete_geonote`
  - `POST /account/delete_geonote`
  - `POST /account/geonote/delete_bug_report_geonote`
  - `POST /account/delete_bug_report_geonote`
  - `POST /account/geonote/edit_message_geonote`
  - `POST /account/edit_message_geonote`
  - `POST /account/geonote/request_n_bug_report_geonotes`
  - `POST /account/request_n_bug_report_geonotes`

- [x] 按共同完成标准更新清单与验证状态（测试与客户端验收未执行）。

### 第 8 批：社交动态、留影与 UGC（26 条，代码已接入）

先持久化留影和动态，再处理反应、评论、举报及 GET /ugc 文件读取；核对分页、路径和大小限制。

进度：26 条路由代码已接入；测试和客户端验收未执行。新增 schema v15 的动态、点赞和附加数据表；留影按旧语义写入 `instance_dir/recordings`（元数据 JSON + v2 分块 NDJSON），`GET /ugc/...` 支持 Range。旧 `feed_extra/*.json` 与 recordings 文件尚未提供导入工具；评论、举报和笔友接口保持旧服务的空响应行为。

- [x] 核对并迁移以下入口（静态构建通过，测试与客户端验收未执行）：

  - `POST /account/social_feed/create`
  - `POST /account/social_feed/get_user_feeds`
  - `POST /account/social_feed/get_curated_feeds`
  - `POST /account/social_feed/get_reactions`
  - `POST /account/social_feed/like`
  - `POST /account/social_feed/delete`
  - `POST /service/recording/api/v1/create`
  - `POST /service/recording/api/v1/get`
  - `POST /service/recording/api/v2/create`
  - `POST /service/recording/api/v2/update`
  - `POST /service/recording/api/v2/get`
  - `GET /ugc/<path:subpath>`
  - `POST /service/social-feed/api/v2/get_curated_feeds`
  - `POST /service/social-feed/api/v1/get_user_feeds`
  - `POST /service/social-feed/api/v1/get_by_ids`
  - `POST /service/social-feed/api/v1/create`
  - `POST /service/social-feed/api/v1/hide`
  - `POST /service/social-feed/api/v1/like`
  - `POST /service/social-feed/api/v1/super_like`
  - `POST /service/social-feed/api/v1/delete`
  - `POST /service/social-feed/api/v1/edit`
  - `POST /service/social-feed/api/v1/get_reactions`
  - `POST /service/social-feed/api/v2/comments`
  - `POST /service/social-feed/api/v1/report`
  - `POST /service/social-feed/api/v2/get_penpals`
  - `POST /service/social-feed/api/v2/get_penpal_by_id`

- [x] 按共同完成标准更新清单与验证状态（测试与客户端验收未执行）。

### 第 9 批：舞台、公寓与预约（6 条）

对齐舞台状态、同公寓好友发现、离开和预约；房间状态不可只保存在单次 HTTP 请求内。

- [ ] 核对并迁移以下入口：

  - `POST /service/reservation/api/v1/reserve`
  - `POST /service/stage/api/v1/get`
  - `POST /service/stage/api/v1/set`
  - `POST /service/stage/api/v1/friends_in_apartments`
  - `POST /service/stage/api/v1/leave_apartment`
  - `POST /service/stage/api/v1/update_apartment_status`

- [ ] 按共同完成标准更新清单与验证状态。

### 第 10 批：管理好友接口（3 条）

隔离管理身份与普通玩家身份；保留旧管理路径的 GET/POST 方法，同时记录修改和审计。

- [ ] 核对并迁移以下入口：

  - `POST,GET /admin/api/friends_list`
  - `POST,GET /admin/api/friends_set`
  - `POST,GET /admin/api/friends_clear`

- [ ] 按共同完成标准更新清单与验证状态。

## 跨批次验收

- [ ] 10 批完成后重新扫描旧服务注册与 Go route map，确认 159 条均有对应处理函数，且动态路由缺口单独列出。
- [ ] 在用户同意测试后运行相关 Go 测试与两端实际 APK 验证，记录首个失败请求和实际版本；修复后再将路由标为客户端通过。
- [ ] 旧实例数据导入、room-server 与实时协议、管理页面和部署切换依照 TODO 另行实施，不因 159 条 HTTP 路由接入而视为完成。
