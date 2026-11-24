# API Task 4：旧数据与事务验收

归档说明：本文引用的三份原始日志未随当前检出提供，保留文件名供追溯。

日期：2026-11-24。范围仅为 `api-server`；旧 LaServer 实例没有切换。

## 旧数据盘点与映射

| 来源 | 发现 | Go 目标及处理 |
| --- | --- | --- |
| `LaServer/services/game-api/var/data/instance/users.db` | 71 张业务表；`user` 2 行，其他表含大量无现存用户的历史行 | 同名且列兼容的表按 Go 外键顺序导入；孤儿行跳过并逐表报告 |
| `LaServer/services/game-api/var/data/legacy/users.db` | 仅 `user` 表且为 0 行 | 可预检，无实际账号可导 |
| `instance/recordings` | 11 个 JSON/NDJSON 文件，共 8,385 字节 | 显式指定全新目标目录后复制，供 Go 的 `InstanceDir/recordings` 使用 |
| `instance/feed_extra` | 1 个 JSON 文件，共 175 字节，但没有对应的 `social_feed_post` | 记为未导入；只有对应动态存在时才转入 `social_feed_extra` |
| `instance/stage` | 空目录 | 无文件需要导入 |
| `instance/magic_defs_config.json` | 1 个配置文件，共 1,008 字节 | 当前未导入，报告中明确计入部分导入 |

列名差异：`quest_stat.stat_category → category`，`season_quest` 和 `season_quest_reward` 的 `daily_quest_def_id → quest_id`。导入器不会猜测其他表的业务含义；没有 Go 对应表、必要列或有效父记录的行会在报告中列为跳过。预检用只读 URI 打开旧库，并在内存目标库试导；正式导入只允许**不存在的目标库路径**，失败时移除自己新建的数据库文件。

隔离实测报告：正式隔离导入（原日志 `logs/api-task4-isolated-import-v5-20261124.json`）。71 张表中，49 张同名表共 68,392 行，导入 4,102 行，跳过 64,290 行；22 张无对应 Go 表共 16,720 行，均未导入。`unlock_item` 跳过 37,907 行，`status_unlock` 跳过 10,633 行，`level_pickup` 跳过 10,053 行；无对应表的 `request_log` 为 15,514 行。`recordings` 11/11 已复制；`feed_extra` 0/1（源库动态表为空，附件没有可显示的父动态）、`magic_defs_config.json` 0/1。实际目标的 2 条 `user_currency` 和 2 条 `user_data` 与源库中可导的记录逐行相符；货币字段合计及 SQLite 外键检查也通过。报告没有输出账号 ID 或存档内容。

这次导入明确标为 `partial=true`，**不是完整旧实例迁移**。默认模式拒绝上述跳过；只有显式 `-allow-partial` 才可生成目标。项目决定旧数据不必完整迁移，新实例可从空 SQLite 开始。如果要保留特定旧账号或内容，必须另行确认源库中仅 2 条现存用户、孤儿行和未迁移配置对这些目标的影响；不能把本次隔离导入视为账号恢复成功。源库中没有 `season_pass_daily` 表，不能从这份数据恢复旧每日领奖标记。

预检与隔离导入示例（从 `services/api-server` 执行，目标路径事先不得存在）：

```sh
go run ./cmd/legacy-import -mode preflight -source /path/to/old/instance/users.db
go run ./cmd/legacy-import -mode import -source /path/to/old/instance/users.db -target /path/to/new/api.db -recordings-dir /path/to/new/recordings
# 仅在审阅报告并接受丢弃项后，显式加入 -allow-partial
```

`-recordings-dir` 应与新服务 `InstanceDir/recordings` 对应，并且不能位于旧实例目录内；数据库目标也必须位于旧实例外。报告中的 `source_rows`、`imported_rows`、`skipped_rows`、`sidecars` 和 `key_checks` 是复核依据；源库始终只读。应在旧服务停止写入或从一致性快照导入，以免两次读取之间源库变化。导入器不迁移运行中的服务进程状态，也不提供正式切换、备份和回退操作。

## 事务与重启验证

临时 SQLite 回归覆盖：余额刚好足够时的并发重复购买；解锁落库失败后的货币回滚；模拟收据订单落库失败后的货币回滚；赛季奖励写余额失败后的领取标记回滚；宝箱兑奖失败后的钱包与库存回滚。重复领取、购买和收据在真正关闭并重新打开数据库连接后仍只入账一次。`America/Los_Angeles` 的 2026 年春季、秋季夏令时切换日界测试通过。真实支付凭据仍不发货；此处收据测试仅在显式启用的本地模拟模式下运行。

邮件已读状态、礼物、社交动态和留影在重新打开 SQLite 后可读取，跨账号错误会话与越权操作有独立测试。`get_pending_messages` 同时合并数据库中的待领礼物与 API 进程内聊天队列：礼物跨重启保留，未消费的普通聊天消息跨重启丢失。当前只适合单个 API 进程提供聊天；多 API 实例会有独立队列，无法保证消息到达另一个实例。Task 5 应通过明确认证、失败响应和超时的 API↔实时服务协议转发在线消息；在联调完成前不要部署多实例聊天。

自动化结果仅证明本地临时库和隔离旧库样本上的这些行为，不等于 0.1.8/0.3.11 客户端或正式数据切换验收。

最终回归记录：Go 全量测试（原日志 `logs/api-task4-full-test-final-20261124.log`）、go vet（原日志 `logs/api-task4-vet-final-20261124.log`）；`git diff --check` 通过。Windows 当前账号无创建目录符号链接的权限，链接目录专项测试被跳过；直接位于旧实例内的目标拒绝用例已通过，链接解析逻辑还需在支持符号链接的环境验证。
