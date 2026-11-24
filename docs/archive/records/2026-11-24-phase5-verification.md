# 第五轮整体验证

从当前工作区重新构建 API，两个服务分别离线运行 go test -count=1、go vet、go test -race -count=1、Linux 构建和 Windows amd64 交叉构建。房间测试以 SKYLA_TEST_API_BINARY 指向本轮构建的 API。

| 服务 | 常规测试 | race | vet、Linux 构建、Windows amd64 交叉构建 |
| --- | --- | --- | --- |
| API | 221 通过 | 221 通过 | 通过 |
| 房间 | 82 通过，2 跳过 | 82 通过，2 跳过 | 通过 |

数量含子测试，与第二轮基线一致。跨服务 HTTP 准入用例在普通与 race 测试均通过，使用真实 API 子进程和临时 SQLite。日志位于根目录 logs/phase5-*-test.jsonl、logs/phase5-*-race.jsonl 和 logs/phase5-*-vet.log。

静态核对确认两个 module 无跨服务 internal 导入及 replace 耦合，API 的生产 handler 不直接导入 repository 或 database/sql。201 条旧路由的方法一致，210 个方法、路径、回调绑定与重构前一致。房间代码索引已按第四轮后的目录重新生成。API 剩余兼容包装和其他业务的 SQL 编排留作后续独立改动。

真实 SkyENet 桥接加载和 Node 往返缺少已构建桥接库及对端，分别标为跳过。Windows 是交叉构建结果，真实 APK 双端同步与自然过图尚无本轮验收结果。
