# api-server 文件与函数说明

> 由 `scripts/docs/generate_api_code_map.py` 从源码注释生成。
> 范围：`services/api-server` 下全部非测试 Go 文件；函数说明取自函数上方的一行中文注释。
> 测试文件（`_test.go`）未包含。文件与函数命名调整记录见 [api-server 文件整理记录](api-server-refactor-2026-11-24.md)。

## 统计

| 目录 | 文件数 | 函数数 |
| --- | ---: | ---: |
| `cmd/legacy-import` | 1 | 1 |
| `cmd/server` | 1 | 2 |
| `internal/app` | 1 | 1 |
| `internal/config` | 1 | 4 |
| `internal/handler` | 13 | 22 |
| `internal/legacyimport` | 1 | 18 |
| `internal/middleware` | 2 | 6 |
| `internal/model` | 73 | 72 |
| `internal/repository` | 19 | 115 |
| `internal/service` | 1 | 1 |
| `internal/service/account` | 8 | 44 |
| `internal/service/commerce` | 15 | 53 |
| `internal/service/content` | 7 | 46 |
| `internal/service/core` | 13 | 60 |
| `internal/service/inventory` | 8 | 59 |
| `internal/service/legacydata` | 1 | 2 |
| `internal/service/messaging` | 4 | 62 |
| `internal/service/quests` | 5 | 47 |
| `internal/service/relationships` | 7 | 51 |
| `internal/service/social` | 6 | 102 |
| `internal/service/stage` | 2 | 11 |
| `migrations` | 1 | 0 |
| **合计** | **190** | **779** |

## cmd/legacy-import

### `cmd/legacy-import/main.go`

功能：旧数据导入命令行入口

| 函数 | 作用 |
| --- | --- |
| `main` | 程序入口 |

## cmd/server

### `cmd/server/main.go`

功能：api-server 启动入口

| 函数 | 作用 |
| --- | --- |
| `main` | 程序入口 |
| `run` | 启动服务主流程 |

## internal/app

### `internal/app/run.go`

功能：服务依赖组装与生命周期管理

| 函数 | 作用 |
| --- | --- |
| `Run` | 运行服务主流程 |

## internal/config

### `internal/config/config.go`

功能：服务配置加载与校验

| 函数 | 作用 |
| --- | --- |
| `Load` | 加载并解析服务配置 |
| `currencyField` | 判断是否为货币字段 |
| `ValidateRuntime` | 校验运行时配置 |
| `validateRoomAdmission` | 校验房间准入配置 |

## internal/handler

### `internal/handler/http_helpers.go`

功能：健康检查、UGC 静态文件与 CORS 预检处理

| 函数 | 作用 |
| --- | --- |
| `health` | 返回健康检查响应 |
| `serveUGC` | 提供 UGC 文件下载 |
| `preflight` | 生成 CORS 预检响应 |

### `internal/handler/request.go`

功能：HTTP 请求解析与统一 JSON 响应/错误输出

| 函数 | 作用 |
| --- | --- |
| `post` | 解析 POST 请求并交给服务处理 |
| `adminQuery` | 把查询参数交给管理接口处理 |
| `respond` | 调用服务并写出 JSON 响应 |
| `write` | 写出 JSON 响应 |
| `remoteIP` | 返回请求来源 IP |

### `internal/handler/router.go`

功能：注册全部业务路由入口

| 函数 | 作用 |
| --- | --- |
| `registerRoutes` | 注册全部业务路由 |

### `internal/handler/routes_account.go`

功能：account 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerAccount` | 注册 account 服务处理的路径与方法。 |

### `internal/handler/routes_commerce.go`

功能：commerce 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerCommerce` | 注册 commerce 服务处理的路径与方法。 |

### `internal/handler/routes_content.go`

功能：content 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerContent` | 注册 content 服务处理的路径与方法。 |

### `internal/handler/routes_inventory.go`

功能：inventory 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerInventory` | 注册 inventory 服务处理的路径与方法。 |

### `internal/handler/routes_messaging.go`

功能：messaging 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerMessaging` | 注册 messaging 服务处理的路径与方法。 |

### `internal/handler/routes_quests.go`

功能：quests 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerQuests` | 注册 quests 服务处理的路径与方法。 |

### `internal/handler/routes_relationships.go`

功能：relationships 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerRelationships` | 注册 relationships 服务处理的路径与方法。 |

### `internal/handler/routes_social.go`

功能：social 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerSocial` | 注册 social 服务处理的路径与方法。 |

### `internal/handler/routes_stage.go`

功能：stage 业务 HTTP 路由。

| 函数 | 作用 |
| --- | --- |
| `registerStage` | 注册 stage 服务处理的路径与方法。 |

### `internal/handler/server.go`

功能：API 服务构造：创建 Gin 引擎并装配路由

| 函数 | 作用 |
| --- | --- |
| `NewAPI` | 创建 API 实例并配置 Gin 引擎 |
| `API.ServeHTTP` | 提供 HTTP 处理入口 |
| `API.RuntimeConfig` | 返回运行时配置 |
| `API.Engine` | 返回底层路由引擎 |

## internal/legacyimport

### `internal/legacyimport/import.go`

功能：旧数据库预检与导入实现

| 函数 | 作用 |
| --- | --- |
| `quoted` | 转义 SQLite 标识符 |
| `readonlyURI` | 生成旧库只读 URI |
| `schemaTables` | 读取数据库表清单 |
| `tableColumns` | 读取表字段信息 |
| `tableForeignKeys` | 读取表外键信息 |
| `sidecars` | 检查旧库附属文件 |
| `ensureOutsideLegacy` | 确保目标不在旧实例目录内 |
| `migrateMemory` | 创建迁移用内存库 |
| `Preflight` | 预检旧数据库并生成报告 |
| `Import` | 导入旧数据库到新目标 |
| `ImportWithOptions` | 按选项导入旧数据库 |
| `copyInto` | 复制旧库数据到目标 |
| `reportHasLoss` | 判断报告是否存在数据丢失 |
| `reportHasLossExceptPlannedRecordings` | 判断除计划留影外是否存在数据丢失 |
| `importFeedExtras` | 导入动态附加数据 |
| `copyRecordings` | 复制留影文件 |
| `copyTable` | 复制单张表数据 |
| `compareKeyValues` | 校验导入后的键值一致性 |

## internal/middleware

### `internal/middleware/access.go`

功能：访问控制与来源校验中间件

| 函数 | 作用 |
| --- | --- |
| `statusWriter.WriteHeader` | 记录状态码并写出响应头 |
| `statusWriter.Write` | 记录字节数并写出响应体 |
| `statusWriter.Unwrap` | 返回底层响应对象 |
| `Access` | 记录访问日志的中间件 |

### `internal/middleware/body.go`

功能：请求体读取与大小限制中间件

| 函数 | 作用 |
| --- | --- |
| `BodyError.Error` | 返回错误文本 |
| `ReadObject` | 读取并解析请求体 |

## internal/model

### `internal/model/achievement.go`

功能：成就记录数据模型

| 函数 | 作用 |
| --- | --- |
| `Achievement.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/achievement_def.go`

功能：成就定义数据模型

| 函数 | 作用 |
| --- | --- |
| `AchievementDef.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/achievement_stat.go`

功能：成就统计数据模型

| 函数 | 作用 |
| --- | --- |
| `AchievementStat.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/achievement_stats.go`

功能：成就统计集合数据模型

| 函数 | 作用 |
| --- | --- |
| `AchievementStats.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/app_badge.go`

功能：应用徽章数据模型

| 函数 | 作用 |
| --- | --- |
| `AppBadge.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/boat_follow.go`

功能：船跟随关系数据模型

| 函数 | 作用 |
| --- | --- |
| `BoatFollow.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/collectible.go`

功能：收集品数据模型

| 函数 | 作用 |
| --- | --- |
| `Collectible.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/color_dye_currency.go`

功能：染料货币数据模型

| 函数 | 作用 |
| --- | --- |
| `ColorDyeCurrency.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/constellation_friend_page.go`

功能：好友星座页面数据模型

| 函数 | 作用 |
| --- | --- |
| `ConstellationFriendPage.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/consumable_def.go`

功能：消耗品定义数据模型

| 函数 | 作用 |
| --- | --- |
| `ConsumableDef.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/default_unlock_config.go`

功能：默认解锁配置数据模型

| 函数 | 作用 |
| --- | --- |
| `DefaultUnlockConfig.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/doc.go`

功能：数据模型包：与旧 SQLite 表结构对应的持久化记录

无顶层函数。

### `internal/model/event_currency_def.go`

功能：活动货币定义数据模型

| 函数 | 作用 |
| --- | --- |
| `EventCurrencyDef.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/event_schedule.go`

功能：活动日程数据模型

| 函数 | 作用 |
| --- | --- |
| `EventSchedule.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/external_account_friend.go`

功能：外部账号好友数据模型

| 函数 | 作用 |
| --- | --- |
| `ExternalAccountFriend.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/follow_relationship.go`

功能：关注关系数据模型

| 函数 | 作用 |
| --- | --- |
| `FollowRelationship.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/forge_rate.go`

功能：锻造概率数据模型

| 函数 | 作用 |
| --- | --- |
| `ForgeRate.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/free_gift.go`

功能：免费礼物数据模型

| 函数 | 作用 |
| --- | --- |
| `FreeGift.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/friend_status.go`

功能：好友状态数据模型

| 函数 | 作用 |
| --- | --- |
| `FriendStatus.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/friendship.go`

功能：好友关系数据模型

| 函数 | 作用 |
| --- | --- |
| `Friendship.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/game_server.go`

功能：游戏服务器数据模型

| 函数 | 作用 |
| --- | --- |
| `GameServer.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/generic_shop_def.go`

功能：通用商店定义数据模型

| 函数 | 作用 |
| --- | --- |
| `GenericShopDef.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/iap_product.go`

功能：内购商品数据模型

| 函数 | 作用 |
| --- | --- |
| `IAPProduct.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/infraction.go`

功能：违规记录数据模型

| 函数 | 作用 |
| --- | --- |
| `Infraction.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/invite.go`

功能：邀请数据模型

| 函数 | 作用 |
| --- | --- |
| `Invite.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/item_dye.go`

功能：物品染料数据模型

| 函数 | 作用 |
| --- | --- |
| `ItemDye.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/level_pickup.go`

功能：关卡拾取数据模型

| 函数 | 作用 |
| --- | --- |
| `LevelPickup.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/lootbox.go`

功能：战利品箱数据模型

| 函数 | 作用 |
| --- | --- |
| `Lootbox.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/map_definition.go`

功能：地图定义数据模型

| 函数 | 作用 |
| --- | --- |
| `MapDefinition.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/message.go`

功能：消息数据模型

| 函数 | 作用 |
| --- | --- |
| `Message.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/player_quest.go`

功能：玩家任务数据模型

| 函数 | 作用 |
| --- | --- |
| `PlayerQuest.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/quest_stat.go`

功能：任务统计数据模型

| 函数 | 作用 |
| --- | --- |
| `QuestStat.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/request_log.go`

功能：请求日志数据模型

| 函数 | 作用 |
| --- | --- |
| `RequestLog.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/resource_token.go`

功能：资源令牌数据模型

| 函数 | 作用 |
| --- | --- |
| `ResourceToken.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/response_override.go`

功能：响应覆盖数据模型

| 函数 | 作用 |
| --- | --- |
| `ResponseOverride.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/season_quest.go`

功能：赛季任务数据模型

| 函数 | 作用 |
| --- | --- |
| `SeasonQuest.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/season_quest_reward.go`

功能：赛季任务奖励数据模型

| 函数 | 作用 |
| --- | --- |
| `SeasonQuestReward.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/season_unlock.go`

功能：赛季解锁数据模型

| 函数 | 作用 |
| --- | --- |
| `SeasonUnlock.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/serendipity_match.go`

功能：缘分匹配数据模型

| 函数 | 作用 |
| --- | --- |
| `SerendipityMatch.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/server_assignment.go`

功能：服务器分配数据模型

| 函数 | 作用 |
| --- | --- |
| `ServerAssignment.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/shop_cost_currency.go`

功能：商店货币消耗数据模型

| 函数 | 作用 |
| --- | --- |
| `ShopCostCurrency.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/shop_item.go`

功能：商店物品数据模型

| 函数 | 作用 |
| --- | --- |
| `ShopItem.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/shop_unlock_requirement.go`

功能：商店解锁条件数据模型

| 函数 | 作用 |
| --- | --- |
| `ShopUnlockRequirement.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/social_feed_like.go`

功能：社交动态点赞数据模型

| 函数 | 作用 |
| --- | --- |
| `SocialFeedLike.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/social_feed_post.go`

功能：社交动态帖子数据模型

| 函数 | 作用 |
| --- | --- |
| `SocialFeedPost.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/stage.go`

功能：舞台数据模型

| 函数 | 作用 |
| --- | --- |
| `Stage.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/star_tag_def.go`

功能：星标定义数据模型

| 函数 | 作用 |
| --- | --- |
| `StarTagDef.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/star_tag_link.go`

功能：星标关联数据模型

| 函数 | 作用 |
| --- | --- |
| `StarTagLink.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/status_unlock.go`

功能：状态解锁数据模型

| 函数 | 作用 |
| --- | --- |
| `StatusUnlock.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/status_unlock_deleted.go`

功能：已删除状态解锁数据模型

| 函数 | 作用 |
| --- | --- |
| `StatusUnlockDeleted.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/survey.go`

功能：问卷数据模型

| 函数 | 作用 |
| --- | --- |
| `Survey.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/trust_activity_def.go`

功能：信任活动定义数据模型

| 函数 | 作用 |
| --- | --- |
| `TrustActivityDef.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/trust_group_def.go`

功能：信任分组定义数据模型

| 函数 | 作用 |
| --- | --- |
| `TrustGroupDef.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/unlock_item.go`

功能：解锁项数据模型

| 函数 | 作用 |
| --- | --- |
| `UnlockItem.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user.go`

功能：用户账号数据模型

| 函数 | 作用 |
| --- | --- |
| `User.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_activity.go`

功能：用户活动数据模型

| 函数 | 作用 |
| --- | --- |
| `UserActivity.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_buff.go`

功能：用户增益数据模型

| 函数 | 作用 |
| --- | --- |
| `UserBuff.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_checkpoint.go`

功能：用户检查点数据模型

| 函数 | 作用 |
| --- | --- |
| `UserCheckpoint.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_consumable.go`

功能：用户消耗品数据模型

| 函数 | 作用 |
| --- | --- |
| `UserConsumable.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_currency.go`

功能：用户货币数据模型

| 函数 | 作用 |
| --- | --- |
| `UserCurrency.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_data.go`

功能：用户存档数据数据模型

| 函数 | 作用 |
| --- | --- |
| `UserData.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_initialization.go`

功能：用户初始化数据模型

| 函数 | 作用 |
| --- | --- |
| `UserInitialization.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_online_status.go`

功能：用户在线状态数据模型

| 函数 | 作用 |
| --- | --- |
| `UserOnlineStatus.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_outfit.go`

功能：用户装扮数据模型

| 函数 | 作用 |
| --- | --- |
| `UserOutfit.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_pickup.go`

功能：用户拾取数据模型

| 函数 | 作用 |
| --- | --- |
| `UserPickup.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_rank.go`

功能：用户排行数据模型

| 函数 | 作用 |
| --- | --- |
| `UserRank.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_survey.go`

功能：用户问卷数据模型

| 函数 | 作用 |
| --- | --- |
| `UserSurvey.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_trust_group.go`

功能：用户信任分组数据模型

| 函数 | 作用 |
| --- | --- |
| `UserTrustGroup.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_unlocks.go`

功能：用户解锁数据模型

| 函数 | 作用 |
| --- | --- |
| `UserUnlocks.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/user_wing_buff.go`

功能：用户光翼增益数据模型

| 函数 | 作用 |
| --- | --- |
| `UserWingBuff.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/version.go`

功能：版本信息数据模型

| 函数 | 作用 |
| --- | --- |
| `Version.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/video_definition.go`

功能：视频定义数据模型

| 函数 | 作用 |
| --- | --- |
| `VideoDefinition.TableName` | 返回旧版 SQLite 表名 |

### `internal/model/video_schedule.go`

功能：视频日程数据模型

| 函数 | 作用 |
| --- | --- |
| `VideoSchedule.TableName` | 返回旧版 SQLite 表名 |

## internal/repository

### `internal/repository/admin_friends.go`

功能：管理后台好友数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.AdminFriendIDs` | 读取管理员好友 ID |
| `Store.AdminFriendRow` | 读取管理员好友记录 |
| `Store.AdminSetFriendship` | 设置管理员好友关系 |

### `internal/repository/auth_views.go`

功能：鉴权视图查询

无顶层函数。

### `internal/repository/commerce.go`

功能：商城订单与商品数据访问

| 函数 | 作用 |
| --- | --- |
| `CurrencyRead` | 构造读取货币的 SQL 语句 |
| `Tx.SpendCurrency` | 扣减货币 |
| `Tx.GrantCurrency` | 发放货币 |

### `internal/repository/constellations.go`

功能：星座好友数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.Statues` | 读取好友状态列表 |
| `Store.Constellations` | 读取星座列表 |

### `internal/repository/feed.go`

功能：社交动态数据访问

| 函数 | 作用 |
| --- | --- |
| `scanFeed` | 扫描动态 |
| `Store.CreateFeed` | 创建动态 |
| `Store.FeedByID` | 按 ID 获取动态 |
| `Store.Feeds` | 处理动态 |
| `Store.FeedsByIDs` | 按 ID 列表获取动态 |
| `Store.DeleteFeed` | 删除动态 |
| `Store.UpdateFeed` | 更新动态 |
| `Store.UpdateFeedCounters` | 更新动态计数 |
| `Store.FeedLikeExists` | 判断是否已点赞动态 |
| `Store.AddFeedLike` | 新增动态点赞 |
| `Store.RemoveFeedLike` | 移除动态点赞 |
| `Store.FeedExtra` | 处理动态附加 |
| `Store.SaveFeedExtra` | 保存动态附加 |
| `Store.FeedIDForResource` | 读取资源对应的动态 ID |
| `Store.ResourceIDForFeed` | 读取动态对应的资源 ID |
| `feedKey` | 生成动态键 |
| `Store.AreFriends` | 判断是否为好友 |
| `placeholders` | 生成 SQL 占位符 |

### `internal/repository/friends.go`

功能：好友数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.Friends` | 读取好友列表 |

### `internal/repository/geonotes.go`

功能：玩家留言数据访问

| 函数 | 作用 |
| --- | --- |
| `scanGeonote` | 扫描留言 |
| `Store.CreateGeonote` | 创建留言 |
| `Store.Geonotes` | 处理留言 |
| `Store.GeonoteByID` | 按 ID 获取留言 |
| `Store.DeleteGeonote` | 删除留言 |
| `Store.EditGeonote` | 编辑留言 |
| `Store.MutateGeonote` | 修改留言 |

### `internal/repository/invites.go`

功能：邀请数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.InviteByToken` | 按令牌获取邀请 |
| `Store.DeleteInvite` | 删除邀请 |
| `Store.CreateInvite` | 创建邀请 |
| `Store.AcceptInvite` | 接受邀请 |
| `Store.FriendCount` | 统计好友数量 |
| `Store.Invites` | 获取邀请列表 |

### `internal/repository/level_pickups.go`

功能：关卡拾取数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.LevelPickupEmitters` | 处理关卡拾取发射器 |

### `internal/repository/magic_status.go`

功能：魔法与状态效果数据访问

无顶层函数。

### `internal/repository/messages.go`

功能：消息数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.UserIDs` | 读取用户 ID 列表 |
| `scanGift` | 扫描礼物 |
| `Store.SentGifts` | 读取已发送礼物 |
| `Store.ReceivedGifts` | 读取已接收礼物 |
| `Store.PendingGiftCount` | 统计待领取礼物数量 |
| `Tx.EnsureCurrency` | 确保货币 |
| `Tx.CurrencyBalance` | 处理货币余额 |
| `Tx.ClampSpend` | 限制扣减 |
| `Tx.AddCurrency` | 新增货币 |
| `Tx.SetCurrency` | 设置货币 |
| `Tx.InsertGift` | 插入礼物 |
| `Tx.GiftByID` | 按 ID 获取礼物 |
| `Tx.LatestUnreceivedGiftFrom` | 获取来自指定好友的最新未接收礼物 |
| `Tx.UnreceivedGiftsFrom` | 获取来自指定好友的未接收礼物 |
| `Tx.MarkGiftReceived` | 标记礼物已接收 |
| `Store.MarkMessageReceived` | 标记消息已接收 |
| `Tx.RelationTx` | 读取关系事务数据 |
| `Tx.EnsureRelationPairTx` | 确保关系配对事务 |
| `Tx.SaveRelationTx` | 保存关系事务 |
| `Store.CountAddedFriends` | 统计已添加好友 |

### `internal/repository/player_state.go`

功能：玩家状态数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.HasUser` | 判断用户是否存在 |
| `Store.ForgeRates` | 读取锻造概率 |
| `scanOutfit` | 扫描装扮 |
| `Store.GetOrCreateOutfit` | 获取或创建装扮 |
| `Store.Outfits` | 批量读取装扮 |
| `outfitPrefixScanner.Scan` | 扫描数据行 |
| `Store.SetOutfitParts` | 设置装扮部件 |
| `Store.AchievementStats` | 读取成就统计 |

### `internal/repository/queries.go`

功能：通用 SQL 查询封装

| 函数 | 作用 |
| --- | --- |
| `New` | 创建实例 |
| `Store.ExecContext` | 执行上下文 |
| `Store.QueryContext` | 查询上下文 |
| `Store.QueryRowContext` | 查询行上下文 |
| `Store.BeginTx` | 开始事务 |
| `Tx.ExecContext` | 执行上下文 |
| `Tx.QueryRowContext` | 查询行上下文 |
| `Tx.QueryContext` | 查询上下文 |
| `Tx.Commit` | 提交 |
| `Tx.Rollback` | 回滚 |
| `UserLookup` | 构造用户查找的 SQL 语句 |
| `CurrencyUpdate` | 构造更新货币的 SQL 语句 |

### `internal/repository/relationships.go`

功能：关系数据访问

| 函数 | 作用 |
| --- | --- |
| `nowEpoch` | 返回当前时间戳 |
| `scanRelation` | 扫描关系 |
| `Store.Relation` | 处理关系 |
| `Store.BlockedFriends` | 处理拉黑好友 |
| `ensureOneRelationship` | 确保单个关系 |
| `addHints` | 新增提示 |
| `Store.SaveFriendName` | 保存好友名称 |
| `Store.SaveFriendBlock` | 保存好友拉黑 |
| `Store.RemoveFriendHint` | 移除好友提示 |
| `Store.ExternalFriends` | 读取外部好友 |
| `Store.SerendipityMatches` | 读取缘分匹配 |
| `Store.StarTags` | 获取星标标签 |
| `Store.LatestAssignment` | 读取最新服务器分配 |
| `Store.AssignServer` | 分配服务器 |
| `Store.SaveJoinAssignment` | 保存加入分配 |
| `Store.SaveConstellationPages` | 保存星座页面 |
| `Store.FollowerCount` | 统计关注者数量 |
| `Store.UpsertBoatFollow` | 写入船关注 |
| `Store.DeleteBoatFollow` | 删除船关注 |
| `CleanCreator` | 清理创建者 ID |

### `internal/repository/resource_views.go`

功能：资源视图查询

无顶层函数。

### `internal/repository/stage.go`

功能：舞台数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.StageState` | 读取舞台状态 |
| `Store.SaveStageState` | 保存舞台状态 |

### `internal/repository/storage.go`

功能：数据库连接、迁移与存储工具

| 函数 | 作用 |
| --- | --- |
| `Open` | 打开 SQLite 数据库连接 |
| `fileURIPath` | 辅助函数：文件URI路径 |
| `Migrate` | 初始化新库并执行迁移 |
| `CheckSchema` | 校验数据库结构 |

### `internal/repository/unlocks.go`

功能：解锁数据访问

| 函数 | 作用 |
| --- | --- |
| `Store.SeedDefaultUnlocks` | 初始化默认解锁 |
| `Store.Unlocks` | 读取解锁列表 |
| `Store.AckUnlock` | 确认解锁 |

### `internal/repository/wing_conversion.go`

功能：光翼兑换的事务内持久化操作。

| 函数 | 作用 |
| --- | --- |
| `Tx.PrepareWingConversion` | 确保货币行存在，并返回决定是否初始化光翼所需的当前行数。 |
| `Tx.SeedWingForConversion` | 按业务层给定的名称和存入 ID 初始化光翼。 |
| `Tx.MarkWingsConverted` | 返回本次实际发生状态变化的行数，用于决定声望奖励。 |
| `Tx.CreditWingPrestige` | 在当前事务内增加兑换所得声望。 |
| `Store.WingBuffs` | 按原有记录顺序读取玩家的光翼。 |

## internal/service

### `internal/service/service.go`

功能：服务门面：按业务组合子服务

| 函数 | 作用 |
| --- | --- |
| `New` | 创建实例 |

## internal/service/account

### `internal/service/account/account.go`

功能：账号创建、登录与会话鉴权

| 函数 | 作用 |
| --- | --- |
| `Service.createUser` | 创建用户记录 |
| `Service.Create` | 创建账号或资源 |
| `Service.Login` | 执行账号登录 |
| `augmentLegacy` | 为旧版响应补充字段 |

### `internal/service/account/account_data.go`

功能：货币、存档、排行与世界探索数据

| 函数 | 作用 |
| --- | --- |
| `Service.Currency` | 获取账号货币 |
| `Service.dailyBonus` | 发放每日奖励 |
| `Service.Userdata` | 读取或保存用户数据 |
| `Service.Rank` | 获取排行信息 |
| `Service.Discover` | 查找可加入玩家 |
| `Service.Checkpoints` | 读取或设置检查点 |
| `levelKey` | 规范化关卡键 |

### `internal/service/account/auth_tokens.go`

功能：资源访问令牌与 Starwatch 鉴权

| 函数 | 作用 |
| --- | --- |
| `Service.ResourceAuthToken` | 签发资源访问令牌 |
| `randomHex` | 生成随机十六进制字符串 |
| `signedJWT` | 生成签名的 JWT |
| `Service.StarwatchAuth` | 执行星巡鉴权 |

### `internal/service/account/bootstrap.go`

功能：客户端启动资源（版本、变量、MOTD 等）

| 函数 | 作用 |
| --- | --- |
| `Service.LatestBuild` | 获取最新构建版本 |
| `Service.Capabilities` | 获取设备能力配置 |
| `Service.Variables` | 获取客户端变量 |
| `Service.Motd` | 获取公告信息 |
| `Service.Support` | 查询客服回复 |

### `internal/service/account/compat.go`

功能：本业务包使用的兼容类型和转换。

| 函数 | 作用 |
| --- | --- |
| `text` | （未提供函数说明） |
| `number` | （未提供函数说明） |
| `integer` | （未提供函数说明） |
| `stringValue` | （未提供函数说明） |
| `require` | （未提供函数说明） |
| `fail` | （未提供函数说明） |
| `now` | （未提供函数说明） |
| `remoteIP` | （未提供函数说明） |
| `token` | （未提供函数说明） |
| `uuid` | （未提供函数说明） |
| `currencyField` | （未提供函数说明） |
| `roomAdmissionToken` | （未提供函数说明） |
| `validConfiguredArray` | （未提供函数说明） |

### `internal/service/account/profile.go`

功能：成就、地图、徽章与收集品档案

| 函数 | 作用 |
| --- | --- |
| `Service.AchievementDefinitions` | 获取成就定义 |
| `nullableString` | 可空字符串转接口值 |
| `nullableInt` | 可空整数转接口值 |
| `nullableBool` | 可空布尔转接口值 |
| `Service.MapDefinitions` | 获取地图定义 |
| `Service.Achievements` | 获取成就列表 |
| `Service.Badge` | 获取徽章数量 |
| `Service.Collectibles` | 获取收集品列表 |
| `Service.optionalFullCollectibles` | 辅助函数：可选完整收集品 |

### `internal/service/account/room_admission.go`

功能：房间准入令牌签发与校验

| 函数 | 作用 |
| --- | --- |
| `Service.VerifyRoomAdmission` | 校验房间准入 |

### `internal/service/account/service.go`

功能：account 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |

## internal/service/commerce

### `internal/service/commerce/buy_candle_wax.go`

功能：购买蜡烛蜡/锻造资源

| 函数 | 作用 |
| --- | --- |
| `forgeSource` | 解析锻造资源来源 |
| `Service.BuyCandleWax` | 购买蜡烛蜡 |

### `internal/service/commerce/code_verify.go`

功能：邀请码校验

| 函数 | 作用 |
| --- | --- |
| `inviteError` | 构造邀请错误 |
| `Service.VerifyCode` | 校验邀请码 |

### `internal/service/commerce/commerce_config.go`

功能：商城配置、产品信息与占位接口

| 函数 | 作用 |
| --- | --- |
| `Service.RedeemReward` | 确认奖励（占位） |
| `Service.AckForgeRateReset` | 确认锻造概率重置（占位） |
| `Service.CommerceUpdate` | 更新商城信息（占位） |
| `Service.CommerceProductInfo` | 获取商品信息 |

### `internal/service/commerce/commerce_shop.go`

功能：商店与战利品箱读取

| 函数 | 作用 |
| --- | --- |
| `Service.seedShopDefaults` | 初始化商店默认 |
| `Service.GetShop` | 获取商店 |
| `Service.openLootboxes` | 打开战利品箱 |
| `Service.GetLootboxes` | 获取战利品箱 |

### `internal/service/commerce/commerce_transactions.go`

功能：收据与履约交易处理

| 函数 | 作用 |
| --- | --- |
| `commerceFailure` | 构造交易失败错误 |
| `commerceCurrencyField` | 规范化交易货币字段 |
| `Service.CommerceReceipt` | 处理交易收据 |
| `Service.CommerceFulfill` | 处理交易履约 |

### `internal/service/commerce/commerce_wallet.go`

功能：钱包/货币快照读取

无顶层函数。

### `internal/service/commerce/compat.go`

功能：本业务包使用的兼容类型和转换。

| 函数 | 作用 |
| --- | --- |
| `text` | （未提供函数说明） |
| `number` | （未提供函数说明） |
| `stringValue` | （未提供函数说明） |
| `fail` | （未提供函数说明） |
| `now` | （未提供函数说明） |
| `token` | （未提供函数说明） |
| `defInt` | （未提供函数说明） |
| `legacyResponse` | （未提供函数说明） |
| `clientVersion` | （未提供函数说明） |
| `currencyField` | （未提供函数说明） |
| `currencySnapshot` | （未提供函数说明） |
| `defSeconds` | （未提供函数说明） |
| `asObject` | （未提供函数说明） |
| `boolSetting` | （未提供函数说明） |
| `validConfiguredArray` | （未提供函数说明） |
| `firstText` | （未提供函数说明） |
| `Service.genericShopDef` | （未提供函数说明） |
| `Service.spiritNode` | （未提供函数说明） |

### `internal/service/commerce/event_buff.go`

功能：活动增益购买

| 函数 | 作用 |
| --- | --- |
| `Service.PurchaseEventBuff` | 购买活动增益 |

### `internal/service/commerce/purchase_dye.go`

功能：染料购买

| 函数 | 作用 |
| --- | --- |
| `dyeSlot` | 解析染料槽位 |
| `Service.PurchaseDye` | 购买染料 |

### `internal/service/commerce/purchase_generic.go`

功能：通用商店物品购买

| 函数 | 作用 |
| --- | --- |
| `genericFailure` | 构造通用购买失败响应 |
| `Service.PurchaseGenericShopItem` | 购买通用商店物品 |

### `internal/service/commerce/purchase_spirit.go`

功能：精灵商店物品购买

| 函数 | 作用 |
| --- | --- |
| `spiritUnlockID` | 解析精灵解锁 ID |
| `spiritFailure` | 构造精灵购买失败响应 |
| `Service.PurchaseSpiritShopItem` | 购买精灵商店物品 |

### `internal/service/commerce/purchase_unlock.go`

功能：解锁项购买

| 函数 | 作用 |
| --- | --- |
| `Service.PurchaseUnlock` | 购买解锁 |

### `internal/service/commerce/redeem_box.go`

功能：战利品箱兑换

| 函数 | 作用 |
| --- | --- |
| `Service.boxMagicID` | 解析箱子魔法 ID |
| `redeemBoxError` | 构造兑换失败响应 |
| `Service.RedeemBox` | 兑换战利品箱 |

### `internal/service/commerce/service.go`

功能：commerce 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |

### `internal/service/commerce/spirit_shops.go`

功能：精灵商店配置与节点处理

| 函数 | 作用 |
| --- | --- |
| `spiritNodeID` | 读取精灵节点 ID |
| `validSpiritNodes` | 校验精灵节点 |
| `spiritTreeLimit` | 读取精灵树上限 |
| `limitSpiritTree` | 限制精灵树节点数 |
| `neutralizeSpiritNodes` | 中和精灵节点 |
| `Service.SpiritShops` | 获取精灵商店 |

## internal/service/content

### `internal/service/content/commerce_catalog.go`

功能：商城静态目录数据

无顶层函数。

### `internal/service/content/compat.go`

功能：本业务包使用的兼容类型和转换。

| 函数 | 作用 |
| --- | --- |
| `text` | （未提供函数说明） |
| `number` | （未提供函数说明） |
| `integer` | （未提供函数说明） |
| `stringValue` | （未提供函数说明） |
| `fail` | （未提供函数说明） |
| `now` | （未提供函数说明） |
| `token` | （未提供函数说明） |
| `uuid` | （未提供函数说明） |
| `legacyResponse` | （未提供函数说明） |
| `clientVersion` | （未提供函数说明） |
| `asObject` | （未提供函数说明） |
| `boolSetting` | （未提供函数说明） |
| `validConfiguredArray` | （未提供函数说明） |

### `internal/service/content/config_resources.go`

功能：可配置资源读取与邀请管理

| 函数 | 作用 |
| --- | --- |
| `Service.QuestDefinitions` | 获取任务定义 |
| `Service.IAPList` | 获取内购列表 |
| `Service.QuestionnaireGroups` | 获取问卷分组 |
| `Service.GenericShops` | 获取通用商店 |
| `Service.Invites` | 获取邀请列表 |
| `Service.CheckInvite` | 检查邀请 |
| `Service.DeleteInvite` | 删除邀请 |
| `formatInviteCode` | 格式化邀请码 |
| `Service.CreateInvite` | 创建邀请 |
| `Service.AcceptInvite` | 接受邀请 |

### `internal/service/content/definitions_misc.go`

功能：问卷、视频、信任与违规定义

| 函数 | 作用 |
| --- | --- |
| `isoDate` | 把日期转为 ISO 格式 |
| `Service.Surveys` | 获取问卷列表 |
| `Service.VideoDefinitions` | 获取视频定义 |
| `Service.TrustActivityDefinitions` | 获取信任活动定义 |
| `Service.TrustGroupDefinitions` | 获取信任分组定义 |
| `Service.InfractionStatus` | 获取违规状态 |

### `internal/service/content/definitions_resources.go`

功能：货币、信任、辐光等定义资源

| 函数 | 作用 |
| --- | --- |
| `Service.CurrencyDefinitions` | 获取货币定义 |
| `Service.TrustTierDefinitions` | 获取信任等级定义 |
| `Service.RadianceSpawnDefinitions` | 获取辐光刷新定义 |
| `Service.batch1User` | 解析旧版批量接口用户会话 |
| `Service.RemoteConfigUpload` | 上传远程配置 |
| `Service.GenericShopTracking` | 记录通用商店购买跟踪 |
| `Service.LevelPickups` | 获取关卡拾取 |

### `internal/service/content/event_schedule.go`

功能：活动日程生成与版本适配

| 函数 | 作用 |
| --- | --- |
| `asObjects` | 转换为对象列表 |
| `tcandleGroup` | 判断蜡烛活动分组 |
| `validScheduleEvents` | 校验活动日程事件 |
| `Service.scheduleSource` | 读取活动日程配置源 |
| `scheduledNames` | 提取已排期名称 |
| `selectedDaily` | 选取每日排期 |
| `applyCandleSchedule` | 应用蜡烛活动日程 |
| `applySeasonSchedule` | 应用赛季活动日程 |
| `Service.EventSchedule` | 获取活动日程 |

### `internal/service/content/service.go`

功能：content 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |

## internal/service/core

### `internal/service/core/auth.go`

功能：会话与用户校验

| 函数 | 作用 |
| --- | --- |
| `Base.FindUser` | 按条件查找用户 |
| `BanError` | 构造封禁错误 |
| `Base.CheckAllowed` | 校验账号是否允许登录 |
| `Base.Authenticated` | 校验会话并返回用户 |
| `Base.QuestUser` | 校验任务请求用户 |

### `internal/service/core/base.go`

功能：服务共享依赖

| 函数 | 作用 |
| --- | --- |
| `NewBase` | 创建共享依赖 |

### `internal/service/core/catalog.go`

功能：静态资源与配置读取

| 函数 | 作用 |
| --- | --- |
| `Base.Catalog` | 读取静态资源目录中的列表 |
| `Base.ConfiguredObject` | 读取配置对象 |
| `Base.StatusConfig` | 读取状态配置列表 |
| `emoteFamily` | 解析表情动作系列 |
| `Base.SpiritNode` | 查找精灵节点 |
| `Base.GenericShopDef` | 查找通用商店定义 |
| `Base.EmoteFamilyLevels` | 获取表情动作升级系列 |

### `internal/service/core/currency.go`

功能：货币字段与快照

| 函数 | 作用 |
| --- | --- |
| `CurrencyField` | 判断是否为货币字段 |
| `Base.CurrencySnapshot` | 读取货币快照 |
| `CurrencySnapshotReader` | 读取货币快照（可指定读取器） |

### `internal/service/core/legacy.go`

功能：旧版静态响应读取

| 函数 | 作用 |
| --- | --- |
| `LegacyResponse` | 从旧版静态数据读取响应 |

### `internal/service/core/legacy_contracts.go`

功能：旧版接口路径与登录资源清单（生成文件）

无顶层函数。

### `internal/service/core/magic.go`

功能：魔法/消耗品/增益共享逻辑

| 函数 | 作用 |
| --- | --- |
| `magicVersionAllowed` | 判断魔法定义是否版本可见 |
| `Base.BuffIDAllowed` | 判断增益是否版本可见 |
| `IsUGCMagic` | 判断是否为 UGC 魔法 |
| `Base.MagicDefinitions` | 读取魔法定义 |
| `Base.VersionMagicDefinitions` | 读取按版本过滤的魔法定义 |
| `Base.MagicUser` | 校验魔法请求用户 |
| `Base.Consumables` | 读取消耗品列表 |
| `Base.Buffs` | 读取增益列表 |
| `Base.ForgeRates` | 读取锻造概率 |

### `internal/service/core/policy.go`

功能：授权/开关策略读取

| 函数 | 作用 |
| --- | --- |
| `Base.LegacyObject` | 读取旧版配置文件对象 |
| `Base.Policy` | 读取开关策略 |

### `internal/service/core/relationship.go`

功能：关系定义版本过滤

| 函数 | 作用 |
| --- | --- |
| `RelationshipEventAllowed` | 判断关系事件是否允许 |
| `LegacyRelationshipDefs` | 按版本过滤关系定义 |

### `internal/service/core/room.go`

功能：房间准入令牌

| 函数 | 作用 |
| --- | --- |
| `Base.RoomAdmissionURI` | 返回房间准入服务地址 |
| `Base.IssueRoomAdmission` | 签发房间准入令牌 |
| `RoomAdmissionToken` | 计算房间准入令牌 |

### `internal/service/core/types.go`

功能：共享类型与字段工具

| 函数 | 作用 |
| --- | --- |
| `Error.Error` | 返回错误文本 |
| `Fail` | 构造服务错误响应 |
| `Text` | 读取字符串字段 |
| `Number` | 读取整数字段 |
| `Integer` | 读取整数指针 |
| `StringValue` | 读取字符串指针 |
| `RemoteIP` | 返回请求来源 IP |
| `Now` | 返回当前时间戳 |
| `Require` | 校验必需字段 |
| `DefInt` | 读取通用整数 |
| `Token` | 生成随机令牌 |
| `UUID` | 生成 UUID |

### `internal/service/core/values.go`

功能：通用取值与格式化工具

| 函数 | 作用 |
| --- | --- |
| `RelationTime` | 格式化关系时间 |
| `BoolValue` | 读取布尔值 |
| `FloatValue` | 读取浮点值 |
| `IntList` | 解析整数列表 |
| `DefSeconds` | 将定义秒值转换为秒 |
| `AsObject` | 转换为对象 |
| `BoolSetting` | 读取布尔设置 |
| `ValidConfiguredArray` | 校验配置数组 |
| `NowMillis` | 返回当前毫秒时间戳 |
| `strconvFormatFloat` | 格式化浮点数 |
| `ScalarString` | 转换标量为字符串 |
| `FirstText` | 读取第一个非空字符串 |

### `internal/service/core/version.go`

功能：客户端版本解析

| 函数 | 作用 |
| --- | --- |
| `LegacyVersion` | 判断是否为旧版客户端 |
| `ClientVersion` | 解析客户端版本号 |
| `Base.LegacyUA` | 判断是否为旧版客户端（含配置覆盖） |

## internal/service/inventory

### `internal/service/inventory/compat.go`

功能：本业务包使用的兼容类型和转换。

| 函数 | 作用 |
| --- | --- |
| `text` | （未提供函数说明） |
| `number` | （未提供函数说明） |
| `integer` | （未提供函数说明） |
| `stringValue` | （未提供函数说明） |
| `require` | （未提供函数说明） |
| `fail` | （未提供函数说明） |
| `now` | （未提供函数说明） |
| `defInt` | （未提供函数说明） |
| `legacyResponse` | （未提供函数说明） |
| `clientVersion` | （未提供函数说明） |
| `defSeconds` | （未提供函数说明） |
| `isUGCMagic` | （未提供函数说明） |
| `legacyRelationshipDefs` | （未提供函数说明） |

### `internal/service/inventory/emote_family.go`

功能：表情家族与等级映射

无顶层函数。

### `internal/service/inventory/magic.go`

功能：消耗品、增益定义与使用逻辑

| 函数 | 作用 |
| --- | --- |
| `Service.ConsumableDefinitions` | 获取消耗品定义 |
| `Service.BuffDefinitions` | 获取增益定义 |
| `Service.boundMagicDefinitions` | 绑定并筛选魔法定义 |
| `Service.GetConsumables` | 获取消耗品（新版） |
| `Service.GetConsumables1` | 获取消耗品（旧版） |
| `Service.GetBuffs` | 获取增益（新版） |
| `Service.GetBuffs1` | 获取增益（旧版） |
| `Service.RemoveBuff` | 移除增益 |
| `Service.ConsumeItem` | 使用消耗品 |

### `internal/service/inventory/player_state.go`

功能：玩家锻造、装扮、问卷与成就统计

| 函数 | 作用 |
| --- | --- |
| `Service.Outfit` | 获取装扮 |
| `Service.Questionnaires` | 获取问卷 |
| `Service.AchievementStats` | 读取成就统计 |
| `Service.AchievementStatsTracking` | 获取成就统计跟踪 |
| `Service.EventCurrencyDefinitions` | 获取活动货币定义 |
| `Service.RelationshipDefinitions` | 获取关系能力定义 |

### `internal/service/inventory/service.go`

功能：inventory 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |
| `Service.ForgeRates` | 读取锻造概率 |

### `internal/service/inventory/status_unlocks.go`

功能：状态解锁与染料移除

| 函数 | 作用 |
| --- | --- |
| `Service.StatusUnlocks` | 获取状态解锁 |
| `Service.AddStatusUnlock` | 新增状态解锁 |
| `Service.DeleteStatusUnlock` | 删除状态解锁 |
| `Service.RemoveDye` | 移除染料 |
| `Service.CollectCollectible` | 收集收集品 |
| `Service.AckCollectible` | 确认收集品 |

### `internal/service/inventory/unlocks.go`

功能：解锁项列表、默认解锁与确认

| 函数 | 作用 |
| --- | --- |
| `Service.defaultUnlockNames` | 读取默认解锁名称 |
| `isDefaultUnlock` | 判断是否默认解锁 |
| `Service.seedDefaultUnlocks` | 初始化默认解锁 |
| `Service.unlockConfigLimit` | 读取解锁配置上限 |
| `unlockPriority` | 计算解锁优先级 |
| `Service.finalizeUnlocks` | 整理解锁列表 |
| `unlockPayload` | 构造解锁响应 |
| `Service.maxCountUnlocks` | 构造上限解锁列表 |
| `Service.Unlocks` | 读取解锁列表 |
| `Service.mergeFullUnlocks` | 合并完整解锁列表 |
| `Service.AckUnlock` | 确认解锁 |

### `internal/service/inventory/wing_buffs.go`

功能：光翼增益收集/存入/转换

| 函数 | 作用 |
| --- | --- |
| `Service.wingUser` | 解析光翼接口用户 |
| `wingNames` | 提取光翼名称 |
| `depositMask` | 解析存入掩码 |
| `stringifyNumber` | 把数字转为字符串 |
| `Service.wingRows` | 读取光翼数据 |
| `Service.fullWingRows` | 读取完整光翼数据 |
| `Service.GetWingBuffs` | 获取光翼增益 |
| `Service.CollectWingBuffs` | 收集光翼增益 |
| `Service.DropWingBuffs` | 丢弃光翼增益 |
| `Service.wingChange` | 变更光翼数据 |
| `Service.DepositWingBuffs` | 存入光翼增益 |
| `Service.ConvertWingBuffs` | 转换光翼增益 |

## internal/service/legacydata

### `internal/service/legacydata/data.go`

功能：旧版静态响应数据（内嵌 JSON）

| 函数 | 作用 |
| --- | --- |
| `Read` | 读取原始数据文件 |
| `Response` | 读取并解析旧版响应 |

## internal/service/messaging

### `internal/service/messaging/compat.go`

功能：本业务包使用的兼容类型和转换。

| 函数 | 作用 |
| --- | --- |
| `text` | （未提供函数说明） |
| `number` | （未提供函数说明） |
| `stringValue` | （未提供函数说明） |
| `fail` | （未提供函数说明） |
| `now` | （未提供函数说明） |
| `uuid` | （未提供函数说明） |
| `legacyResponse` | （未提供函数说明） |
| `clientVersion` | （未提供函数说明） |
| `currencyField` | （未提供函数说明） |
| `floatValue` | （未提供函数说明） |
| `intList` | （未提供函数说明） |
| `relationTime` | （未提供函数说明） |
| `nowMillis` | （未提供函数说明） |
| `scalarString` | （未提供函数说明） |

### `internal/service/messaging/mailbox.go`

功能：进程内消息总线与邮件/礼物公共逻辑

| 函数 | 作用 |
| --- | --- |
| `newMessageBus` | 创建消息总线 |
| `messageBus.push` | 推送消息 |
| `messageBus.drain` | 取出并清空消息 |
| `messageBus.empty` | 判断消息队列是否为空 |
| `Service.broadcastChat` | 广播聊天消息 |
| `Service.pushChat` | 推送聊天消息 |
| `normalizeGiftType` | 规范化礼物类型 |
| `newGiftID` | 生成礼物 ID |
| `giftIDInt` | 转换礼物 ID |
| `sentGiftEntry` | 构造已发送礼物响应 |
| `recvdGiftEntry` | 构造已接收礼物响应 |
| `recvdMessageEntry` | 构造礼物消息响应 |
| `Service.mailbox` | 读取邮箱配置 |
| `versionBound` | 解析客户端版本 |
| `versionAtLeast` | 判断版本不低于 |
| `versionAtMost` | 判断版本不高于 |
| `mailVisible` | 判断邮件是否可见 |
| `mailPayload` | 构造邮件响应 |
| `sortedUnique` | 排序去重 |
| `intsJSON` | 把整数列表转 JSON |

### `internal/service/messaging/messaging_routes.go`

功能：聊天、消息、蜡烛与免费礼物接口

| 函数 | 作用 |
| --- | --- |
| `Service.messageAuth` | 校验消息接口会话 |
| `chatPayload` | 构造聊天载荷 |
| `Service.GetPendingMessages` | 获取待处理消息 |
| `Service.ChatSend` | 发送聊天 |
| `Service.AccountSend` | 发送账号消息或礼物 |
| `Service.accountGiftSend` | 发送账号礼物 |
| `Service.Translate` | 翻译占位接口 |
| `Service.SetDeviceToken` | 设置设备令牌（占位） |
| `Service.relationAbilities` | 读取关系能力定义 |
| `abilityContains` | 判断能力是否包含 |
| `unionAbilities` | 合并能力列表 |
| `removeHint` | 移除关系提示 |
| `giveFailure` | 构造赠送失败响应 |
| `friendUpdateEntry` | 构造好友更新响应 |
| `Service.GiveCandle` | 赠送蜡烛 |
| `Service.giftIDs` | 提取礼物 ID 列表 |
| `Service.ClaimMessageGift` | 领取消息礼物 |
| `Service.MessageGetAllV2` | 获取全部消息（v2 新版） |
| `Service.MessageGetAllV1` | 获取全部消息（v1 旧版） |
| `Service.mailboxList` | 构造邮箱列表 |
| `mailboxVersion` | 解析邮箱版本 |
| `Service.MessageGet` | 获取单条消息 |
| `Service.MarkMessagesReceived` | 标记消息已接收 |
| `Service.MarkMessagesSeen` | 标记消息已读 |
| `Service.FreeGiftsGetPending` | 获取待领取免费礼物 |
| `Service.FreeGiftsSend` | 发送免费礼物 |
| `Service.FreeGiftsClaim` | 领取免费礼物 |

### `internal/service/messaging/service.go`

功能：messaging 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |

## internal/service/quests

### `internal/service/quests/compat.go`

功能：本业务包使用的兼容类型和转换。

| 函数 | 作用 |
| --- | --- |
| `text` | （未提供函数说明） |
| `number` | （未提供函数说明） |
| `fail` | （未提供函数说明） |
| `now` | （未提供函数说明） |
| `uuid` | （未提供函数说明） |
| `defInt` | （未提供函数说明） |
| `currencySnapshot` | （未提供函数说明） |

### `internal/service/quests/player_quests.go`

功能：玩家任务创建、更新与进度

| 函数 | 作用 |
| --- | --- |
| `emptyQuestBuckets` | 构造空任务分桶 |
| `Service.questRow` | 读取单条任务 |
| `Service.PlayerQuests` | 获取玩家任务 |
| `Service.PlayerQuestsByName` | 按名称获取玩家任务 |
| `Service.CheckPreviousCompletion` | 检查任务先前完成记录 |
| `Service.CreateQuest` | 创建任务 |
| `Service.UpdateQuest` | 更新任务 |
| `Service.SetAchievementStats` | 设置成就统计 |
| `numericStat` | 转换数值统计 |
| `Service.QuestSchedule` | 获取任务日程 |
| `previewQuestIDs` | 预览任务 ID 列表 |

### `internal/service/quests/quests_common.go`

功能：任务公共工具与空占位响应

| 函数 | 作用 |
| --- | --- |
| `Service.EmptyDailyQuests` | 返回空的每日任务 |
| `Service.EmptyWeeklyQuests` | 返回空的每周任务 |
| `Service.EmptySeasonQuestsV2` | 返回空的赛季任务（v2 新版占位） |
| `Service.EmptySeasonPassDefs` | 返回空的赛季通行证定义 |
| `Service.EmptyEventCurrency` | 返回空的活动货币 |
| `Service.gameControl` | 读取游戏控制配置 |
| `nestedObject` | 读取嵌套对象 |
| `questPool` | 读取任务池 |
| `Service.dailyQuestPools` | 读取每日任务池 |
| `Service.questReward` | 读取任务奖励 |
| `Service.seasonNumber` | 读取当前赛季编号 |
| `Service.seasonDisabled` | 判断赛季是否禁用 |
| `pacificDay` | 按太平洋时区计算日期 |
| `localDay` | 返回当前太平洋日期 |
| `questNames` | 提取任务名称列表 |
| `questID` | 读取任务 ID |
| `insertionCount` | 判断是否插入成功 |
| `questStats` | 读取任务统计 |
| `questDefinition` | 构造任务定义 |

### `internal/service/quests/season_quests.go`

功能：赛季任务与世界任务奖励

| 函数 | 作用 |
| --- | --- |
| `Service.SeasonQuests` | 获取赛季任务 |
| `Service.ActivateSeasonQuest` | 激活赛季任务 |
| `Service.ClaimSeasonQuestReward` | 领取赛季任务奖励 |
| `Service.questEligible` | 判断任务是否可领取 |
| `Service.SeasonStart` | 开始赛季 |
| `Service.SeasonFinalize` | 结算赛季 |
| `Service.WorldQuests` | 获取世界任务 |
| `Service.ClaimWorldQuestReward` | 领取世界任务奖励 |
| `Service.CollectPickupBatch` | 批量收集拾取 |

### `internal/service/quests/service.go`

功能：quests 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |

## internal/service/relationships

### `internal/service/relationships/admin_friends.go`

功能：管理后台好友增删与审计

| 函数 | 作用 |
| --- | --- |
| `Service.adminAllowed` | 判断管理员请求是否允许 |
| `adminForbidden` | 返回管理员拒绝错误 |
| `Service.AdminFriendsList` | 列出管理员好友 |
| `Service.AdminFriendsSet` | 设置管理员好友关系 |
| `Service.AdminFriendsClear` | 清空管理员好友关系 |

### `internal/service/relationships/compat.go`

功能：本业务包使用的兼容类型和转换。

| 函数 | 作用 |
| --- | --- |
| `text` | （未提供函数说明） |
| `number` | （未提供函数说明） |
| `stringValue` | （未提供函数说明） |
| `fail` | （未提供函数说明） |
| `now` | （未提供函数说明） |
| `token` | （未提供函数说明） |
| `legacyResponse` | （未提供函数说明） |
| `clientVersion` | （未提供函数说明） |
| `boolValue` | （未提供函数说明） |
| `floatValue` | （未提供函数说明） |
| `intList` | （未提供函数说明） |
| `relationTime` | （未提供函数说明） |

### `internal/service/relationships/friend_status.go`

功能：好友在线状态与星座状态

| 函数 | 作用 |
| --- | --- |
| `Service.OnlineFriends` | 获取在线好友 |
| `Service.FriendStatues` | 获取好友状态 |

### `internal/service/relationships/friends.go`

功能：好友列表读取

| 函数 | 作用 |
| --- | --- |
| `Service.Friends` | 读取好友列表 |

### `internal/service/relationships/relationship_definitions.go`

功能：关系能力定义与客户端版本过滤

无顶层函数。

### `internal/service/relationships/relationships.go`

功能：好友关系、关注、拉黑与星座页面

| 函数 | 作用 |
| --- | --- |
| `compactRelation` | 构造精简关系响应 |
| `outfitPayload` | 构造装扮响应 |
| `candidateFriendPayload` | 构造候选好友响应 |
| `Service.relationshipUser` | 解析关系接口用户 |
| `valueOr` | 读取值或默认值 |
| `Service.GetBlockedFriends` | 获取拉黑好友 |
| `emptyBlockedFriends` | 构造空拉黑列表 |
| `missingFields` | 检查缺失字段 |
| `Service.SetFriendName` | 设置好友名称 |
| `Service.SetFriendBlock` | 设置好友拉黑 |
| `Service.GetExternalFriends` | 获取外部好友 |
| `Service.Serendipity` | 获取缘分匹配 |
| `Service.StarTags` | 获取星标标签 |
| `joinError` | 构造加入好友错误 |
| `Service.JoinFriendGame` | 加入好友游戏 |
| `Service.SetOutfit` | 设置装扮 |
| `Service.SetJoinable` | 设置可加入状态 |
| `defaultRemoteOutfit` | 生成默认远程装扮 |
| `Service.GetRemoteOutfit` | 获取远程装扮 |
| `Service.RemoveRelationshipHint` | 移除关系提示 |
| `Service.GetAllFriendGist` | 获取全部好友摘要 |
| `Service.GetFollowerCount` | 获取关注者数量 |
| `constellationFailure` | 构造星座操作失败响应 |
| `Service.UpdateFriendConstellationPages` | 更新好友星座页面 |
| `Service.TrustGet` | 获取信任信息 |
| `Service.TrustGetCapped` | 获取信任上限信息 |
| `followResponse` | 构造关注响应 |
| `Service.AddFollow` | 新增关注 |
| `Service.SetFollowName` | 设置关注名称 |
| `Service.DeleteFollow` | 删除关注 |

### `internal/service/relationships/service.go`

功能：relationships 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |

## internal/service/social

### `internal/service/social/compat.go`

功能：本业务包使用的兼容类型和转换。

| 函数 | 作用 |
| --- | --- |
| `text` | （未提供函数说明） |
| `number` | （未提供函数说明） |
| `integer` | （未提供函数说明） |
| `stringValue` | （未提供函数说明） |
| `fail` | （未提供函数说明） |
| `now` | （未提供函数说明） |
| `uuid` | （未提供函数说明） |
| `boolValue` | （未提供函数说明） |
| `floatValue` | （未提供函数说明） |
| `nowMillis` | （未提供函数说明） |
| `scalarString` | （未提供函数说明） |
| `firstText` | （未提供函数说明） |

### `internal/service/social/geonotes.go`

功能：玩家留言（geonote）增删改查

| 函数 | 作用 |
| --- | --- |
| `newGeonoteID` | 生成留言 ID |
| `geonoteError` | 构造留言错误 |
| `Service.geonoteReadUser` | 解析留言读取用户 |
| `Service.geonoteWriteUser` | 解析留言写入用户 |
| `vec3` | 解析三维向量 |
| `geonoteFlag` | 解析留言布尔标记 |
| `geonoteClient` | 构造留言响应 |
| `geonotePayload` | 构造留言列表响应 |
| `geonoteLevel` | 解析留言关卡 |
| `geonoteLimit` | 限制留言数量 |
| `Service.GetUserGeonotes` | 获取用户留言 |
| `Service.RequestGeonotesForNFriends` | 请求若干好友留言 |
| `Service.RequestNBugReportGeonotes` | 请求若干问题举报留言 |
| `runeLimit` | 按字符数截断文本 |
| `Service.CreateGeonote` | 创建留言 |
| `Service.CreateBugReportGeonote` | 创建问题举报留言 |
| `Service.createGeonote` | 创建留言 |
| `floatOr` | 读取浮点数或默认值 |
| `Service.DeleteGeonote` | 删除留言 |
| `Service.DeleteBugReportGeonote` | 删除问题举报留言 |
| `Service.deleteGeonote` | 删除留言 |
| `Service.EditMessageGeonote` | 编辑留言 |
| `Service.SocialGeonote` | 执行社交留言操作 |

### `internal/service/social/recordings.go`

功能：留影录制上传与读取

| 函数 | 作用 |
| --- | --- |
| `Service.recordingDir` | 返回留影目录 |
| `safeRecordingID` | 规范化留影 ID |
| `newRecordingID` | 生成留影 ID |
| `Service.recordingMetaPath` | 返回留影元数据路径 |
| `Service.recordingDataPath` | 返回留影数据路径 |
| `Service.loadRecordingMeta` | 读取留影元数据 |
| `Service.saveRecordingMeta` | 保存留影元数据 |
| `recordingAuthor` | 解析留影作者 |
| `recordingName` | 解析留影名称 |
| `recordingData` | 解析留影数据 |
| `pickRecordingID` | 选取留影 ID |
| `Service.recordingLinkBase` | 返回留影链接前缀 |
| `firstHeaderValue` | 读取首个请求头值 |
| `Service.recordingLink` | 构造留影链接 |
| `Service.recordingOKPayload` | 构造留影成功响应 |
| `Service.RecordingCreateV1` | 创建留影（v1 旧版） |
| `Service.findRecordingForID` | 按 ID 查找留影 |
| `Service.RecordingGetV1` | 获取留影（v1 旧版） |
| `Service.RecordingCreateV2` | 创建留影（v2 新版） |
| `Service.RecordingUpdateV2` | 更新留影（v2 新版） |
| `Service.RecordingGetV2` | 获取留影（v2 新版） |
| `parseRange` | 解析 Range 请求头 |
| `Service.ServeUGC` | 提供UGC |

### `internal/service/social/service.go`

功能：social 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |

### `internal/service/social/social_feed_common.go`

功能：社交动态公共响应构造

| 函数 | 作用 |
| --- | --- |
| `newRecentHints` | 创建最近提示缓存 |
| `recentHints.remember` | 记录最近提示 |
| `recentHints.recall` | 读取最近提示 |
| `feedBareID` | 去除动态 ID 前缀 |
| `feedPublicID` | 生成动态公开 ID |
| `feedISO` | 格式化动态时间 |
| `Service.feedExtra` | 读取动态附加数据 |
| `Service.feedTags` | 读取动态标签 |
| `feedReactions8` | 构造旧版动态回应 |
| `feedReactions9` | 构造新版动态回应 |
| `Service.feedItemV1` | 构造旧版动态条目 |
| `Service.feedItemV2` | 构造新版动态条目 |
| `Service.feedItemAccount` | 构造账号动态条目 |
| `asStringList` | 转换为字符串列表 |
| `feedPoolTypes` | 提取动态池类型 |
| `feedPoolNames` | 提取动态池名称 |
| `clampBatch` | 限制批量大小 |
| `Service.feedAuth` | 校验动态接口会话 |
| `Service.feedByAnyID` | 按任意 ID 查找动态 |

### `internal/service/social/social_feed_routes.go`

功能：社交动态增删改查与互动接口

| 函数 | 作用 |
| --- | --- |
| `Service.socialCreate` | 创建社交动态 |
| `Service.storeFeedExtra` | 保存动态附加数据 |
| `Service.SocialFeedCreate` | 创建社交动态（账号接口） |
| `Service.ServiceSocialFeedCreate` | 创建社交动态（服务接口） |
| `Service.AccountGetUserFeeds` | 获取用户动态（账号接口） |
| `Service.AccountGetCuratedFeeds` | 获取精选动态（账号接口） |
| `emptyCurated` | 构造空精选列表 |
| `Service.AccountGetFeedReactions` | 获取动态回应（账号接口） |
| `Service.AccountFeedLike` | 点赞动态（账号接口） |
| `Service.AccountDeleteFeed` | 删除动态（账号接口） |
| `Service.ServiceFeedEdit` | 编辑动态 |
| `Service.ServiceFeedHide` | 隐藏动态 |
| `Service.ServiceFeedLike` | 点赞动态（服务接口） |
| `Service.ServiceFeedSuperLike` | 超级点赞动态 |
| `Service.ServiceFeedDelete` | 删除动态（服务接口） |
| `Service.SocialFeedComments` | 社交动态评论占位 |
| `Service.SocialFeedReport` | 社交动态举报占位 |
| `Service.ServiceGetCuratedFeeds` | 获取精选动态（服务接口） |
| `emptyServiceCurated` | 构造空服务精选列表 |
| `Service.ServiceGetUserFeeds` | 获取用户动态（服务接口） |
| `Service.ServiceGetByIDs` | 按 ID 批量获取动态 |
| `Service.ServiceGetFeedReactions` | 获取动态回应（服务接口） |
| `Service.ServiceGetPenpals` | 获取笔友列表（占位） |
| `Service.ServiceGetPenpalByID` | 按 ID 获取笔友 |

## internal/service/stage

### `internal/service/stage/service.go`

功能：stage 服务类型

| 函数 | 作用 |
| --- | --- |
| `New` | 创建服务实例 |

### `internal/service/stage/stage.go`

功能：舞台读写与预约占位

| 函数 | 作用 |
| --- | --- |
| `stageIDFor` | 解析舞台 ID 与归属 |
| `normalizeStageProp` | 规范化舞台道具 |
| `normalizeStageProps` | 规范化舞台道具列表 |
| `stagePropsFromRequest` | 从请求解析舞台道具 |
| `Service.StageGet` | 获取舞台 |
| `Service.StageSet` | 保存舞台 |
| `Service.FriendsInApartments` | 获取公寓内好友（占位） |
| `Service.LeaveApartment` | 离开公寓（占位） |
| `Service.UpdateApartmentStatus` | 更新公寓状态（占位） |
| `Service.Reserve` | 预约房间（占位） |

## migrations

### `migrations/embed.go`

功能：SQL 迁移文件嵌入

无顶层函数。
