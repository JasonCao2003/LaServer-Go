# 登录资源依赖基线

从四种旧认证响应的 resource_list 提取，不代表每个客户端都会请求全部资源。路径是根据服务端名称建立的候选映射；实际请求顺序、版本和阻断点必须以客户端记录验证，不能把 501 自动等同于整次登录失败。

| 资源名 | 候选请求路径 | 当前实现 |
| --- | --- | --- |
| `device_capabilities` | `/account/device_capabilities/lookup` | implemented_local_only |
| `userdata` | `/account/userdata` | implemented_local_only |
| `set_friends` | `/account/get_friends` | implemented_local_only |
| `set_friend_statues` | `/account/get_friend_statues` | implemented_local_only |
| `online_friends` | `/account/get_online_friends` | implemented_local_only |
| `set_rank` | `/account/get_rank` | implemented_local_only |
| `currency` | `/account/get_currency` | implemented_local_only |
| `forge_rates` | `/account/get_forge_rates` | implemented_local_only |
| `motd` | `/account/get_motd` | implemented_local_only |
| `outfit_defs` | `/account/get_outfit` | implemented_local_only |
| `achievements` | `/account/get_achievements` | implemented_local_only |
| `achievement_stats` | `/account/get_achievement_stats` | implemented_local_only |
| `relationship_defs` | `/account/get_relationship_abilities` | implemented_local_only |
| `questionnaires` | `/account/get_questionnaires` | implemented_local_only |
| `event_schedule` | `/account/get_event_schedule` | implemented_local_only |
| `set_world_quests` | `/account/get_account_world_quests` | pending |
| `iap_list` | `/account/iaplist` | implemented_local_only |
| `invite_list` | `/account/get_invites` | implemented_local_only |
| `unlocks` | `/account/get_unlocks` | implemented_local_only |
| `get_shop` | `/account/get_shop` | pending |
| `spirit_shops` | `/account/get_spirit_shops` | implemented_local_only |
| `collectibles` | `/account/get_collectibles` | implemented_local_only |
| `wing_buffs` | `/account/wing_buffs/get` | pending |
| `gift_messages` | `/account/get_pending_messages` | pending |
| `set_app_badge_number` | `/account/get_app_badge_number` | implemented_local_only |
| `get_consumable_defs` | `/account/consumable/get_consumable_defs` | pending |
| `get_consumables` | `/account/consumable/get_consumables` | pending |
| `get_buff_defs` | `/account/buff/get_buff_defs` | pending |
| `get_buffs` | `/account/buff/get_buffs` | pending |
| `get_lootboxes` | `/account/lootboxes/get` | pending |
| `checkpoints` | `/account/get_checkpoints` | implemented_local_only |
| `generic_shop_defs` | `/account/get_generic_shops` | implemented_local_only |
| `external_account_friends` | `/account/get_external_friends` | pending |
| `serendipity_matches` | `/account/serendipity/get` | pending |
| `get_star_tag_defs` | `/account/star/get` | pending |
| `status_unlocks` | `/service/status/api/v1/get_unlocks` | pending |
| `free_gifts` | `/service/relationship/api/v1/free_gifts/get_pending` | pending |
| `get_map_defs` | `/account/get_map_defs` | implemented_local_only |
| `achievement_defs` | `/account/get_achievement_defs` | implemented_local_only |
| `achievement_stats_tracking` | `/account/get_achievement_stats_tracking` | implemented_local_only |
| `event_currency_defs` | `/account/get_event_currency_defs` | implemented_local_only |
| `infractions` | `/service/infraction/api/v1/get_infractions_status_decorated` | implemented_local_only |
| `messages` | `/service/message/api/v2/get_all` | pending |
| `set_friend_status` | 待定位 | unresolved |
| `set_random` | 待定位 | unresolved |
| `external_links` | 待定位 | unresolved |
| `get_user_profile` | 待定位 | unresolved |
| `emote_sequences` | 待定位 | unresolved |

优先补齐：装扮 → 解锁/光翼 → 好友/邀请 → 任务 → 魔法和 buff 定义。该顺序是开发建议，不是已观测的客户端请求顺序。

set_friend_status、set_random、external_links、get_user_profile、emote_sequences 等无法由现有路径直接证明的资源保留 unresolved；不能根据相似名字擅自返回成功。
