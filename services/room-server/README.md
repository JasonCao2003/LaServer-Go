# 实时联机服务

独立 Go module，计划使用 cmd/server/main.go 入口和 internal/transport、protocol、session、room、config、app 目录。当前实现包括时钟 Ping/Pong 编解码、EnterGame 名册编码、两字节 JoinGame 指纹识别、LevelUpdate 解码、独立房间配置、连接代次与在线会话登记、服务启停逻辑，以及 Windows amd64 原生 SkyENet 适配器。示例见 [realtime.example.json](config/realtime.example.json)，其 UDP 地址仅绑定回环。已提供 Windows 启动入口并接入 Go API 认证、初始名册、离房通知和时钟响应。完整快照、移动、过图和多版本玩法仍待实现。该模块目前使用 Go 标准库，无第三方依赖。

此服务独立构建和部署，不导入 api-server 的 internal 包，不直接读写其 SQLite。ENet/UDP 兼容验证和实现按项目开发计划推进。

## Windows 原生传输验证

需要 Go 1.26、Visual Studio 2022 C 工具链、CMake，以及 sky-enet 0.1.5 中的 ENet C 源码。可从已经安装的旧服务依赖目录取得该版本的源码。以下命令在 room-server 目录运行，路径按实际本机环境填写。

~~~powershell
cmake -S internal/transport/native -B ../../tmp/room-enet-bridge -G 'Visual Studio 17 2022' -A x64 "-DSKYLA_ENET_SOURCE=<sky-enet 0.1.5 的 enet 目录>"
cmake --build ../../tmp/room-enet-bridge --config Release
$env:SKYLA_ENET_BRIDGE_TEST_DLL = (Resolve-Path '../../tmp/room-enet-bridge/Release/skyla_enet_bridge.dll').Path
$env:SKYLA_ENET_TEST_NODE_MODULE = '<旧服务 node_modules/sky-enet 的绝对路径>'
go test -count=1 -v ./internal/transport
~~~

DLL 使用相同源码静态链接 ENet，Go 端通过版本化 ABI 绑定。实际客户端 JoinGame 验证需先用隔离测试账号进入场景，确认公告 UDP 地址空闲，并把该本机 IPv4 公告地址设置为 SKYLA_ENET_TEST_REAL_ADDR。验证仅记录协议元数据，收到 JoinGame 后关闭监听。

## 启动已认证的 Windows 房间

API 配置需要设置 room_admission_key 和 room_admission_bearer。前者使用独立的 32 字节随机值并以 64 位十六进制字符串配置，后者使用独立的至少 32 字节随机凭据。现有未配置认证的 API 会继续返回旧发现响应。不要把实际凭据写入提交。

room-server 的 SKYLA_ROOM_ADMISSION_BEARER 使用与 API 相同的服务凭据。API 地址使用 HTTPS 或 HTTP 回环地址，公告值须与 API 的 room_host 和 room_port 相同。支持已验证签名偏移的 284/542 客户端。

~~~powershell
go run ./cmd/server -config config/realtime.json -enet-bridge ../../tmp/room-enet-bridge/Release/skyla_enet_bridge.dll -admission-api http://127.0.0.1:18443 -announce 127.0.0.1:25565
~~~
