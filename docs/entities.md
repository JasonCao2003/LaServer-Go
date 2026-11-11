# 旧服务实体映射

本次只创建持久化数据结构，不包含建库、迁移、查询、默认值构造器或 HTTP 接口。`internal/entity/` 是旧表兼容层；后续业务模块使用独立 DTO，不把数据库实体直接返回客户端。

## 来源与覆盖

- 旧 `services/game-api/infrastructure/database/models.py`：65 个模型、579 个字段。
- 业务中的直接建表声明：7 个表、31 个字段，包括 color_dye_currency、item_dye、boat_follow、response_override、request_log、user_activity、status_unlock_deleted。
- 合计 72 个实体、610 个字段；逐字段来源、声明、主键、可空性和约束见 `json/entities/legacy-schema.json`。
- 基于当前工作区源码声明生成，不读取真实数据库；不保证声明与历史实例实际结构相同。旧迁移脚本和管理端有重复建表定义，未将重复表扩展为第二套实体；数据库迁移前仍需单独核对结构差异。

## 类型与规则

- SQLite 整数映射 int64，浮点映射 float64，字符串和 TEXT 映射 string，模型 Boolean 映射 bool。
- 可空列映射指针。SQLAlchemy 仅设置 default 并不意味着 NOT NULL，因此很多字段使用指针；nil 与零值不能互换。
- DateTime 和 FlexibleDateTime 映射 time.Time 或 *time.Time，整数时间戳保留 int64；FlexibileDateTime 的混合格式读取兼容须在后续 repository 中实现，实体本身不负责转换。
- JSON TEXT、base64 存档仍保留 string，不猜测结构或解码。直接 SQL 建表的 INTEGER 标志仍保留整数类型。
- 保留旧表名、列名与复合主键字段。外键以 ID 字段表达，不复制 ORM 的惰性关系或循环对象图。
- db 标签是列名元数据，database/sql 不会自动据此映射字段。json 标签统一为 `-`，防止直接序列化泄漏；API DTO 在后续独立定义。
- 默认值、唯一约束、外键、索引、自动更新时间及 UUID 生成不由 Go 零值实现；原声明保存在报告，后续迁移和 service 分别负责。

## 再生成

在项目根目录运行 `python scripts/entities/import_legacy.py --source ../LaServer`。脚本只解析 Python AST，对额外 CREATE TABLE 字符串使用临时内存 SQLite 检查列定义；不会导入或启动旧应用，不读取或写入真实数据库。重新生成会覆盖同名生成文件，修改前需保护手工改动；不会自动删除已不在源中的旧文件。

实体包本身无第三方依赖。接口阶段引入 SQLite 驱动后，项目 go.mod 最低版本已更新为 1.26.0，当前验证工具链为 Go 1.27.4。

## 验证状态

已核对源码模型字段数及生成清单数量。实体创建阶段未找到 Go；接口阶段经授权安装 Go 后已执行 gofmt 并随项目编译。未验证实体与真实存档的读取兼容性。

有 Go 环境后可先运行 `gofmt -w internal/entity`、`go vet ./...` 做静态检查。数据库读写与迁移测试留待对应实现，并在用户授权后运行。
