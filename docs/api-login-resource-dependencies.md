# API 登录资源依赖表

来源：旧 `create/login/auth_create/auth_login` 返回体、旧静态响应清单、Go `legacy_contracts.go` 与 route map。逐项机器可读证据见 [资源审计 JSON](../json/api/task1-login-resources.json)。运行 `python scripts/docs/audit_api_task1.py` 更新。

四种流程的旧资源列表与 Go 列表逐项一致，共 48 个不同名称；这只证明服务端声明一致。现有 APK 日志未保存登录响应体，也未保存客户端资源名到 URL 的映射，因此下表的 URL 是响应键匹配或显式标注的候选，绝不当作客户端映射实测。0.1.8 和 0.3.11 列仅表示候选 URL 在已有隔离 API 日志中出现 POST 200，不代表资源内容正确。两版测试包均实际调用 `/account/auth/login`；0.3.11 也曾调用 `/account/auth/create`。APK 真实版本与其上报的 User-Agent 分开记录在 [兼容矩阵](client-compatibility.md)。

旧静态库 `before_request` 仅在上报版本达到 0.26.0 且路径不在 `_dynamic_exclude.json` 时截断动态处理；这项门槛由请求头决定，不能仅凭 APK 文件版本推断。Go 版本过滤按各 handler 单独处理。配置来源列只列出候选 Go handler 中可直接看到的文件、SQLite 或内嵌数据；辅助函数的间接依赖仍须在 Task 3 检查。认证列同样只表达该 handler 中直接可见的会话校验信号。

失败影响为排查优先级推断：`启动链` 表示登录后资源同步可能受阻；`功能` 表示相关界面或状态可能不可用；`待确认` 表示找不到独立资源入口，须以客户端追踪判定是否实际请求。

| 资源名 | 候选接口 / 映射 | 配置或数据来源（Go 直接可见） | 认证信号 | Go 处理 | 0.1.8 / 0.3.11 200 | 失败影响 |
| --- | --- | --- | --- | --- | --- | --- |
| `achievement_defs` | `/account/get_achievement_defs` | SQLite; 旧静态库:account__get_achievement_defs.json | 见直接会话检查 | `AchievementDefinitions` / `profile.go` | — / — | 功能 |
| `achievement_stats` | `/account/get_achievement_stats` | SQLite; 旧静态库:account__get_achievement_stats.json | 见直接会话检查 | `AchievementStats` / `player_state.go` | ✓ / ✓ | 功能 |
| `achievement_stats_tracking` | `/account/get_achievement_stats_tracking` | 代码常量或辅助函数，待逐分支核对; 旧静态库:account__get_achievement_stats_tracking.json | 未见直接会话检查 | `AchievementStatsTracking` / `player_state.go` | — / — | 功能 |
| `achievements` | `/account/get_achievements` | SQLite; 旧静态库:account__get_achievements.json | 见直接会话检查 | `Achievements` / `profile.go` | ✓ / ✓ | 功能 |
| `checkpoints` | `/account/get_checkpoints` | SQLite; 旧静态库:account__get_checkpoints.json | 见直接会话检查 | `Checkpoints` / `data.go` | ✓ / ✓ | 启动链 |
| `collectibles` | `/account/get_collectibles` | game_config_or_embedded_json:auto_full_unlock.json,collectibles.json; SQLite; 旧静态库:account__get_collectibles.json | 见直接会话检查 | `Collectibles` / `profile.go` | ✓ / ✓ | 功能 |
| `currency` | `/account/get_currency` | SQLite; 旧静态库:account__get_currency.json | 见直接会话检查 | `Currency` / `data.go` | ✓ / ✓ | 启动链 |
| `device_capabilities` | `/account/device_capabilities/lookup` | 代码常量或辅助函数，待逐分支核对; 旧静态库:account__device_capabilities__lookup.json | 未见直接会话检查 | `Capabilities` / `bootstrap.go` | ✓ / ✓ | 启动链 |
| `emote_sequences` | 无独立入口证据 | 未找到独立数据来源 | 未见直接会话检查 | 无已知处理 | — / — | 待确认 |
| `event_currency_defs` | `/account/get_event_currency_defs` | embedded_legacydata; 旧静态库:account__get_event_currency_defs.json | 未见直接会话检查 | `EventCurrencyDefinitions` / `player_state.go` | — / — | 功能 |
| `event_schedule` | `/account/get_event_schedule` | game_config_or_embedded_json:game_control.json; 旧静态库:account__get_event_schedule.json | 未见直接会话检查 | `EventSchedule` / `batch1_schedule.go` | ✓ / ✓ | 启动链 |
| `external_account_friends` | `/account/get_external_friends` | SQLite; 旧静态库:account__get_external_friends.json | 见直接会话检查 | `GetExternalFriends` / `relationships_batch5.go` | ✓ / ✓ | 功能 |
| `external_links` | 登录响应字段（无独立请求） | SQLite; 旧静态库:account__auth__login.json | 未见直接会话检查 | `Login` / `account.go` | ✓ / ✓ | 功能 |
| `forge_rates` | `/account/get_forge_rates` | SQLite; 旧静态库:account__get_forge_rates.json | 未见直接会话检查 | `ForgeRates` / `player_state.go` | ✓ / ✓ | 功能 |
| `free_gifts` | `/service/relationship/api/v1/free_gifts/get_pending`（别名推断） | SQLite; 旧静态库:service__relationship__api__v1__free_gifts__get_pending.json | 见直接会话检查 | `FreeGiftsGetPending` / `messaging_routes.go` | — / — | 功能 |
| `generic_shop_defs` | `/account/get_generic_shops` | 代码常量或辅助函数，待逐分支核对; 旧静态库:account__get_generic_shops.json | 未见直接会话检查 | `GenericShops` / `legacy_config.go` | ✓ / ✓ | 功能 |
| `get_buff_defs` | `/account/buff/get_buff_defs` | game_config_or_embedded_json:account__buff__get_buff_defs.json; 旧静态库:account__buff__get_buff_defs.json | 未见直接会话检查 | `BuffDefinitions` / `magic.go` | ✓ / ✓ | 功能 |
| `get_buffs` | `/account/buff/get_buffs1` | 代码常量或辅助函数，待逐分支核对; 旧静态库:account__buff__get_buffs1.json | 见直接会话检查 | `GetBuffs1` / `magic.go` | ✓ / ✓ | 功能 |
| `get_consumable_defs` | `/account/consumable/get_consumable_defs` | game_config_or_embedded_json:account__consumable__get_consumable_defs.json; 旧静态库:account__consumable__get_consumable_defs.json | 未见直接会话检查 | `ConsumableDefinitions` / `magic.go` | ✓ / ✓ | 功能 |
| `get_consumables` | `/account/consumable/get_consumables1` | 代码常量或辅助函数，待逐分支核对; 旧静态库:account__consumable__get_consumables1.json | 见直接会话检查 | `GetConsumables1` / `magic.go` | ✓ / ✓ | 功能 |
| `get_lootboxes` | `/account/lootboxes/get` | 代码常量或辅助函数，待逐分支核对; 旧静态库:account__lootboxes__get.json | 见直接会话检查 | `GetLootboxes` / `commerce_read.go` | ✓ / ✓ | 功能 |
| `get_map_defs` | `/account/get_map_defs` | game_config_or_embedded_json:map.json; 旧静态库:account__get_map_defs.json | 未见直接会话检查 | `MapDefinitions` / `profile.go` | — / — | 启动链 |
| `get_shop` | `/account/get_shop` | SQLite; 旧静态库:account__get_shop.json | 见直接会话检查 | `GetShop` / `commerce_read.go` | ✓ / ✓ | 启动链 |
| `get_star_tag_defs` | `/account/star/get` | SQLite; embedded_legacydata; 旧静态库:account__star__get.json | 见直接会话检查 | `StarTags` / `relationships_batch5.go` | ✓ / ✓ | 功能 |
| `get_user_profile` | 无独立入口证据 | 未找到独立数据来源 | 未见直接会话检查 | 无已知处理 | — / — | 待确认 |
| `gift_messages` | `/account/get_pending_messages`（别名推断） | SQLite; 旧静态库:account__get_pending_messages.json | 未见直接会话检查 | `GetPendingMessages` / `messaging_routes.go` | ✓ / ✓ | 功能 |
| `iap_list` | `/account/iaplist` | game_config_or_embedded_json:iaplist.json; 旧静态库:account__iaplist.json | 未见直接会话检查 | `IAPList` / `legacy_config.go` | ✓ / ✓ | 功能 |
| `infractions` | `/service/infraction/api/v1/get_infractions_status_decorated`（别名推断） | SQLite; 旧静态库:service__infraction__api__v1__get_infractions_status_decorated.json | 见直接会话检查 | `InfractionStatus` / `batch1_database.go` | — / — | 功能 |
| `invite_list` | `/account/get_invites`（别名推断） | SQLite; 旧静态库:account__get_invites.json | 见直接会话检查 | `Invites` / `legacy_config.go` | ✓ / ✓ | 功能 |
| `messages` | `/service/message/api/v2/get_all` | 代码常量或辅助函数，待逐分支核对; 旧静态库:service__message__api__v2__get_all.json | 未见直接会话检查 | `MessageGetAllV2` / `messaging_routes.go` | — / — | 功能 |
| `motd` | `/account/get_motd` | game_config_or_embedded_json:motd_config.json; 旧静态库:account__get_motd.json | 未见直接会话检查 | `Motd` / `bootstrap.go` | ✓ / ✓ | 启动链 |
| `online_friends` | `/account/get_online_friends` | SQLite; 旧静态库:account__get_online_friends.json | 见直接会话检查 | `OnlineFriends` / `constellations.go` | ✓ / ✓ | 功能 |
| `outfit_defs` | 无独立入口证据 | 未找到独立数据来源 | 未见直接会话检查 | 无已知处理 | — / — | 待确认 |
| `questionnaires` | `/account/get_questionnaires`（别名推断） | 代码常量或辅助函数，待逐分支核对 | 见直接会话检查 | `Questionnaires` / `player_state.go` | — / — | 功能 |
| `relationship_defs` | `/account/get_relationship_abilities` | embedded_legacydata; 旧静态库:account__get_relationship_abilities.json | 未见直接会话检查 | `RelationshipDefinitions` / `player_state.go` | ✓ / ✓ | 功能 |
| `serendipity_matches` | `/account/serendipity/get` | SQLite; 旧静态库:account__serendipity__get.json | 见直接会话检查 | `Serendipity` / `relationships_batch5.go` | ✓ / ✓ | 功能 |
| `set_app_badge_number` | `/account/get_app_badge_number` | SQLite; 旧静态库:account__get_app_badge_number.json | 见直接会话检查 | `Badge` / `profile.go` | ✓ / ✓ | 功能 |
| `set_friend_statues` | `/account/get_friend_statues` | SQLite; 旧静态库:account__get_friend_statues.json | 见直接会话检查 | `FriendStatues` / `constellations.go` | ✓ / ✓ | 功能 |
| `set_friend_status` | `/account/get_friend_statues`（别名推断） | SQLite; 旧静态库:account__get_friend_statues.json | 见直接会话检查 | `FriendStatues` / `constellations.go` | ✓ / ✓ | 功能 |
| `set_friends` | `/account/get_friends` | SQLite; 旧静态库:account__get_friends.json | 见直接会话检查 | `Friends` / `friends.go` | ✓ / ✓ | 功能 |
| `set_random` | `/account/get_rank`（别名推断） | SQLite; 旧静态库:account__get_rank.json | 见直接会话检查 | `Rank` / `data.go` | ✓ / ✓ | 功能 |
| `set_rank` | `/account/get_rank` | SQLite; 旧静态库:account__get_rank.json | 见直接会话检查 | `Rank` / `data.go` | ✓ / ✓ | 功能 |
| `set_world_quests` | `/account/get_account_world_quests` | SQLite; 旧静态库:account__get_account_world_quests.json | 见直接会话检查 | `WorldQuests` / `quests_account.go` | ✓ / ✓ | 启动链 |
| `spirit_shops` | `/account/get_spirit_shops` | game_config_or_embedded_json:account__get_spirit_shops.json,game_control.json,spirit_shops.json; 旧静态库:account__get_spirit_shops.json | 未见直接会话检查 | `SpiritShops` / `batch1_spirit.go` | ✓ / ✓ | 启动链 |
| `status_unlocks` | `/service/status/api/v1/get_unlocks` | game_config_or_embedded_json:all_unlock_status.json,auto_full_unlock.json,kaiquantu.json; SQLite; 旧静态库:service__status__api__v1__get_unlocks.json | 见直接会话检查 | `StatusUnlocks` / `status_unlocks.go` | ✓ / ✓ | 功能 |
| `unlocks` | `/account/get_unlocks` | game_config_or_embedded_json:auto_full_unlock.json; SQLite; 旧静态库:account__get_unlocks.json | 见直接会话检查 | `Unlocks` / `unlocks.go` | ✓ / ✓ | 启动链 |
| `userdata` | `/account/userdata` | SQLite; 旧静态库:account__userdata.json | 见直接会话检查 | `Userdata` / `data.go` | ✓ / ✓ | 启动链 |
| `wing_buffs` | `/account/wing_buffs/get` | game_config_or_embedded_json:auto_full_unlock.json; 旧静态库:account__wing_buffs__get.json | 未见直接会话检查 | `GetWingBuffs` / `wing_buffs.go` | ✓ / ✓ | 功能 |

## 已解释的名称差异与缺口

- `set_friend_status`、`set_random` 只在旧版及 Go 的 `auth_create` 列表中出现；已知候选接口分别返回 `set_friend_statues`、`set_rank`。这两个资源名不能据此判定为成功加载。
- `invite_list`、`gift_messages`、`free_gifts`、`infractions`、`questionnaires` 的已知业务入口与列表名不是相同响应键；表内候选仅用于追踪，客户端映射待抓取。`questionnaires` 当前 Go 只返回 `result=ok`，是旧占位语义。
- `outfit_defs`、`get_user_profile`、`emote_sequences` 在旧静态响应键及注册路由中都找不到独立资源入口；目前作为显式待确认依赖，不以成功占位掩盖。
- `get_shop`、`spirit_shops`、`event_schedule`、`motd`、`unlocks`、魔法定义等配置依赖要在 Task 3 分别测试正常、缺失、损坏及版本差异。现有日志中的 HTTP 200 不代表客户端接受了内容。
