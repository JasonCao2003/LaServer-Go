# 实时联机服务

独立 Go module，使用 internal/transport、protocol、session、room、config 和 app 目录。Windows amd64 与 Linux cgo 入口接入原生 SkyENet、Go API 认证、初始名册、离房通知和时钟响应。`284/542` 房间支持同关卡 RPC、生物事件、亲和事件和音乐消息，转发玩家与关卡状态快照并处理 ACK、关卡权威选举和目标关卡名册刷新。示例见 [realtime.example.json](config/realtime.example.json)，其 UDP 地址绑定回环。真实双端同步和自然过图、多房间流转及其他协议族的玩法仍待验收或开发。Go 代码使用标准库，ENet 桥接单独构建。

此服务独立构建和部署，不导入 api-server 的 internal 包，不直接读写其 SQLite。ENet/UDP 兼容验证和实现按项目开发计划推进。


## Linux 构建与运行

需要 Go ≥ 1.26、C 编译器、CMake ≥ 3.20 和 **sky-enet 0.1.5 对应的 ENet 源码**。Go module 没有第三方依赖。Linux 通过 cgo/libdl 加载 ABI v1 `.so`；不能设置 `CGO_ENABLED=0`。普通系统 ENet 不保证与该分支的 API 或线协议兼容。

从仓库根目录构建原生库（脚本不下载或安装依赖）：

```sh
sh scripts/room-server/build-native.sh /absolute/path/to/sky-enet-0.1.5/enet
sh scripts/verify/room-server.sh
```

进入 `services/room-server` 后，复制示例配置并按下面 Windows 段落的规则配置 API 入场密钥、服务凭据和公告地址。Fish 设置服务凭据使用 `set -gx SKYLA_ROOM_ADMISSION_BEARER <本地凭据>`，不要将真实值写入版本库。

```sh
cp config/realtime.example.json config/realtime.json
go run ./cmd/server -config config/realtime.json -enet-bridge tmp/enet-bridge/libskyla_enet_bridge.so -admission-api http://127.0.0.1:8443 -announce 127.0.0.1:25565
```

Linux 入口响应 SIGINT 和 SIGTERM，复用已有的退出及资源回收流程。管理端口默认为回环 `127.0.0.1:8080`，`GET /healthz` 和 `GET /readyz` 成功返回 204。

验证范围：Linux/amd64 全量 Go 测试、race、vet、Linux 构建和 Windows 交叉构建已通过；动态加载测试使用独立 C ABI fixture，不验证真实 ENet。当前机器缺少 sky-enet 源码，因此真实 `.so` 构建、实际 UDP 互通和 APK 入房尚未验证。

真实桥接可用后，可在目标 module 内设置 `SKYLA_ENET_BRIDGE_TEST_LIBRARY` 为 `.so` 绝对路径并运行 `go test -count=1 -v ./internal/transport`；若同时提供 `SKYLA_ENET_TEST_NODE_MODULE`，会运行 Node SkyENet 往返测试。未设置时相应用例明确跳过。Windows 旧变量 `SKYLA_ENET_BRIDGE_TEST_DLL` 仍兼容。

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
go run ./cmd/server -config config/realtime.example.json -enet-bridge ../../tmp/room-enet-bridge/Release/skyla_enet_bridge.dll -admission-api http://127.0.0.1:18443 -announce 127.0.0.1:25565
~~~

需要自定义监听地址时，将 `config/realtime.example.json` 复制为 `config/realtime.json`，修改后把 `-config` 指向该文件。

帧长上限需同时容纳 7500 字节快照及 9 字节包头，以及配置人数所需的完整名册。示例使用 16384 字节。
