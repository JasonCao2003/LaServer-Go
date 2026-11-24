# API 启动资源验证（Task 3）

日期：2026-11-24。范围：Go `api-server` 的登录后资源候选入口。资源名、候选 URL、直接依赖与已见 APK 请求证据见 [Task 1 依赖表](../../api-login-resource-dependencies.md)；这里记录本轮实际返回合同、配置故障策略和版本分流。

## 本轮可复现证据

- `TestTask3LegacyConfigCandidateResources` 读取旧 LaServer 的 `config/`，在临时 SQLite 账号上执行 `/account/auth/login`，然后对 **42 条不同的候选资源 URL** 分别发送模拟 `0.1.8`、`0.3.11` 和曾观察到的 `0.13.4.170388` User-Agent，共 126 次拉取。每次均返回 HTTP 200 和 JSON 对象；可确认的响应键也通过断言。旧配置只读，日志在 `logs/api-task3-resource-smoke-20261124.log`（忽略提交）。
- `TestTask3ConfiguredLoginResourcesReportMissingAndCorruptSources` 对地图、任务定义、IAP、收集物、先祖商店分别验证有效、缺失、语法错误和错误结构。`TestTask3EventSchedule*`、`TestTask3MagicStaticOverride*`、`TestTask3ShopUnlocksAndOutfitConfigAndVersion` 及先祖商店回退用例覆盖下面的条件分支。
- 这里的 User-Agent 是测试输入，不代表 APK 已真实拉取这些 URL。Task 1 表中三个无独立入口的名称也没有被计入 42 条。真机/模拟器的冷启动资源接受情况仍属 Task 6。

复现命令（在 `services/api-server`，复用已安装的 Go 工具链）：

```powershell
$env:LASERVER_GAME_CONFIG_DIR = 'C:\Users\Jason\Documents\Project\SkyLA\LaServer\services\game-api\config'
go test ./internal/handler -run '^TestTask3' -count=1
```

## 配置、SQLite 与版本合同

| 资源 / 接口 | 数据来源与成功字段 | 缺失或损坏时 | 版本差异 |
| --- | --- | --- | --- |
| `get_map_defs` `/account/get_map_defs` | `config/map.json` → `get_map_defs[]` | 503；不返回空对象冒充成功 | 两旧版相同 |
| `get_quest_defs` `/account/quest/get_quest_defs` | `config/quest.json` → `get_quest_defs[]`；不在登录资源名清单内 | 503 | 两旧版相同 |
| `iap_list` `/account/iaplist` | `config/iaplist.json` → `iaplist[]`，并可含购买状态字段 | 503 | 资源名 `iap_list`，实际键 `iaplist` |
| `collectibles` `/account/get_collectibles` | `config/collectibles.json` 的列表加 SQLite 用户物品 → `collectibles[]` | 基础配置缺失/坏结构 503；可选全解锁附加文件缺失时只保留基础物品 | 按账号合并，不从资源名推断版本 |
| `spirit_shops` `/account/get_spirit_shops` | 优先 `resources/static_responses/account__get_spirit_shops.json`，再 `config/spirit_shops.json` → `spirit_shops[]`、`spirit_shops_total_count`、`spirit_shops_version` | 首选文件坏时尝试次选；两者无有效数据则 503 | 0.1.8、0.3.11 均使用 120 节点的旧版先祖树上限；本轮验证了两版都能拉取 |
| `event_schedule` `/account/get_event_schedule` | 先读低/高版本日程，再退回 `config/event_schedule.json`；输出 `base_time`、`valid_until`、`events[]` | 缺失、坏 JSON、尾随垃圾时返回兼容的空日程 200 | 0.1.8、0.3.11、0.13.4 使用低版并将时间限制在有符号 32 位内；0.26+ 使用高版并带 `server_time` |
| `motd` `/account/get_motd` | 可选 `config/motd_config.json` → `motd`、标题、按钮及时间字段 | 缺失时生成默认公告；字段缺失或类型不对时补默认公告字段；坏 JSON 503 | 两旧版相同 |
| `get_consumable_defs`、`get_buff_defs` | 静态覆盖文件或内嵌数据 → 同名定义数组 | 覆盖文件缺失、坏结构或尾随垃圾时使用内嵌数据 | 0.1.8、0.3.11 过滤新事件/效果；0.26+ 使用可用定义 |
| `get_shop` `/account/get_shop` | SQLite 商品表，首次读时从内嵌默认商品初始化 → `get_shop`（单元素外层数组） | 不依赖外部配置；SQLite 错误透传为服务错误 | 两旧版相同 |
| `unlocks`、`status_unlocks` | SQLite 已拥有项及可选 `auto_full_unlock.json`、`full_unlocks_template.json`、`outfit_version_map.json`、状态模板 → 对应数组及总数 | 自动全解锁策略坏时关闭；模板缺失保留已拥有项和默认项 | 有效版本映射将 0.1.8/0.3.11 的服饰名分别筛选，且保留通用项 |
| `wing_buffs` | SQLite 及可选全解锁配置 → `wing_buffs[]` | 可选文件缺失时使用已拥有项 | 按账号策略合并；未见单独的版本过滤 |
| `/account/get_outfit` | SQLite 当前装扮 → `set_outfit` 对象 | 不依赖外部配置；无独立的 `outfit_defs` 资源 URL 证据 | 两旧版合同相同 |

`no_season` 标记使所有赛季事件都被过滤时，旧 Python 和 Go 都返回空 JSON 对象 `{}`，属明确保留的旧行为。`generic_shop_defs` 目前仅返回空数组，`questionnaires` 仅返回 `result=ok`；这两项是旧占位语义，不能当成功业务能力。

## 其余登录资源

下列资源逐一包含在 42 条候选入口巡检中。来源为 Task 1 表中的 SQLite、内嵌静态数据或常量；无外部必需配置，正常测试账号返回 200。需要账号的接口仍可因缺参数/无效会话返回 400/401，SQLite 故障仍可能返回 500，本轮没有做数据库故障注入。

| 来源 | 登录资源名（默认响应键与资源名相同） |
| --- | --- |
| SQLite 账号与存档 | `achievement_defs`, `achievement_stats`, `achievements`, `checkpoints`, `currency`, `external_account_friends`, `forge_rates`, `get_buffs`, `get_consumables`, `get_lootboxes`, `get_star_tag_defs`, `online_friends`, `serendipity_matches`, `set_app_badge_number`, `set_friend_statues`, `set_friends`, `set_rank`, `set_world_quests`, `userdata` |
| SQLite 社交与消息 | `free_gifts`, `gift_messages`, `infractions`, `invite_list`, `messages` |
| 内嵌或固定响应 | `achievement_stats_tracking`, `device_capabilities`, `event_currency_defs`, `generic_shop_defs`, `relationship_defs`, `questionnaires` |

以下候选名称与实际返回键不同，巡检按右侧字段断言；名称到 URL 的客户端映射仍未抓取：

| 登录资源名 | 候选接口实际返回键 |
| --- | --- |
| `achievement_stats_tracking` | `achievement_stats`（旧动态处理器也是此键；旧静态 JSON 则使用 `achievement_stats_tracking`） |
| `iap_list` | `iaplist` |
| `free_gifts` | `set_recvd_free_gifts`, `set_sent_free_gifts` |
| `gift_messages` | `set_recvd_messages`, `set_sent_messages` |
| `infractions` | `status`, `access_status` |
| `invite_list` | `invites` |
| `questionnaires` | `result` |
| `set_friend_status`、`set_friend_statues` | `set_friend_statues` |
| `set_random`、`set_rank` | `set_rank` |

`external_links` 嵌在 `/account/auth/login` 返回体中，巡检单独检查；不是第 43 条独立资源入口。`emote_sequences`、`get_user_profile`、`outfit_defs` 在旧路由和静态库中均没有可验证的独立入口，仍应在实际客户端请求追踪中查明，不伪造成功结果。当前的 200 记录只证明候选 HTTP 合同；不能证明客户端能完成启动，也不能证明商城购买、消息投递或任务状态正确。
