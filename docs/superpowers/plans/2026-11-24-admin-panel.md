# 独立管理面板开发计划

**目标**　在 `services/admin-panel/` 交付可独立启动的 Go 模板 Web 面板，首版覆盖服务状态和好友管理。

**架构**　独立 Go 1.26 module。采用标准库 `net/http`、`html/template` 与 `embed`。浏览器请求面板，面板通过 HTTP 访问 API 和房间服务。账号、好友写入及 SQLite 由 API 持有。

**设计依据**　[独立管理面板设计](../specs/2026-11-24-admin-panel-design.md)

## 工程与接口约定

| 文件 | 职责 |
| --- | --- |
| `go.mod`、`cmd/admin-panel/main.go` | 独立模块、配置入口、HTTP Server、信号与关闭 |
| `internal/panel/config.go` | 本地监听、上游地址、超时、环境凭据 |
| `internal/panel/server.go`、`auth.go` | 页面路由、登录、会话、Host/Origin 和 CSRF 校验 |
| `internal/panel/upstream.go`、`friends.go` | 固定上游请求、好友结果解析与表单处理 |
| `internal/panel/templates/`、`static/` | 概览、登录、好友页面及内嵌资源 |
| `config/admin-panel.example.json`、`README.md` | 示例配置、启动、令牌设置、测试和排查 |
| `scripts/verify/admin-panel.sh` | 从仓库根进入面板 module 执行检查，输出至根 `logs/` |

配置采用 `listen`（默认 `127.0.0.1:18080`）、`api_url`、`room_url` 和 `upstream_timeout_ms`（默认 2000）。两种独立凭据分别由 `LASERVER_PANEL_PASSWORD` 与 `LASERVER_PANEL_API_TOKEN` 提供。API 的 `admin_token` 需与后者相同。为独立启动验收，上游 URL 可指向当前未监听的有效地址。

## 任务一　独立启动和配置

- [x] 在面板 module 中添加配置测试。覆盖默认监听、无效超时、缺少密码或令牌、非回环监听、非回环明文上游，以及含凭据、路径、查询或片段的上游地址。来源校验使用配置的浏览器访问源，HTTP Host 只允许配置对应的回环主机及端口。
- [x] 实现 `LoadConfig(path string) (Config, error)` 和 `New(cfg Config) (http.Handler, error)`。增加 `-config` 参数与示例 JSON。入口设置服务器超时，捕获 SIGINT/SIGTERM，等待有界关闭。启动只验证配置和模板。
- [x] 启动独立二进制，确认两上游未启动时登录页可访问。发送 SIGTERM，验证正常退出和监听端口释放。

实测：两个回环上游均未监听时，独立面板进程返回登录页 HTTP 200。SIGTERM 退出码为 0，端口可重新绑定。配置与 Host 测试、race、vet 和构建通过。

## 任务二　登录和服务概览

- [x] 添加测试覆盖未登录访问、错误密码、过期或退出后的会话、跨站 POST、Host 不匹配和无效 CSRF。测试上游探测的正常状态、503、连接错误和超时。
- [x] 实现 `GET /login`、`POST /login`、`POST /logout` 和 `GET /`。登录使用随机服务端会话，固定过期并清理过期记录。Cookie 采用 HttpOnly、SameSite=Strict。登录 POST 检查来源，已登录管理 POST 同时检查来源与会话绑定的 CSRF。敏感页面设置 `Cache-Control: no-store`。
- [x] 固定 API `GET /healthz` 与房间 `GET /healthz`、`GET /readyz`。独立探测并渲染各项状态，API 200 与房间 204 作为正常值，房间就绪 503 单独显示。HTTP 客户端禁止自动重定向，限制响应体，默认校验 TLS。
- [x] 验收一项上游超时仍能读取另一项状态，探针请求不携带管理令牌。

验证：配置与 Host 检查、登录、过期会话、退出、跨站及 CSRF 拒绝通过。状态测试覆盖在线、未就绪、超时、不可达和禁止重定向，确认探针不携带令牌。全量测试、race、vet、构建通过。

## 任务三　好友管理

- [x] 添加 HTTP 合同测试。验证上游路径、POST JSON、`X-Admin-Token`、HTTP 200 的 `ok:false`、失败条目及写入超时。对昵称和备注中的 HTML 断言转义。未登录、CSRF 失败、非法动作与清空确认错误均不得触发上游写入。
- [x] 实现 `GET /friends`，通过 `POST /admin/api/friends_list` 根据用户 ID 和可选关键字查询。服务端保留管理员令牌，模板只渲染所需字段。
- [x] 实现 `POST /friends/set` 和 `POST /friends/clear`。限制表单大小、字段长度和动作枚举。清空时要求操作者再次输入目标用户 ID。上游使用 `friends_set` 的 `user`、`friend`、`action` 字段和 `friends_clear` 的 `user` 字段。
- [x] 回显完整业务结果，包含清空中的成功数量与失败条目。写入超时显示结果待确认，引导重新查询，禁止自动重试。日志只记录请求 ID、动作与结果。

验证：回环模拟 API 合同测试覆盖查询、恢复、清空、部分失败、模板转义、鉴权头和超时不重试。未登录、无效 CSRF、非法操作和确认不匹配均未到达上游。全量测试、race、vet 和构建通过。

## 任务四　联调及交付

- [x] 使用已编译 API、临时 SQLite 和两名测试账号，核对查询、增加、移除、恢复与清空的结果。重启 API 确认持久结果。通过模拟上游验证批量部分失败。检查错误令牌、服务离线及请求超时。
- [x] 浏览器核对登录、状态页、表单反馈、清空确认、键盘操作和窄屏布局。检查页面及日志中没有凭据。
- [x] 在面板 module 内执行 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`。从仓库根执行面板及既有 API 检查脚本。记录跳过的测试和原因。
- [x] 更新根 README 和模块说明，给出单独构建与启动命令、API 令牌配对方式及服务离线状态。项目约束目前列出两个服务，实现时更新该描述以明确面板为第三个独立 module。提交与推送依项目授权执行。

验收重点包括旧式回环免令牌、跳转导致凭据外传、登录与好友表单冒用、写请求结果不确定、批量部分成功以及失联时的页面响应。账号增删、公告、封禁、日志流与进程控制留待后续单独设计。

任务四验收通过。真实 API 与面板使用临时 SQLite 和两个测试账号完成好友全流程、重启持久化、离线和错误令牌检查。Chrome 以 360px 宽度验证布局、键盘登录及清空确认。面板全量测试、race、vet、构建以及 API 检查脚本通过。凭据未出现在联调页面或面板日志中。复跑命令见[面板 README](../../../services/admin-panel/README.md)，联调日志位于根目录 `logs/admin-panel-task4-smoke.log`。
