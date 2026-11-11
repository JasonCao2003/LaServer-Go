# LaServer Go 重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 上述技能是上游模板建议；执行时先检查可用性，不假定已安装。使用子代理、提交或推送须符合当前用户授权。

**Goal:** 分阶段将 LaServer 迁移为可独立启动、业务边界明确且兼容既有客户端的两个 Go 服务。

**Architecture:** 同一 Go module 下分别构建 game-api 和 realtime-server。业务 API 集中管理业务规则与 SQLite，实时服务维护 ENet 与房间状态；分别管理配置和生命周期，先 API 与管理端、后实时服务。

**Tech Stack:** Go、net/http、database/sql、SQLite、slog、Go 模板与 embed；SQLite 驱动和 ENet 库经技术验证后锁定。

**Spec:** [设计草案](../specs/2026-11-11-go-rewrite-design.md)

**状态:** 本计划是用户请求的规划交付；设计未审批，不执行实现。M0 后依据协议清单逐模块补充精确的路径、请求样本和断言，不能把后续里程碑当成已完成细化的逐接口任务。

## Global Constraints

- 最终数据库使用 SQLite，不引入 Redis、MySQL 或其他数据库服务，不规划数据库换型；不新增前端构建运行时。
- 持久化业务数据由业务 API 管理，实时房间状态保存在实时服务内存中，不通过 Redis 共享状态。
- 日志放 logs/，图片放 pic/，报告放 json/，脚本放 scripts/<功能>/，发布文件放 dist/<版本>/。
- 旧 instance 与 legacy 数据库分别盘点和迁移，不合并；使用副本验证，旧服务保持可回退。
- 目标平台提案为 Windows amd64 与 Linux amd64；开发优先复用 WSL 环境。
- Git commit/push 需要明确授权；运行测试前询问用户，不因计划中的测试命令视为已获授权。
- 不上传真实账号、证书、原始抓包、旧仓库历史或未经许可的资源。

## Review Focus

1. 真实 APK 与上报构建号不一致：兼容报告同时记录两者（M0、M5）。
2. 同一奖励并发领取：事务保证只发放一次（M3）。
3. 无效配置或端口冲突：非零退出并释放已开启监听器（M1、M6）。
4. 旧数据库时间格式及主库/legacy 差异：导入副本并核对，不修改原库（M2）。
5. 未鉴权管理请求和日志泄露：拒绝访问，令牌与账号敏感信息脱敏（M4）。

## 文件与模块边界

```text
cmd/game-api/main.go             业务 API 入口
cmd/realtime-server/main.go      实时服务入口
internal/app/                   API 配置、依赖组装、生命周期
internal/realtimeapp/            实时端配置、依赖组装、生命周期
internal/account/               身份与会话
internal/social/                好友与邀请
internal/inventory/             货币、背包、装扮与奖励
internal/quest/                 任务及领取规则
internal/room/                  房间发现、房间和玩家状态
internal/protocol/              客户端协议适配
internal/transport/enet/        ENet 封装
internal/admin/                管理鉴权、API 与页面
internal/storage/              SQLite 连接和版本检查
migrations/                    版本化 SQL
config/game-api.example.json   API 无凭据配置示例
config/realtime.example.json   实时端无凭据配置示例
tests/compat/                  脱敏兼容用例
scripts/verify/                 环境与发布检查
```

不创建空目录堆砌结构；实现对应模块时才添加。各业务模块自有 handler.go、service.go、repository.go，避免全局共享 helpers 包。

## M0：建立可审计的迁移基线

**文件:** 新建 docs/compatibility.md、docs/api-inventory.md、docs/data-migration.md、json/compatibility-matrix.json。

**输入:** 旧项目路由注册、business/account、business/service、静态响应、数据库模型、协议版本分流和客户端兼容记录。
**产出:** 逐端点清单（路径、方法、认证、读写对象、协议族、动态/静态/桩实现、证据路径）；逐客户端验收矩阵；两类数据库迁移映射。

- [ ] 只读盘点现有端点、管理动作、Socket.IO 和 MQTT 依赖，不将注册路径等同于功能已实现。
- [ ] 记录旧项目 commit 与 dirty 状态，脱敏保存证据；不复制未提交源码到公开仓库。
- [ ] 将 0.1.8、0.3.11 的登录、存档、房间发现列为第一批闭环，明确未知项。
- [ ] 共存默认使用独立数据副本和测试实例；未迁移业务留在旧实例，同一账号不得跨新旧实例写入。对外切换前完成闭环全部依赖；需要路由分流时先单独评审数据所有权和分流规则。
- [ ] 为每个模块编写可执行的兼容任务单；明确 HTTP 状态、字段类型、错误行为与响应样本来源。
- [ ] 验收：所有已注册入口均有分类；每项兼容结论有实际证据或明确标注待验证。

## M1：可诊断的业务 API 启动入口

**文件:** go.mod、cmd/game-api/main.go、internal/app/{config,runtime}.go、internal/app/runtime_test.go、config/game-api.example.json。
**接口:** `LoadConfig(path string) (Config, error)`；`Run(ctx context.Context, cfg Config) error`。Config 包含 HTTPAddr、AdminAddr、DatabasePath、TLSCertPath、TLSKeyPath；配置不得携带真实凭据。

- [ ] 检查并复用 WSL/Go 工具链，记录版本并锁定 go.mod，不覆盖系统环境。
- [ ] 编写 TestLoadConfigInvalid、TestRunPortConflict、TestRunCancellation：错误可定位、返回错误、取消后监听器可重新绑定。
- [ ] 经用户同意运行 `go test ./internal/app -run 'TestLoadConfigInvalid|TestRunPortConflict|TestRunCancellation'`，确认实现前失败。
- [ ] 实现参数入口 `game-api --config <path>`、配置校验、/healthz、slog、信号退出与启动回滚。
- [ ] 运行同组测试，预期通过；执行 gofmt、go vet ./...；记录真实结果。
- [ ] 更新启动文档，检查 diff 后建议提交 `feat(api): add independent server startup`，授权后提交。

## M2：SQLite 与存档迁移基础

**文件:** internal/storage/{sqlite,migrate}.go、internal/storage/migrate_test.go、migrations/、scripts/migrate/、docs/data-migration.md。
**接口:** `Open(ctx context.Context, path string) (*sql.DB, error)`；`CheckVersion(ctx context.Context, db *sql.DB) error`；独立 `game-api migrate --source <copy> --target <new-path>`。

- [ ] 以 M0 映射确定第一批账号表和外键，验证 SQLite 驱动的目标平台构建能力并锁版本。
- [ ] 写 TestMigrationPreservesAccounts、TestMigrationMixedTimestamps、TestMigrationRefusesExistingTarget；断言 ID、余额和时间归一化正确，拒绝覆盖已有文件。
- [ ] 获授权后执行 `go test ./internal/storage`，先失败，再实现连接、显式迁移与只读版本检查，重复验证。
- [ ] 使用合成数据或经许可的隔离副本，核对行数、外键、余额汇总和关键记录；禁止接触真实原库写入。
- [ ] 在 docs/data-migration.md 写清备份、停写、迁移失败退出和切回旧服务步骤；检查 diff 后请求提交授权。

## M3：游戏业务 API 分批迁移

**文件:** internal/account/、internal/inventory/、internal/social/、internal/quest/、internal/room/discovery.go、tests/compat/。
**边界:** handler 不直接写库；跨模块用 service 接口；事务由业务操作拥有；客户端响应通过协议 DTO 输出。

- [ ] 按 M0 为账号认证/会话 → 存档读取 → 房间发现 → 好友 → 背包/货币 → 任务分别建立任务单；每批列精确旧路径、Go 文件、DTO、service 签名和兼容断言后再编码。
- [ ] 每批先写旧请求与新响应的兼容测试；包含缺字段、空值、未知版本、过期会话和原有错误语义。
- [ ] 对奖励操作加入 TestClaimRewardConcurrent 和 TestClaimRewardRollback，断言只发一次、失败无部分写入。
- [ ] 获授权后按模块运行 `go test ./internal/account ./internal/inventory ./internal/social ./internal/quest ./internal/room ./tests/compat`，先失败再最小实现；并发用例经授权运行 race 检查。
- [ ] 写接口覆盖表，已迁移、保留旧实现和未支持入口分别标注；未支持接口不得返回伪造成功。
- [ ] 每批通过兼容验收后才迁移下一批；逐批建议独立提交。

## M4：浏览器管理端与统一业务操作

**文件:** internal/admin/{auth,handler,assets}.go、internal/admin/templates/、internal/admin/static/、internal/admin/handler_test.go。
**接口:** 管理 handler 调用 M3 service，使用独立管理凭据；默认仅监听回环地址，不复用玩家令牌。

- [ ] 盘点旧面板功能，以账号查询、解锁/奖励、好友管理、服务状态为批次；页面布局在实现前另行确认。
- [ ] 写 TestAdminUnauthorized、TestAdminMutationCSRF、TestAdminLogsRedacted，断言未认证不访问业务、跨站写请求被拒、日志不含凭据。
- [ ] 获授权后运行 `go test ./internal/admin`，先失败，再实现鉴权、管理 API、模板页面和 embed 资源；重复验证。
- [ ] 管理修改复用业务事务与校验；记录操作者、动作和结果，避免记录完整玩家敏感信息。
- [ ] 验收浏览器操作、凭据配置与重启持久化；不再依赖 Tkinter 直写库。

## M5：ENet 技术验证与实时迁移

**文件:** internal/transport/enet/、internal/protocol/、internal/room/、tests/compat/enet/、docs/enet-decision.md。
**前置:** 单独批准兼容验证后再做实验；迁移期保留旧 Node.js 服务和明确端口配置。

- [ ] 新增 cmd/realtime-server/main.go、internal/realtimeapp/{config,runtime}.go 和 config/realtime.example.json，实现 `realtime-server --config <path>`；配置含 ENet 地址、本机管理 HTTP 地址，启动不依赖 API 在线。
- [ ] 对 Go ENet 候选验证许可证、维护状态、目标平台、CGO 和动态库要求，记录决定，不能只以 API 类似判定兼容。
- [ ] 定义封装接口：`Start(ctx context.Context) error`、`Events() <-chan Event`、`Send(peerID uint64, channel uint8, payload []byte, reliable bool) error`、`Close() error`；Event 包含 Kind、PeerID、Channel、Payload，业务不得依赖候选库类型。
- [ ] 基于脱敏协议证据补充逐包断言，覆盖校验和、可靠/不可靠通道、分片、重复包、断线和重连；先确认候选与真实客户端建立连接。
- [ ] 技术验证通过后，为房间定义单一状态所有者，编写加入/离开/广播/过图测试，再迁移 packet dispatcher 和具体版本 codec。
- [ ] 获授权后运行 `go test ./internal/transport/enet ./internal/protocol/... ./internal/room ./tests/compat/enet/...`；实际 APK 验收分别记录 0.1.8 和 0.3.11，双端同步与自然过图单独记录。
- [ ] 不通过则保留旧实时服务，报告具体报文差异，暂停全 Go 切换；不宣称已支持全部协议族。

## M6：独立发布与可回退切换

**文件:** internal/app/runtime.go、internal/realtimeapp/runtime.go、scripts/build/、scripts/verify/、docs/{startup,release,rollback}.md。

- [ ] HTTP 与管理端纳入 API 生命周期，ENet 与实时管理 HTTP 纳入实时服务生命周期。分别写 TestPartialStartupCleanup；实时端写 TestShutdownWithActiveRoom；跨进程验收 TestAPIStartsWithoutRealtime、TestRealtimeStartsWithoutAPI 和 TestStoppingOneServiceKeepsOtherRunning。独立启动不代表所有运行时操作均不依赖另一服务；认证依赖异常须明确报错。
- [ ] 获授权后运行对应测试；在 Windows amd64、Linux amd64 构建，真实核验 CGO/动态库依赖，记录启动前提。
- [ ] 检查 dist/<版本>/ 目标是否存在，拒绝覆盖；分别发布 game-api、realtime-server 二进制及各自示例配置，并附迁移说明和校验和，不打包数据或证书。
- [ ] 用隔离目录执行干净启动、端口冲突、异常配置、停机和重启验收，日志与报告按类型归档。
- [ ] 切换前取得生产数据操作授权；停止旧端写入、备份、迁移核验、切换。保留旧服务及其数据；若新端已产生写入，先保存新数据并明确回迁方案，禁止直接切回丢弃数据。
- [ ] 发布前检查 git status、相关 diff 与敏感文件；授权后才 commit/push 或创建 Release。

## 完成定义与验证状态

所有约定第一批客户端完成真实登录、存档、进房、双端同步及过图验证，两个服务均可按文档独立构建和启动，可在干净目标环境复现，切换和回退办法经过隔离演练，才视为首个重构版本完成。其他协议族以兼容矩阵为准。

当前仅完成规划文档编写：未安装依赖、未写产品代码、未运行测试、未验证性能或客户端兼容性。无工期数字承诺；M0 完成后再按接口数量和协议差异估算。
