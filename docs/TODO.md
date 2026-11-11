# LaServer 复刻 TODO

基线：2026-11-11，依据当前 Go 代码、旧 LaServer 源码、路由报告和客户端兼容记录整理。本文件是后续待办；历史修复过程保留在 [TodoList.md](TodoList.md)。

## 当前状态与勾选规则

- 静态清单包含 201 条路由：39 条已有 Go handler、162 条待迁移。该数字不代表相关测试或客户端验收已完成，也不是整个旧服务的接口总数；别名可能指向相同业务。
- api-server 已能独立运行；room-server 只有独立 module，无实现或运行入口。
- 已有实体、HTTP handler 或本地测试通过，不等于真实账号数据迁移、完整客户端登录或联机通过。
- 勾选接口前需核对旧请求/响应、副作用和版本条件，完成实现与相关本地测试，更新路由报告。真实客户端验收单独记录，不随代码项自动勾选。
- 以下优先级是建议顺序。P0 先补齐登录依赖和验证基线，P1 形成联机与核心玩法闭环，P2 扩展内容和管理，P3 完善交付。同一组不代表可以跳过具体依赖。

## P0：基线、结构与登录资源

- [ ] 扩充路由盘点：检查 index.py、装饰器、动态注册、管理注册器和 transport 模块，区分实际注册、条件启用、静态响应和旧占位实现；记录方法、来源、认证、数据写入和版本范围。
- [ ] 建立按实际 APK 的兼容矩阵：优先 0.1.8、0.3.11；同时记录实际版本与上报的 0.13.4.170388，不能仅按上报构建判断。
- [ ] 以登录 resource_list 建立资源依赖表，逐一对应请求路径，明确哪些 501 会阻断登录；优先补齐装扮、解锁、好友、任务和魔法定义。
- [ ] 根据 AGENTS.md 按完整业务单元拆分 internal/gameapi 到 handler/service/repository/middleware/config/app；model 不直接作为 API DTO，禁止 HTTP 层直写库，已有行为不变。
- [ ] 将 SQL 提取到 api-server/migrations/，核对 schema v1–v6 的版本、唯一约束与事务语义；添加失败回滚和未知版本拒绝用例，不重写已发布迁移。
- [ ] 明确静态定义资源的来源、敏感性和可分发性，只读配置可外置，缺失时明确报错，不把真实账号快照作为通用响应。

### 已接入 39 个接口仍待对齐的行为

- [ ] 对齐 auth/create、auth/login 与 legacy create/login 的错误响应分支、表单/BOM 等兼容、恢复令牌及设备信息更新；记录允许与拒绝的请求证据。
- [ ] 研究真实客户端可提供的设备/签名/恢复凭据，再确定登录身份校验；当前仅凭 user ID 签发会话不能视为安全认证。保持隔离开发部署直到方案验证。
- [ ] 迁移自动授权、自动补建账号、默认货币及自动全解锁的配置与生效时间规则；不把新配置字段的简化行为当作旧面板功能已完成。
- [ ] 迁移公告配置与时间模式，不继续使用开发默认公告代替旧配置；核对设备描述写回和登录其他附带副作用。
- [ ] 完成静态响应分流、client version overrides、协议族、响应体积限制和按版本过滤；尤其检查成就定义高版本静态覆盖及老客户端不可识别事件。
- [ ] 对齐收藏品自动全动作、基础条目、账号覆盖条目，以及解锁/服饰版本差异；保留匿名仅返回基础配置的边界。
- [ ] 将 userdata/sync_user_data 的严格会话校验列为显式兼容差异，实际客户端验证后固定；验证空存档包、重启恢复和存储错误回滚。
- [ ] 验证季卡太平洋日界、夏令时、并发/重试幂等，迁移旧领取标记，防止切换当天重复发放。

## P1：持久化数据与核心业务闭环

- [ ] 盘点旧 instance 和 legacy 两类库的真实结构与版本，分别制定映射；不默认合库或复制文件即可使用。
- [ ] 对比 72 个来源实体与当前实际建表，明确尚未建立的表、默认值、外键、索引和复合唯一键。
- [ ] 实现显式导入工具：只处理隔离副本，新目标拒绝覆盖；时间戳混合格式、JSON/base64 原文、重复货币行和外键异常有可审计处理规则。
- [ ] 对齐存档、货币、装扮、解锁、好友、任务和消息行数及关键值；迁移旧旁路文件、UGC 内容与季卡标记，报告无法迁移项。
- [ ] 实现购买/扣费/发物品/奖励发放的一致事务与幂等策略，测试负数、余额不足、并发重复领取和中途失败回滚。
- [ ] 完成好友双向关系、邀请接受、昵称、拉黑、在线状态、赠礼与同房发现，明确离线与不可达行为。
- [ ] 完成任务激活、进度、完成、领奖、过期和赛季结算；跨日口径和货币快照保持一致。

## P1：room-server 与实时通信

- [ ] 在独立 module 实现 cmd/server、配置加载、端口校验、日志、信号退出和监听器清理，不直接导入或读写 api-server 内部实现/SQLite。
- [ ] 只读参考旧 realtime-server 的 transport、protocol、session、services、domain、api、observability，建立迁移清单。
- [ ] 验证 Go ENet 候选的许可证、平台与 CGO 依赖、checksum、channel、可靠/不可靠包、分片和重传；先与实际客户端握手，不承诺未验证的纯 Go 单文件交付。
- [ ] 迁移连接会话、超时、重复连接、断线与重连；房间状态由单一所有者串行管理，测试并发进出和状态释放。
- [ ] 迁移进房握手、玩家广播、移动/姿态同步、游戏消息、离房与自然过图；不同关卡和玩家不可串状态。
- [ ] 分批迁移旧专项版本与 legacy/mqtt/modern 协议族，先核对对应 codec；扫描过的版本不能当作运行已支持。
- [ ] 定义 API 与 room-server 的会话认证、状态查询和失联语义，设置超时；两个服务均可独立启动/停止，运行时依赖失联时不得绕过认证。
- [ ] 实现实时服务本机管理 HTTP、玩家/房间状态及必要指标，敏感抓包按需开启并脱敏。
- [ ] 迁移 API 侧 WebSocket、Socket.IO 和 MQTT over WebSocket，核对认证、主题/事件、消息封装与版本行为，不能统一替换为普通 WebSocket。

## P2：内容与管理功能

- [ ] 迁移留言/纸船、社交动态、点赞/评论/举报/关注等账号隔离与分页规则。
- [ ] 迁移留影 v1/v2、分块上传、seek_table、UGC Range 下载和共享空间持久化；校验大小、路径、生命周期和删除权限。
- [ ] 核对旧 commerce/IAP/客服/翻译等是否真实实现、外部依赖或占位；按证据复刻并明确不支持范围，不编造支付成功或外部服务结果。
- [ ] 盘点 Tkinter 管理端全部实际功能与直写数据库路径；迁移账号授权/封禁、货币/解锁/魔法、好友、公告、游戏配置及响应覆盖等操作。
- [ ] 实现管理 API、独立认证、写操作防跨站请求、审计与敏感字段脱敏；管理端调用业务 service，不直接修改 SQLite。
- [ ] 在 api-server 实现模板管理页面与 embed 静态资源，无须引入 Node.js 构建链；页面区分两个服务在线/离线/不可达状态。
- [ ] 单独评估旧证书、隧道、DDNS、启停、热重载和监控功能的保留方案；不能因原来由桌面面板承载就遗漏，也不默认照搬全局 monkey patch。

## P3：验收、部署和切换

- [ ] 扩展 JSON 合同、真实 SQLite 集成和协议回归；覆盖无效会话、跨账号读写、缺字段、重复请求、并发、过期和恢复。
- [ ] 实际验证 0.1.8 与 0.3.11 的冷启动、认证、资源拉取、存档、进房、双端同步及自然过图，记录具体 APK 和结果；之后再扩大版本和混联组合。
- [ ] 在 Windows amd64、Linux amd64 干净环境分别验证两个服务的配置、数据库迁移、端口冲突、停止与重启；交叉编译不等于 Windows 实机已通过。
- [ ] 验证旧客户端 HTTPS/TLS、证书/地址和 UDP 可达性；API 下发地址成功不等于实时链路已通过。
- [ ] 分别编写服务构建/校验/发布脚本，产物按版本归档并附依赖说明、校验和与回退文档，不覆盖已有正式产物。
- [ ] 经授权后演练真实实例切换：停写、备份、导入、核验、启用。新服务已有写入时先保存并验证回迁，不能直接恢复旧备份声称无损。
- [ ] 每个业务批次更新 README、兼容矩阵、路由报告和修复记录；性能目标只依据实际测量填写。

## 待迁移路由逐项清单

以下 162 项来自 json/api/legacy-routes.json。按注册来源分组，保留 HTTP 方法与处理函数，便于反查旧代码。仅覆盖已扫描注册，不替代前述清单外任务。一个业务及其别名可一起实现，但每个入口均需验证。此清单后续人工更新，不自动把代码存在视为已验收。

### 游戏账号与资源接口（122 条）

- [ ] `POST /account/get_blocked_friends` — `get_blocked_friends`；旧来源 `business/account/__init__.py:164`
- [ ] `POST /account/set_friend_name` — `set_friend_name`；旧来源 `business/account/__init__.py:165`
- [ ] `POST /account/set_friend_block` — `set_friend_block`；旧来源 `business/account/__init__.py:168`
- [ ] `POST /account/set_achievement_stats` — `set_achievement_stats`；旧来源 `business/account/__init__.py:170`
- [ ] `POST /account/get_surveys` — `get_surveys`；旧来源 `business/account/__init__.py:171`
- [ ] `POST /account/get_shop` — `get_shop`；旧来源 `business/account/__init__.py:172`
- [ ] `POST /account/lootboxes/get` — `get_lootboxes`；旧来源 `business/account/__init__.py:173`
- [ ] `POST /account/lootboxes/redeem_box` — `redeem_box`；旧来源 `business/account/__init__.py:174`
- [ ] `POST /account/get_external_friends` — `get_external_friends`；旧来源 `business/account/__init__.py:175`
- [ ] `POST /account/serendipity/get` — `get_serendipity`；旧来源 `business/account/__init__.py:176`
- [ ] `POST /account/star/get` — `get_star_tags`；旧来源 `business/account/__init__.py:177`
- [ ] `POST /account/get_pending_messages` — `get_pending_messages`；旧来源 `business/account/__init__.py:178`
- [ ] `POST /account/get_generic_shops` — `generic_shop_defs`；旧来源 `business/account/__init__.py:179`
- [ ] `POST /account/consumable/get_consumable_defs` — `get_consumable_defs`；旧来源 `business/account/__init__.py:180`
- [ ] `POST /account/consumable/get_consumables` — `get_consumables`；旧来源 `business/account/__init__.py:181`
- [ ] `POST /account/consumable/consume_item1` — `consume_item1`；旧来源 `business/account/__init__.py:182`
- [ ] `POST /account/redeem_reward` — `redeem_reward`；旧来源 `business/account/__init__.py:187`
- [ ] `POST /account/get_unlocks` — `get_unlocks`；旧来源 `business/account/__init__.py:188`
- [ ] `POST /account/get_event_schedule` — `get_event_schedule`；旧来源 `business/account/__init__.py:189`
- [ ] `POST /account/commerce/receipt` — `commerce_receipt`；旧来源 `business/account/__init__.py:191`
- [ ] `POST /account/commerce/fulfill` — `commerce_fulfill`；旧来源 `business/account/__init__.py:192`
- [ ] `POST /account/commerce/update` — `commerce_update`；旧来源 `business/account/__init__.py:193`
- [ ] `POST,GET /account/commerce/product_info` — `commerce_product_info`；旧来源 `business/account/__init__.py:194`
- [ ] `POST /account/get_spirit_shops` — `get_spirit_shops`；旧来源 `business/account/__init__.py:196`
- [ ] `POST /account/purchase_spirit_shop_item` — `purchase_spirit_shop_item`；旧来源 `business/account/__init__.py:198`
- [ ] `POST /account/get_user_geonotes` — `get_user_geonotes`；旧来源 `business/account/__init__.py:199`
- [ ] `POST /account/geonote/get_user_geonotes` — `get_user_geonotes2`；旧来源 `business/account/__init__.py:200`
- [ ] `POST /account/geonote/request_geonotes_for_n_friends` — `request_geonotes_for_n_friends`；旧来源 `business/account/__init__.py:201`
- [ ] `POST /account/geonote/create_geonote2` — `create_geonote2`；旧来源 `business/account/__init__.py:202`
- [ ] `POST /account/create_geonote2` — `create_geonote2`；旧来源 `business/account/__init__.py:203`
- [ ] `POST /account/geonote/create_bug_report_geonote` — `create_bug_report_geonote`；旧来源 `business/account/__init__.py:204`
- [ ] `POST /account/create_bug_report_geonote` — `create_bug_report_geonote`；旧来源 `business/account/__init__.py:205`
- [ ] `POST /account/geonote/social_geonote2` — `social_geonote2`；旧来源 `business/account/__init__.py:206`
- [ ] `POST /account/social_geonote2` — `social_geonote2`；旧来源 `business/account/__init__.py:207`
- [ ] `POST /account/geonote/delete_geonote` — `delete_geonote`；旧来源 `business/account/__init__.py:208`
- [ ] `POST /account/delete_geonote` — `delete_geonote`；旧来源 `business/account/__init__.py:209`
- [ ] `POST /account/geonote/delete_bug_report_geonote` — `delete_bug_report_geonote`；旧来源 `business/account/__init__.py:210`
- [ ] `POST /account/delete_bug_report_geonote` — `delete_bug_report_geonote`；旧来源 `business/account/__init__.py:211`
- [ ] `POST /account/geonote/edit_message_geonote` — `edit_message_geonote`；旧来源 `business/account/__init__.py:212`
- [ ] `POST /account/edit_message_geonote` — `edit_message_geonote`；旧来源 `business/account/__init__.py:213`
- [ ] `POST /account/geonote/request_n_bug_report_geonotes` — `request_n_bug_report_geonotes`；旧来源 `business/account/__init__.py:214`
- [ ] `POST /account/request_n_bug_report_geonotes` — `request_n_bug_report_geonotes`；旧来源 `business/account/__init__.py:215`
- [ ] `POST /account/get_daily_quests` — `get_daily_quests`；旧来源 `business/account/__init__.py:216`
- [ ] `POST /account/get_weekly_quests` — `get_weekly_quests`；旧来源 `business/account/__init__.py:217`
- [ ] `POST /account/get_season_quests_v2` — `get_season_quests_v2`；旧来源 `business/account/__init__.py:218`
- [ ] `POST /account/get_season_pass_defs` — `get_season_pass_defs`；旧来源 `business/account/__init__.py:219`
- [ ] `POST /account/get_currency_defs` — `get_currency_defs`；旧来源 `business/account/__init__.py:220`
- [ ] `POST /account/get_event_currency` — `get_event_currency`；旧来源 `business/account/__init__.py:221`
- [ ] `POST /account/wing_buffs/get` — `get_wing_buffs`；旧来源 `business/account/__init__.py:222`
- [ ] `POST /account/wing_buffs/collect` — `wing_buffs_collect`；旧来源 `business/account/__init__.py:223`
- [ ] `POST /account/collect_collectible` — `collect_collectible`；旧来源 `business/account/__init__.py:224`
- [ ] `POST /account/wing_buffs/drop` — `wing_buffs_drop`；旧来源 `business/account/__init__.py:225`
- [ ] `POST /account/wing_buffs/deposit` — `wing_buffs_deposit`；旧来源 `business/account/__init__.py:226`
- [ ] `POST /account/wing_buffs/convert` — `wing_buffs_convert`；旧来源 `business/account/__init__.py:227`
- [ ] `POST /account/buff/get_buffs1` — `get_buffs1`；旧来源 `business/account/__init__.py:229`
- [ ] `POST /account/buff/get_buffs` — `get_buffs`；旧来源 `business/account/__init__.py:230`
- [ ] `POST /account/buff/remove_buff1` — `remove_buff1`；旧来源 `business/account/__init__.py:232`
- [ ] `POST /account/buff/remove_buff` — `remove_buff1`；旧来源 `business/account/__init__.py:233`
- [ ] `POST /account/purchase_event_buff` — `purchase_event_buff`；旧来源 `business/account/__init__.py:234`
- [ ] `POST /account/purchase_generic_shop_item` — `purchase_generic_shop_item`；旧来源 `business/account/__init__.py:235`
- [ ] `POST /account/purchase_unlock` — `purchase_unlock`；旧来源 `business/account/__init__.py:239`
- [ ] `POST /account/join_friend_game` — `join_friend_game`；旧来源 `business/account/__init__.py:243`
- [ ] `POST /account/consumable/get_consumables1` — `get_consumables1`；旧来源 `business/account/__init__.py:244`
- [ ] `POST /account/trust/get_trust_tier_defs` — `get_trust_tier_defs`；旧来源 `business/account/__init__.py:245`
- [ ] `POST /account/get_video_defs` — `get_video_defs`；旧来源 `business/account/__init__.py:247`
- [ ] `POST /account/get_radiance_spawn_defs` — `get_radiance_spawn_defs`；旧来源 `business/account/__init__.py:248`
- [ ] `POST /account/get_level_pickups` — `get_level_pickups`；旧来源 `business/account/__init__.py:250`
- [ ] `POST /account/ack_forge_rate_reset` — `ack_forge_rate_reset`；旧来源 `business/account/__init__.py:252`
- [ ] `POST /account/trust/get_trust_group_defs` — `get_trust_group_defs`；旧来源 `business/account/__init__.py:254`
- [ ] `POST /account/season_finalize` — `season_finalize`；旧来源 `business/account/__init__.py:255`
- [ ] `POST /account/season_start` — `season_start`；旧来源 `business/account/__init__.py:256`
- [ ] `POST /account/activate_season_quest` — `activate_season_quest`；旧来源 `business/account/__init__.py:257`
- [ ] `POST /account/claim_season_quest_reward` — `claim_season_quest_reward`；旧来源 `business/account/__init__.py:258`
- [ ] `POST /account/ack_collectible` — `ack_collectible`；旧来源 `business/account/__init__.py:259`
- [ ] `POST /account/remote_config/upload` — `upload_remote_config`；旧来源 `business/account/__init__.py:260`
- [ ] `POST /account/get_season_quests` — `get_season_quests`；旧来源 `business/account/__init__.py:261`
- [ ] `POST /account/trust/get_trust_activity_defs` — `get_trust_activity_defs`；旧来源 `business/account/__init__.py:262`
- [ ] `POST /account/buff/get_buff_defs` — `get_buff_defs`；旧来源 `business/account/__init__.py:263`
- [ ] `POST /account/get_account_world_quests` — `get_account_world_quests`；旧来源 `business/account/__init__.py:264`
- [ ] `POST /account/claim_quest_reward` — `claim_quest_reward`；旧来源 `business/account/__init__.py:265`
- [ ] `POST /account/set_outfit` — `set_outfit`；旧来源 `business/account/__init__.py:266`
- [ ] `POST /account/set_joinable` — `set_joinable`；旧来源 `business/account/__init__.py:267`
- [ ] `POST /account/collect_pickup_batch` — `collect_pickup_batch`；旧来源 `business/account/__init__.py:269`
- [ ] `POST /account/buy_candle_wax` — `buy_candle_wax`；旧来源 `business/account/__init__.py:270`
- [ ] `POST /account/code/verify` — `verify_invite_code`；旧来源 `business/account/__init__.py:272`
- [ ] `POST /account/social_feed/create` — `create_social_feed`；旧来源 `business/account/__init__.py:276`
- [ ] `POST /account/social_feed/get_user_feeds` — `get_user_feeds`；旧来源 `business/account/__init__.py:277`
- [ ] `POST /account/social_feed/get_curated_feeds` — `get_curated_feeds`；旧来源 `business/account/__init__.py:278`
- [ ] `POST /account/social_feed/get_reactions` — `get_social_feed_reactions`；旧来源 `business/account/__init__.py:279`
- [ ] `POST /account/social_feed/like` — `social_feed_like`；旧来源 `business/account/__init__.py:280`
- [ ] `POST /account/social_feed/delete` — `delete_social_feed`；旧来源 `business/account/__init__.py:281`
- [ ] `POST /account/ack_unlock` — `ack_unlock`；旧来源 `business/account/__init__.py:282`
- [ ] `POST /account/chat/send` — `handle_chat_message`；旧来源 `business/account/__init__.py:283`
- [ ] `POST /account/send` — `handle_send_message`；旧来源 `business/account/__init__.py:287`
- [ ] `POST /account/send_message` — `handle_send_message`；旧来源 `business/account/__init__.py:288`
- [ ] `POST /account/translate` — `handle_translate`；旧来源 `business/account/__init__.py:289`
- [ ] `POST /account/set_device_token` — `set_device_token`；旧来源 `business/account/__init__.py:290`
- [ ] `POST /account/get_remote_outfit` — `get_remote_outfit`；旧来源 `business/account/__init__.py:292`
- [ ] `POST /account/give_candle` — `give_candle`；旧来源 `business/account/__init__.py:293`
- [ ] `POST /account/claim_message_gift` — `claim_message_gift`；旧来源 `business/account/__init__.py:296`
- [ ] `POST /account/claim_message_gifts` — `claim_message_gift`；旧来源 `business/account/__init__.py:297`
- [ ] `POST /account/remove_relationship_hint` — `remove_relationship_hint`；旧来源 `business/account/__init__.py:299`
- [ ] `POST /account/get_generic_shop_item_tracking_info` — `get_generic_shop_item_tracking_info`；旧来源 `business/account/__init__.py:300`
- [ ] `POST /account/purchase_dye` — `purchase_dye`；旧来源 `business/account/__init__.py:301`

### 游戏业务接口（54 条）

- [ ] `POST /service/auth/api/v1/token` — `get_auth_token`；旧来源 `business/service/__init__.py:86`
- [ ] `POST /service/infraction/api/v1/get_infractions_status_decorated` — `get_infractions_status_decorated`；旧来源 `business/service/__init__.py:87`
- [ ] `POST /service/message/api/v2/get_all` — `get_all_messages_v2`；旧来源 `business/service/__init__.py:88`
- [ ] `POST /service/message/api/v1/get_all` — `get_all_messages_v1`；旧来源 `business/service/__init__.py:89`
- [ ] `POST /service/message/api/v1/get` — `get_message`；旧来源 `business/service/__init__.py:90`
- [ ] `POST /service/message/api/v1/mark_received` — `mark_received`；旧来源 `business/service/__init__.py:91`
- [ ] `POST /service/message/api/v1/mark_seen` — `mark_seen`；旧来源 `business/service/__init__.py:92`
- [ ] `POST /service/quest/api/v1/check_for_previous_completion` — `check_for_previous_completion`；旧来源 `business/service/__init__.py:93`
- [ ] `POST /service/quest/api/v1/get_player_quests` — `get_player_quests`；旧来源 `business/service/__init__.py:94`
- [ ] `POST /service/quest/api/v1/get_player_quests_by_name` — `get_player_quests_by_name`；旧来源 `business/service/__init__.py:95`
- [ ] `POST /service/quest/api/v1/get_quest_schedule` — `get_quest_schedule`；旧来源 `business/service/__init__.py:96`
- [ ] `POST /service/quest/api/v1/create` — `quest_create`；旧来源 `business/service/__init__.py:97`
- [ ] `POST /service/quest/api/v1/update_quest` — `quest_update`；旧来源 `business/service/__init__.py:98`
- [ ] `POST /service/relationship/api/v1/get_all_friend_gist` — `get_all_friend_gist`；旧来源 `business/service/__init__.py:99`
- [ ] `POST /service/relationship/api/v1/get_follower_count` — `get_follower_count`；旧来源 `business/service/__init__.py:100`
- [ ] `POST,OPTIONS /service/relationship/api/v1/update_friend_constellation_pages` — `update_friend_constellation_pages_entry`；旧来源 `business/service/__init__.py:106`
- [ ] `POST /service/relationship/api/v1/free_gifts/get_pending` — `get_pending_free_gifts`；旧来源 `business/service/__init__.py:112`
- [ ] `POST,OPTIONS /service/relationship/api/v1/free_gifts/send` — `send_free_gift_options`；旧来源 `business/service/__init__.py:113`
- [ ] `POST /service/relationship/api/v1/free_gifts/claim` — `claim_free_gift`；旧来源 `business/service/__init__.py:114`
- [ ] `POST /service/reservation/api/v1/reserve` — `reserve`；旧来源 `business/service/__init__.py:115`
- [ ] `POST /service/recording/api/v1/create` — `recording_create`；旧来源 `business/service/__init__.py:116`
- [ ] `POST /service/recording/api/v1/get` — `recording_get`；旧来源 `business/service/__init__.py:117`
- [ ] `POST /service/recording/api/v2/create` — `recording_v2_create`；旧来源 `business/service/__init__.py:119`
- [ ] `POST /service/recording/api/v2/update` — `recording_v2_update`；旧来源 `business/service/__init__.py:120`
- [ ] `POST /service/recording/api/v2/get` — `recording_v2_get`；旧来源 `business/service/__init__.py:121`
- [ ] `GET /ugc/<path:subpath>` — `serve_recording_ugc`；旧来源 `business/service/__init__.py:122`
- [ ] `POST /service/social-feed/api/v2/get_curated_feeds` — `get_curated_feed`；旧来源 `business/service/__init__.py:123`
- [ ] `POST /service/social-feed/api/v1/get_user_feeds` — `v1get_user_feeds`；旧来源 `business/service/__init__.py:124`
- [ ] `POST /service/social-feed/api/v1/get_by_ids` — `v1_get_by_ids`；旧来源 `business/service/__init__.py:125`
- [ ] `POST /service/stage/api/v1/get` — `get_stage`；旧来源 `business/service/__init__.py:126`
- [ ] `POST /service/stage/api/v1/set` — `set_stage`；旧来源 `business/service/__init__.py:128`
- [ ] `POST /service/stage/api/v1/friends_in_apartments` — `friends_in_apartments`；旧来源 `business/service/__init__.py:129`
- [ ] `POST /service/stage/api/v1/leave_apartment` — `leave_apartment`；旧来源 `business/service/__init__.py:130`
- [ ] `POST /service/stage/api/v1/update_apartment_status` — `update_apartment_status`；旧来源 `business/service/__init__.py:131`
- [ ] `POST /service/status/api/v1/add_unlock` — `add_unlock`；旧来源 `business/service/__init__.py:132`
- [ ] `POST /service/status/api/v1/delete_unlock` — `delete_unlock`；旧来源 `business/service/__init__.py:133`
- [ ] `POST /service/status/api/v1/get_unlocks` — `get_unlocks_status`；旧来源 `business/service/__init__.py:134`
- [ ] `POST /service/trust/api/v1/get_capped` — `get_capped`；旧来源 `business/service/__init__.py:135`
- [ ] `POST /service/trust/api/v1/get` — `get_trust`；旧来源 `business/service/__init__.py:137`
- [ ] `POST /service/social-feed/api/v1/create` — `social_feed_create`；旧来源 `business/service/__init__.py:139`
- [ ] `POST /service/social-feed/api/v1/hide` — `social_feed_hide`；旧来源 `business/service/__init__.py:140`
- [ ] `POST /service/social-feed/api/v1/like` — `svc_social_feed_like`；旧来源 `business/service/__init__.py:141`
- [ ] `POST /service/social-feed/api/v1/super_like` — `social_feed_super_like`；旧来源 `business/service/__init__.py:142`
- [ ] `POST /service/social-feed/api/v1/delete` — `social_feed_delete`；旧来源 `business/service/__init__.py:143`
- [ ] `POST /service/social-feed/api/v1/edit` — `social_feed_edit`；旧来源 `business/service/__init__.py:144`
- [ ] `POST /service/relationship/api/v1/add_follow` — `relationship_add_follow`；旧来源 `business/service/__init__.py:145`
- [ ] `POST /service/relationship/api/v1/set_follow_name` — `relationship_set_follow_name`；旧来源 `business/service/__init__.py:146`
- [ ] `POST /service/relationship/api/v1/delete_follow` — `relationship_delete_follow`；旧来源 `business/service/__init__.py:147`
- [ ] `POST /service/inventory/api/v1/unlocks/remove_dye` — `inventory_remove_dye`；旧来源 `business/service/__init__.py:148`
- [ ] `POST /service/social-feed/api/v1/get_reactions` — `social_feed_get_reactions`；旧来源 `business/service/__init__.py:150`
- [ ] `POST /service/social-feed/api/v2/comments` — `social_feed_comments`；旧来源 `business/service/__init__.py:151`
- [ ] `POST /service/social-feed/api/v1/report` — `social_feed_report`；旧来源 `business/service/__init__.py:152`
- [ ] `POST /service/social-feed/api/v2/get_penpals` — `social_feed_get_penpals`；旧来源 `business/service/__init__.py:153`
- [ ] `POST /service/social-feed/api/v2/get_penpal_by_id` — `social_feed_get_penpal_by_id`；旧来源 `business/service/__init__.py:154`

### 已扫描的管理接口（3 条）

- [ ] `POST,GET /admin/api/friends_list` — `_endpoint_friends_list`；旧来源 `business/admin/__init__.py:165`
- [ ] `POST,GET /admin/api/friends_set` — `_endpoint_friends_set`；旧来源 `business/admin/__init__.py:167`
- [ ] `POST,GET /admin/api/friends_clear` — `_endpoint_friends_clear`；旧来源 `business/admin/__init__.py:169`

### 认证与统计接口（1 条）

- [ ] `POST /starwatch/auth/api/v1/auth` — `auth`；旧来源 `business/starwatch/__init__.py:17`
