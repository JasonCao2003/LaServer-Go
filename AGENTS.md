# LaServer-Go 项目约束

## 服务边界与目录

- 同一仓库维护两个独立服务：`services/api-server/` 与 `services/room-server/`。每个服务使用独立 Go module、入口 `cmd/server/main.go`、配置、测试和构建流程；不在根目录建立统一服务 module，不要求同时启动、停止或部署。
- api-server 负责 HTTP、管理 API 和 SQLite 持久化业务；room-server 负责 ENet/UDP、连接、协议版本和房间内存状态。
- 禁止直接导入另一个服务的 internal 包、共享内存状态或通过 replace 指令耦合实现。服务交互使用明确的网络协议，不把共享数据库作为服务间通信方式。
- 暂不提取公共 module；确有稳定复用需求时先说明边界。依赖只添加到实际使用它的服务中。

## 服务内部组织

- API 目标分层：internal/handler（HTTP 与 DTO）、service（规则与事务边界）、repository（SQL）、model（持久化记录）、middleware（通用 HTTP 处理）、config（配置解析）、app（依赖组装与生命周期）。依赖方向 handler → service → repository，不允许 handler 直接操作数据库。
- 实时服务按 internal/transport、protocol、session、room、config、app 组织，不强行套用 HTTP 分层。
- 当前 API 已有 internal/gameapi 是迁移过渡包，仍包含 HTTP、业务和数据访问；后续按完整业务单元逐步拆入上述层次，保持现有测试和协议行为，不新建巨型通用文件。internal/model 已保存旧表模型，不能直接当作 API DTO。
- SQL 迁移最终归属对应服务的 migrations/；现有迁移常量在后续提取时必须保留版本和事务语义，不得为整理目录改写已发布迁移。
- 未实现部分不创建假启动入口或伪造成功接口；目录规范与功能已完成状态应分别说明。

## 数据与验证

- 最终数据库为 SQLite，不引入 Redis、MySQL 或其他数据库服务。实时服务不直接写 API 数据库。
- 旧 LaServer 作为只读兼容参考，不覆盖其未提交代码、数据库或证书。真实数据迁移、破坏性操作、commit/push 按用户授权执行。
- 开发和验证优先复用现有 WSL Go 工具链。测试授权沿用当前会话范围；不自动扩大到生产或高成本任务。
- 文档放 docs/，报告放 json/，脚本放 scripts/<功能>/，日志放 logs/，图片放 pic/；服务独有运行数据放服务自身 data/，不得提交真实数据、凭据或构建缓存。
- 仓库根运行检查脚本；手动 Go 命令必须切入目标服务目录执行，不能假定根目录 go test ./... 覆盖多个 module。
