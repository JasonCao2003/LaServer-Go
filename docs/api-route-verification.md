# 游戏 API 路由验证台账

基线：2026-11-12。A 批共 77 条：原有基础路由 42 条、第 1 批 15 条、第 2 批 20 条。B–D 批的逐分支证据仍按 [测试计划](superpowers/plans/2026-11-12-api-test-coverage.md) 逐项积累；C 批入口覆盖进展见下文。

本表的“测试引用”仅证明测试源码直接调用该路径；不能推断所有认证、版本、错误分支或客户端行为均已覆盖。A 批新增的 `batch_a_test.go` 检查入口状态、关键响应字段，以及邀请码、状态解锁、光翼等状态闭环。真实 APK 验收均未执行。

C 批入口核对（2026-11-13）：原第 5–8 批的 78 条 POST 路径均在 `batch_c_test.go` 或 `batch_c_remaining_test.go` 中直接请求；`GET /ugc/...` 及两个 OPTIONS 入口也已请求。关键状态与失败分支见 [C 批计划进展](superpowers/plans/2026-11-12-api-test-coverage.md)。聊天跨 API 实例投递、room-server 在线状态、APK 客户端和各路径完整分支矩阵仍待验收；旧占位接口只记录响应兼容。

D 批入口核对（2026-11-13）：第 9 批 6 条 POST 路径及第 10 批 3 条管理路径的 GET/POST 均在 `batch_d_test.go` 中直接请求；舞台持久化、越权与故障，以及管理鉴权、双向状态、审计和部分失败均有状态断言。预约、公寓接口按旧占位响应记录；真实房间和反向代理部署未验收。

| 组 | 路由 | 方法 | 旧处理函数 | 测试引用 | 后续验证 |
| --- | --- | --- | --- | --- | --- |
| 基础 | `/account/auth/create` | POST | `create_account` | config_test.go | 逐分支及客户端 |
| 基础 | `/account/create` | POST | `account_create` | api_test.go, config_test.go, policies_test.go | 逐分支及客户端 |
| 基础 | `/account/get_motd` | POST | `get_motd` | policies_test.go | 逐分支及客户端 |
| 基础 | `/account/auth/login` | POST | `logini` | config_test.go, login_compat_test.go | 逐分支及客户端 |
| 基础 | `/account/login` | POST | `account_login` | api_test.go, policies_test.go | 逐分支及客户端 |
| 基础 | `/account/get_currency` | POST | `get_currency` | api_test.go | 逐分支及客户端 |
| 基础 | `/account/get_rank` | POST | `get_rank` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/userdata` | POST | `get_userdata` | api_test.go | 逐分支及客户端 |
| 基础 | `/account/sync_user_data` | POST | `sync_user_data` | api_test.go | 逐分支及客户端 |
| 基础 | `/account/device_capabilities/lookup` | POST | `lookup` | api_test.go | 逐分支及客户端 |
| 基础 | `/account/get_latest_build_version` | POST | `get_latest_build_version` | api_test.go | 逐分支及客户端 |
| 基础 | `/account/get_vars` | POST | `get_vars` | api_test.go | 逐分支及客户端 |
| 基础 | `/account/support/has_reply` | POST | `has_reply` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_achievement_stats_tracking` | POST | `get_achievement_stats_tracking` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_event_currency_defs` | POST | `get_event_currency_defs` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_questionnaires` | POST | `get_questionnaires` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_generic_shops` | POST | `generic_shop_defs` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_forge_rates` | POST | `get_forge_rates` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_achievements` | POST | `get_achievements` | profile_test.go | 逐分支及客户端 |
| 基础 | `/account/get_achievement_stats` | POST | `get_achievement_stats` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_relationship_abilities` | POST | `get_relationship_abilities` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_unlocks` | POST | `get_unlocks` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/iaplist` | POST | `iap_list` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_collectibles` | POST | `get_collectibles` | policies_test.go, profile_test.go | 逐分支及客户端 |
| 基础 | `/account/get_questionnaire_groups` | POST | `get_questionnaire_groups` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_app_badge_number` | POST | `get_app_badge_number` | profile_test.go | 逐分支及客户端 |
| 基础 | `/account/get_checkpoints` | POST | `get_checkpoints` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_invites` | POST | `get_invites` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/get_online_friends` | POST | `get_online_friends` | friends_test.go | 逐分支及客户端 |
| 基础 | `/account/get_achievement_defs` | POST | `get_achievement_defs` | profile_test.go | 逐分支及客户端 |
| 基础 | `/account/get_friend_statues` | POST, OPTIONS | `get_friend_statues` | friends_test.go | 逐分支及客户端 |
| 基础 | `/account/get_friends` | POST | `get_friends` | friends_test.go | 逐分支及客户端 |
| 基础 | `/account/get_map_defs` | POST | `get_map_defs` | profile_test.go | 逐分支及客户端 |
| 基础 | `/account/get_outfit` | POST | `get_outfit` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/find_previous_or_empty` | POST | `find_previous_or_empty` | api_test.go | 逐分支及客户端 |
| 基础 | `/account/quest/get_quest_defs` | POST | `get_quest_defs` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/set_checkpoint` | POST | `set_checkpoint` | api_test.go | 逐分支及客户端 |
| 基础 | `/account/create_invite` | POST | `create_invite` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/check_invite` | POST | `check_invite` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/accept_invite` | POST | `accept_invite` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/delete_invite` | POST | `delete_invite` | batch_a_test.go | 逐分支及客户端 |
| 基础 | `/account/ack_unlock` | POST | `ack_unlock` | batch_a_test.go | 逐分支及客户端 |
| 第 1 批 | `/account/get_surveys` | POST | `get_surveys` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/get_event_schedule` | POST | `get_event_schedule` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/get_spirit_shops` | POST | `get_spirit_shops` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/get_currency_defs` | POST | `get_currency_defs` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/trust/get_trust_tier_defs` | POST | `get_trust_tier_defs` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/get_video_defs` | POST | `get_video_defs` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/get_radiance_spawn_defs` | POST | `get_radiance_spawn_defs` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/get_level_pickups` | POST | `get_level_pickups` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/trust/get_trust_group_defs` | POST | `get_trust_group_defs` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/remote_config/upload` | POST | `upload_remote_config` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/trust/get_trust_activity_defs` | POST | `get_trust_activity_defs` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/account/get_generic_shop_item_tracking_info` | POST | `get_generic_shop_item_tracking_info` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/service/auth/api/v1/token` | POST | `get_auth_token` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/service/infraction/api/v1/get_infractions_status_decorated` | POST | `get_infractions_status_decorated` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 1 批 | `/starwatch/auth/api/v1/auth` | POST | `auth` | batch_a_test.go | 版本/错误分支及客户端 |
| 第 2 批 | `/account/consumable/get_consumable_defs` | POST | `get_consumable_defs` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/consumable/get_consumables` | POST | `get_consumables` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/consumable/consume_item1` | POST | `consume_item1` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/wing_buffs/get` | POST | `get_wing_buffs` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/wing_buffs/collect` | POST | `wing_buffs_collect` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/collect_collectible` | POST | `collect_collectible` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/wing_buffs/drop` | POST | `wing_buffs_drop` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/wing_buffs/deposit` | POST | `wing_buffs_deposit` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/wing_buffs/convert` | POST | `wing_buffs_convert` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/buff/get_buffs1` | POST | `get_buffs1` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/buff/get_buffs` | POST | `get_buffs` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/buff/remove_buff1` | POST | `remove_buff1` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/buff/remove_buff` | POST | `remove_buff1` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/consumable/get_consumables1` | POST | `get_consumables1` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/ack_collectible` | POST | `ack_collectible` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/account/buff/get_buff_defs` | POST | `get_buff_defs` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/service/status/api/v1/add_unlock` | POST | `add_unlock` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/service/status/api/v1/delete_unlock` | POST | `delete_unlock` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/service/status/api/v1/get_unlocks` | POST | `get_unlocks_status` | batch_a_test.go | 重复/跨账号/库存及客户端 |
| 第 2 批 | `/service/inventory/api/v1/unlocks/remove_dye` | POST | `inventory_remove_dye` | batch_a_test.go | 重复/跨账号/库存及客户端 |

## 本轮结果

- A 批 77/77 条路径有直接测试引用；这是入口覆盖，不是逐分支合同验收。
- 新增配置缺失/损坏/有效三态资源检查，以及 0.1.8、0.3.11 与 0.19.5 的魔法定义版本过滤检查。魔法库存不足仍消耗的缺陷已通过失败用例确认并修复，重复消耗与跨账号请求通过。
- `TestBatchAResourceRouteContracts` 逐条核对第 1 批 15 条入口；`TestBatchAStateRouteContracts` 逐条核对第 2 批 20 条入口。基础 42 条由既有测试和新增 `TestBatchABaselineReadContracts`、`TestBatchAInviteLifecycleAndAck` 覆盖入口。
- 配置缺失的 503 是预期结果；配置完整时的内容、版本分流和真实客户端请求仍需另测。
- 失败或差异必须更新本表和 `docs/TODO.md`，不得仅依据路径存在标记为通过。
