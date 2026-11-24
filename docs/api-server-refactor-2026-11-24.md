# api-server 文件整理与注释规范落地记录（2026-11-24）

## 范围

本次只调整 `services/api-server` 的代码组织、文件命名与注释，不改业务逻辑、协议行为、数据库结构或测试断言。所有改名使用 `git mv`（未跟踪的新文件使用 `mv`），未做 commit。

## 1. handler 文件重命名与路由拆分

| 原文件 | 结果 |
| --- | --- |
| `api.go` | `server.go`（API 构造与 Gin 引擎装配） |
| `endpoint.go` | `request.go`（请求解析、响应与错误输出） |
| `http_compat.go` | `http_helpers.go`（健康检查、UGC、CORS 预检） |
| `routes.go` | `router.go` + 8 个业务路由文件 |

`router.go` 只保留 `registerRoutes`；路由注册函数拆分为：

- `routes_account.go`
- `routes_player_state.go`
- `routes_commerce.go`
- `routes_quests.go`
- `routes_relationships.go`
- `routes_messaging.go`
- `routes_community.go`
- `routes_rooms_admin.go`

## 2. service 文件重命名（不拆包）

| 原文件 | 新文件 |
| --- | --- |
| `batch1_auth.go` | `auth_tokens.go` |
| `batch1_database.go` | `definitions_misc.go` |
| `batch1_resources.go` | `definitions_resources.go` |
| `batch1_schedule.go` | `event_schedule.go` |
| `batch1_spirit.go` | `spirit_shops.go` |
| `relationships_batch5.go` | `relationships.go` |
| `legacy_config.go` | `config_resources.go` |
| `legacy_resources.go` | `relationship_definitions.go` |
| `data.go` | `account_data.go` |
| `constellations.go` | `friend_status.go` |
| `messaging.go` | `mailbox.go` |
| `feed_common.go` | `social_feed_common.go` |
| `feed_routes.go` | `social_feed_routes.go` |
| `quests_account.go` | `season_quests.go` |
| `quests_player.go` | `player_quests.go` |
| `admin.go` | `admin_friends.go` |
| `commerce_static.go` | `commerce_config.go` |
| `commerce_read.go` | `commerce_shop.go` |
| `commerce_currency.go` | `commerce_wallet.go` |

保留说明：

- `legacy_contracts.go` 保持文件名不变，因为 `scripts/entities/import_contracts.py` 与 `scripts/docs/audit_api_task1.py` 硬编码该路径。
- 其余已按业务命名的文件（如 `bootstrap.go`、`profile.go`、`unlocks.go`、`geonotes.go`、`recordings.go`、`purchase_*.go`）不再改名。

## 3. 测试文件处理

### 3.1 重命名

| 原文件 | 新文件 |
| --- | --- |
| `handler/compat_test.go` | `handler/test_harness_test.go` |
| `handler/api_test.go` | `handler/core_flows_test.go` |
| `handler/batch_a_test.go` | `handler/inventory_contract_test.go` |
| `handler/batch_b_routes_test.go` | `handler/transaction_routes_test.go` |
| `handler/batch_b_test.go` | `handler/transaction_flow_test.go` |
| `handler/batch_c_test.go` | `handler/relationship_content_test.go` |
| `handler/batch_c_remaining_test.go` | `handler/content_contract_test.go` |
| `handler/batch_d_test.go` | `handler/stage_admin_test.go` |
| `handler/json_compat_test.go` | `handler/json_encoding_test.go` |
| `handler/login_compat_test.go` | `handler/login_test.go` |
| `handler/router_compat_test.go` | `handler/router_test.go` |
| `handler/task3_config_test.go` | `handler/config_resources_test.go` |
| `handler/task3_magic_test.go` | `handler/magic_config_test.go` |
| `handler/task3_schedule_test.go` | `handler/event_schedule_test.go` |
| `handler/task4_restart_test.go` | `handler/persistence_restart_test.go` |
| `handler/task4_transactions_test.go` | `handler/transactions_rollback_test.go` |
| `service/task4_day_test.go` | `service/pacific_day_test.go` |

### 3.2 归档

| 文件 | 处理 |
| --- | --- |
| `handler/task3_legacy_config_test.go` | 归档到 `docs/archive/tests/task3_legacy_config_test.go` |

归档原因：该测试在未设置 `LASERVER_GAME_CONFIG_DIR` 时始终 `t.Skip`，属于手动冒烟用例，不参与常规自动化。

## 4. 注释规范

### 4.1 文件头注释

`services/api-server` 下所有 Go 文件（含测试）在首行添加一行中文说明，描述该文件职责。示例：

```go
// 账号、鉴权与公共配置路由注册。

package handler
```

`model/doc.go` 保留原包说明，并在首行追加中文说明。

### 4.2 函数注释

- 范围：所有非测试文件（`_test.go` 除外）的顶层函数与方法，共 657 个函数。
- 处理方式：删除原有英文函数注释；每个函数上方保留且仅保留一行中文注释。
- 未处理：`_test.go` 函数、类型注释、结构体字段注释、函数体内行内注释。
- 示例：

```go
// 执行账号登录。
func (a *Service) Login(r Request, m object) (object, error) {

// 返回旧版 SQLite 表名。
func (Achievement) TableName() string {

// 构造读取货币的 SQL 语句。
func CurrencyRead(field string) Statement {
```

## 5. 脚本与文档同步

- `scripts/verify/gin_routes.py`：由只读取 `handler/routes.go` 改为读取 `handler/router.go` 与 `handler/routes*.go`，其余路由解析逻辑不变。
- `scripts/docs/generate_api_test_cases.py`：来源说明改为 `router.go` 与 `routes_*.go`，并重新生成 `docs/api-test-cases.md`。
- `scripts/docs/audit_api_task1.py`：路径未变（仍读取 `legacy_contracts.go`）；重新生成 `docs/api-login-resource-dependencies.md`、`json/api/task1-login-resources.json`、`json/api/task1-route-inventory.json`。
- 手工同步以下文档中的旧文件名引用：
  - `docs/api-route-verification.md`
  - `docs/api-testing.md`
  - `json/room-server/stage-1-inventory.json`

## 6. 验证结果

在 `services/api-server` 下执行：

```sh
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

并在仓库根执行：

```sh
sh scripts/verify/api-server.sh
```

结果：`gofmt` 无差异；`go test ./...` 全部通过；`go vet ./...` 通过；Linux 构建与 Windows 交叉构建通过；验证脚本输出：

```text
Tests, vet, Linux build and Windows cross-build passed.
```

补充校验：

- 非测试 `func` 上方均有且仅有一行中文注释（657/657）。
- 已无残留的英文函数注释。
- `grep` 确认除 `docs/archive/` 与历史计划外，文档、JSON 与脚本中没有旧文件名（`batch1_*`、`relationships_batch5`、`legacy_config.go`、`data.go`、`routes.go` 等）残留引用。

## 7. 后续建议

1. 当前 `handler`/`service` 仍为单包按文件分组，未拆业务子包。若后续要拆 `internal/service/<domain>`，需先处理 `Request`、`Object`、`Service` 的共享依赖与潜在循环引用。
2. `scripts/verify/gin_routes.py` 现依赖 `routes*.go` 命名。新增业务路由文件需保持 `routes_<业务>.go` 前缀，否则脚本不会纳入路由台账。
3. 归档测试如后续需要回归，可将其移回 `handler` 包并在 CI 提供 `LASERVER_GAME_CONFIG_DIR` 后再启用。
