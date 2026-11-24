# 两个 Go 服务的 Gin 迁移审查

审查日期为 2026-11-24。范围包括两个 module 的启动入口、HTTP 分发、中间件、请求解析、服务依赖方向、回归测试和路由清单生成器。两个 module 均未依赖 Gin。Gin 提供 HTTP 路由与中间件，项目的 handler、service、repository 分层属于本仓库的设计约束。[Gin 官方说明](https://github.com/gin-gonic/gin)

## 审查发现

| 优先级 | 现状及影响 | 迁移前的处理 |
| --- | --- | --- |
| P1 · 验收基线 | Linux 下 `TestMigrateStartAndShutdownWithoutRealtime` 在第二次执行 migrate 时失败。测试期待非零退出码，已达 v18 的实现提交空事务并返回成功。测试因此跳过后续启动、健康、退出及端口释放断言。 | 先统一重跑迁移的合同。按当前实现，应断言第二次成功、schema 和已有数据保持一致，再继续执行进程生命周期断言。[启动测试](../../../services/api-server/cmd/server/main_test.go) · [迁移实现](../../../services/api-server/internal/repository/storage.go) |
| P2 · API 路由 | `http.Server` 装配 `middleware.Access(api)`。API 手动分发 201 个 map 条目，另外单独处理健康检查和 UGC 通配路径。现有 handler 依赖 service，生产 handler 未直接导入 repository 或 database/sql。 | 改动边界可限于 HTTP 层。由 Gin 显式注册路径和方法，复用 service、数据库事务和现有 Server 的 TLS、超时与 Shutdown。[API 组装](../../../services/api-server/internal/app/run.go) · [路由](../../../services/api-server/internal/handler/api.go) |
| P2 · 路由清单 | 三个审计或生成工具按正则读取 Go 源码里的 `a.routes["…"]`。修改路由注册写法会造成漏计，影响逐接口覆盖报告。 | 同步改为读取实际注册的路由，按方法和路径与旧清单比对，单列健康、UGC、服务间认证及别名。[清单生成](../../../scripts/verify/inventory_routes.py) · [路由审计](../../../scripts/docs/audit_api_task1.py) · [用例生成](../../../scripts/docs/generate_api_test_cases.py) |
| 管理 HTTP · 适用范围 | 房间服务的 ServeMux 仅处理回环管理面的健康与就绪探针。就绪标志随真实传输循环变化，房间入场与消息收发经独立 ENet 传输。 | 如两个 HTTP 入口统一使用 Gin，只替换管理 Handler 的路由，保留现有监听、就绪回调、错误退出与资源回收。[房间生命周期](../../../services/room-server/internal/app/run.go) · [原生入口](../../../services/room-server/cmd/server/main.go) |

## 迁移验收清单

1. **API 路由合同。** 记录现有请求方法与路径对应的响应，包括管理 GET/POST、商品查询 GET、三个 OPTIONS 入口各自的 CORS 头、错误方法的 Allow 与 JSON、未知路径 404，以及旧清单中无实现时的 501 分支。验证尾斜杠不会新增重定向。UGC 需要覆盖 GET、Range、返回头及二进制响应。[路由实现](../../../services/api-server/internal/handler/api.go)
2. **请求与响应编码。** 保留 2 MiB 上限、text/plain JSON、BOM、UseNumber、尾随 JSON 拒绝，以及仅认证登录允许的表单与重复字段拒绝。保持现有错误体、Cache-Control 和响应中的字面量 &，复用现有解析和编码更易核对。[解析器](../../../services/api-server/internal/middleware/body.go) · [编码用例](../../../services/api-server/internal/handler/json_compat_test.go)
3. **调用方身份与日志。** API 管理端在无 token 配置时按连接对端判断回环地址。迁移须保留 RemoteAddr 语义和鉴权用例，确保任意转发头不能改变该判断。日志保持方法、路径和状态等元数据，避免记录含凭据的查询串或请求体。[管理鉴权](../../../services/api-server/internal/service/admin.go) · [访问日志](../../../services/api-server/internal/middleware/access.go)
4. **房间管理探针。** 检查 GET 和原 ServeMux 隐式支持的 HEAD、空响应体、健康 204、就绪 204/503、错误方法 405 和未知路径 404。覆盖传输启动、取消、监听失败和端口释放。房间协议、内存状态与认证调用继续由现有包负责。[管理测试](../../../services/room-server/internal/app/run_test.go)
5. **独立验证。** 仅在实际采用 Gin 的服务 module 增加依赖。分别运行两个根验证脚本，再核对路由和响应合同。API 成功构建、房间管理 HTTP 返回成功、实际客户端 ENet 入房分别记录。

## 本次实测

房间测试、vet、Linux 构建和 Windows amd64 交叉构建通过。真实 SkyENet 的两项可选测试跳过。API 的 handler、service、repository、config 和旧数据导入测试通过，cmd/server 的上述子进程用例失败。单独执行的 API vet、Linux 构建和 Windows amd64 交叉构建均通过。
