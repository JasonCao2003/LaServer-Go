# 开发与修复记录

## 2026-11-11：首批游戏 API

- 完成：16 个旧游戏路径及独立 HTTP 入口；状态清单见 json/api/legacy-routes.json，具体边界见 api-development.md。
- 待完成：其余 185 条扫描注册路径，以及扫描未覆盖的动态/装饰器注册、WebSocket、MQTT 和实时服务；真实客户端与旧数据迁移尚未验证。
- 不允许将当前预览实现作为完整旧服务替代品，亦不将继承的仅凭 user ID 登录行为视为安全认证。

### 数值关卡与最远进度

- 根因：初稿检查点参数仅接受字符串；请求使用 json.Number，存档读取却会得到 float64，导致相同数值关卡不能匹配；尚未读取旧关卡顺序定义。
- 修改：internal/gameapi/data.go 统一用 UseNumber 解码和 levelKey 比较，导入 checkpoint_defs 的首条顺序，更新已有关卡并维护最远标记；scripts/entities/import_contracts.py 负责提取定义。
- 验证：TestNumericCheckpointUpdatesAndFurthestProgress 在修复前以数值 ID 返回 400 失败，修复后通过，检查连续关卡、同关卡更新不重复与最远标记。
- 适用范围：当前旧 HTTP 检查点协议；真实客户端效果尚未验证。

### 季卡每日奖励

- 根因：初稿仅读取货币，尚未迁移旧 get_currency 的每日季蜡副作用；旧旁路文件与数据库提交分离也不能保证跨失败的一致性。
- 修改：internal/gameapi/data.go 与 storage.go 将领取标记和货币增加放入同一 SQLite 事务，按太平洋日期使用唯一键。
- 验证：TestVIPDailyCurrencyBonusIsAtomic 修复前为 0 根而失败，修复后 8 个并发请求只发放 1 根。
- 迁移约束：旧 season_pass_daily.json 标记未导入，不能直接切换真实玩家数据。
