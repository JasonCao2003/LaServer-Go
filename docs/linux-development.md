# Linux 开发环境与适配验证

2026-11-24，Fedora 45 x86_64，Go 1.27.4，GCC 16.2.1，CGO_ENABLED=1。

## 已完成的适配

- room-server 新增 Linux cgo/libdl 加载器，沿用原生桥接 ABI v1；C 侧持有库句柄和 ENet host，不保留 Go 指针。
- Windows/Linux 共用真实启动入口，Linux 支持 SIGTERM；Windows DLL 传输保留。
- 根检查脚本使用 PATH 中的 Go，不再优先引用旧 WSL 安装目录。默认编译缓存放根 tmp/，日志放 logs/，房间可执行文件放服务 dist/。
- 增加原生库构建脚本：`sh scripts/room-server/build-native.sh <sky-enet 0.1.5 ENet 源码绝对路径>`，产物位于 services/room-server/tmp/enet-bridge/。

## 还需用户准备的依赖

本轮未安装任何系统包、Go module 或 npm 依赖。两个检查脚本均固定 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，避免验证过程自动下载依赖或工具链；API 依赖需要在检查前单独准备。

首次适配时 API 依赖缓存为空，离线检查在 `modernc.org/sqlite v1.62.0` 处失败。由用户自行进入 module 下载锁定依赖：

```sh
cd /home/jason/Jason-WorkSapce/LaServer-Go/services/api-server
go mod download
```

不需要 SQLite 数据库服务，也不需要为 API 安装 SQLite C 开发包。

房间原生库需要 sky-enet 0.1.5 对应的 ENet 源码。当前项目及旧服务目录未找到该源码；须由用户提供原版本源码目录，不能用普通系统 ENet 代替。只有需要运行 Node 互通测试时才需要相应可用的 Node sky-enet 模块。

## API 配置

进入 API 服务目录，从 config/api-server.example.json 复制到未跟踪的 config/api-server.json。保持监听 127.0.0.1:8443、数据库 ../data/api-server.db，可将 game_config_dir 设为 ../../../../LaServer/services/game-api/config。相对路径以配置文件目录解析。该只读目录的 map.json 和 collectibles.json 已确认存在。

安装依赖后，从根执行 `sh scripts/verify/api-server.sh`；然后在 API module 内执行迁移及启动命令。此次没有复制本地配置、初始化数据库或修改旧服务。

## 验证证据与边界

从根运行的检查：

```sh
GOPROXY=off GOSUMDB=off sh scripts/verify/room-server.sh
```

上面的临时环境变量语法适用于 Bash；Fish 可写为 `env GOPROXY=off GOSUMDB=off sh scripts/verify/room-server.sh`。本轮实际检查还将 GOMODCACHE 指向根 tmp/go-mod，以避免写入用户目录。

| 检查 | 结果 | 日志 |
| --- | --- | --- |
| 房间全量 Go 测试 | 通过；真实原生集成两项跳过 | logs/room-server-tests.log |
| 房间 race | 通过 | logs/room-linux-race.log |
| 房间 vet | 通过 | logs/room-server-vet.log |
| Linux room-server 构建 | 通过 | logs/room-server-build-linux.log |
| Windows amd64 交叉构建 | 通过；未运行 Windows 程序 | logs/room-server-build-windows.log |
| API 离线检查 | 修正 v18 重复迁移的测试预期后，独立启动、健康检查、SIGTERM 退出和端口释放均通过；API 全量测试、vet 及 Linux/Windows 构建通过 | logs/api-startup-after.log、logs/api-server-tests.jsonl |
| 真实 SkyENet .so 构建 / UDP / APK 入房 | 源码缺失，未验证 | 无 |

Linux 动态加载测试使用编译出的测试 C 库，覆盖缺库、缺符号、ABI 不匹配、打开失败、报文通道/可靠标记、报文数据所有权、超时范围、异常长度/事件、发送失败以及重复关闭。该测试库不实现 ENet，也不作为运行时库交付。

首次沙箱内全量测试受禁止绑定 socket 的限制，app 和 session 测试失败；在允许回环监听的执行环境重跑后通过。初次日志保留在 logs/room-linux-tests.log。最终检查以 room-server-tests.log 为准。
