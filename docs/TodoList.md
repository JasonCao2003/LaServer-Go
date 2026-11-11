# 开发与修复记录

## 2026-11-11：继续迁移账号资源

- 新增 get_collectibles、get_achievements、get_app_badge_number、get_achievement_defs、get_map_defs，累计 21 个已实现路径，扫描清单其余 180 个待迁移。
- 新增 schema v2，显式支持本项目 v1 开发库升级；创建成就、角标、收藏品和成就定义表，事务内更新版本，保留已有账号。不升级或导入旧 Python 实例数据库。
- 基础收藏品和地图定义通过 game_config_dir 只读复用旧配置，缺失或无效配置返回 503，不伪造空数据。
- 审阅修复：收藏品初稿遗漏 user_id 别名及匿名基础动作响应；新增失败回归后修复，匿名只读基础配置，账号覆盖条目要求有效会话，增加跨账号隔离断言。
- 兼容边界：未迁移自动全解锁、配置创建时间规则和成就定义高版本静态分流；真实客户端未验收。
- 检查：复用现有 WSL Go，执行本地集成测试、race、vet 和 Linux/Windows 编译；具体结果见 logs/。未提交、推送或修改旧实例数据。

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

## 2026-11-11：独立服务目录

- 将 API 源码、配置和 Go module 移至 services/api-server，入口改为 cmd/server，旧 entity 包迁至 internal/model；保留前一轮未提交功能改动。
- 新建 services/room-server 独立 module，尚无实现或运行入口。根 AGENTS.md 明确服务边界、目标分层、SQLite 所有权、独立构建和禁止跨服务 internal 导入。
- 调整生成脚本、验证脚本和文档路径；修正 Windows 写文件导致验证 shell 脚本带 CRLF 的问题，shell 文件固定 LF。迁移后本地测试、vet、Linux/Windows 编译通过；没有访问真实数据或新增环境。

## 2026-11-11：服务命名

- 按用户决定采用 api-server 和 room-server，更新服务目录、module/import 路径、入口名称、示例配置、验证脚本、文档及 AGENTS.md。旧 LaServer 参考路径和模型来源记录保留原名称。
- 复用现有 WSL 工具链运行测试、vet、Linux 编译及 Windows 交叉编译，均通过。未改动真实数据；此前未提交功能改动保留，未提交或推送。
