# 第四轮 UDP 重构

入房解码现在一次取得 JoinGame 指纹和准入凭据。保留独立解码入口及 284/542 的凭据布局判断。房间包按事件循环、准入、活跃消息分发和连接回收组织。连接、关卡和快照状态仍归事件循环所有。认证 worker 只返回结果，连接世代校验仍在事件循环内完成。发包失败和 ACK 处理沿用第二轮基线语义。

房间 module 的常规及 race 检查各 82 项通过、2 项跳过。跑过使用第三轮 API 二进制的 HTTP 准入集成测试，以及连接复用、认证队列已满时取消、快照发送失败和协议字节回归。vet、Linux 构建和 Windows amd64 交叉构建通过。日志位于 logs/phase4-room-tests.jsonl 和 logs/phase4-room-race.jsonl。真实 SkyENet 启动及 Node 互通两项按条件跳过，真实 APK 联机未验证。
