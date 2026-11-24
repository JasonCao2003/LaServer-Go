# 第三轮 API 重构

以 f4bebf6 和第二轮工作区基线为起点。Gin 的九组业务路由分别接收对应领域的具体 Service，UGC 接收 social.Service；组合根通过具名字段配置依赖。九个领域的兼容层去掉无引用的声明，stage 直接调用 core。其他领域保留业务仍使用的旧名称，避免全量替换。

光翼兑换仍由 inventory 决定事务边界、初始化条件和声望奖励。repository 负责事务内的数据库操作以及光翼列表扫描。提交后读取光翼和货币，复用原有货币快照读取器。其余资产操作可按独立变更逐步迁移；本轮仅覆盖光翼兑换这个完整操作。

在 Fedora Linux/amd64 的两个独立 module 内按第二轮基线验证。对比 210 个方法、路径、回调绑定和全部 201 条旧路由；API 全量测试和 race 各 221 项通过，vet、Linux 构建及 Windows/amd64 交叉构建通过。房间服务使用本轮编译的 API 二进制执行真实 HTTP 认证测试通过。运行记录见根 logs/phase3-api-final.jsonl、logs/phase3-api-race.jsonl、logs/phase3-admission-e2e.jsonl。未执行真实 APK 或 ENet 联机。未调整 schema、迁移版本及模块依赖。
