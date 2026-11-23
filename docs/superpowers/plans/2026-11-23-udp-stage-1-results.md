# UDP 第一阶段验证记录

日期 2026-11-23 · 部分完成

## 源码盘点

[覆盖清单](../../../json/room-server/stage-1-inventory.json)以旧 LaServer 当前提交 f93654cb0d953f6aee8b68c2260af426fca0aa54 的实时服务为基线。清单包括 16 个顶层包声明、7 个客户端调度分支、13 个包 codec、18 种 GameMsg、版本引擎和 74 条静态构建扫描记录。扫描条目仅提供指纹线索。协议声明、分支存在、已在基线运行和真实客户端通过分别记录。

旧严格 JoinGame codec 读取 UUID、joinToken 和 joinSignature。旧 Go API 发现响应由公告地址与固定字符串产生签名。两个链路之间还缺少真实客户端字节证据，不能据此定稿认证。

## Windows 回环对照

使用项目已有 sky-enet 0.1.5（MIT）与 Windows Node.js v24.22.1，在 127.0.0.1:25577 上启用 checksum，使用同库客户端执行两次独立连接。每次均通过 channel 0 的 8 字节与 4096 字节可靠双向收发、channel 1 的 9 字节非可靠双向收发，以及携带 data=42 的断开通知。4096 字节测试观察到重组后的数据完整，未捕获分片本身。未注入丢包或乱序，重传行为未执行。

首次回环探针轮询了已由库内部消费的接收事件，等待回包超时。改用客户端 receive 事件后双向通过。原生对照仅证明同库回环。WSL 未安装 Linux 发行版，当前未验证 Linux 构建或 Go 候选。

## 旧协议回归基线

PacketCodec.test.js 与 GameMessageService.test.js 共 9 项通过。sky0134-regression.mjs 有 1 项失败，断言为 0134 build scoped (other build rejected)。此失败来自只读旧实现的回归基线。日志见 stage-1-legacy-codec-20261123.log 和 stage-1-legacy-sky0134-20261123.log。

## 实际 APK

| APK | 当前证据 |
| --- | --- |
| 0.1.8 go-api rebuilt | MuMu 安装包与归档 APK 的 SHA-256 相同，4928D3A1D30D54719976A2708CD53B711D0C523ED643579D40D4042AFE75A0F7。包处理 ABI 为 arm64-v8a。重启后的客户端日志上报 Version 0.13.4.170388 和 NetVersion [12274, 21082, 19616, 0, 0, 11920]。该证据来自客户端日志，JoinGame 布局需以 UDP 样本确认 |
| 0.3.11 hidden-menu-hotspot | 归档 APK SHA-256 为 024C1FE118889F74BB6084B91E63A1E7BD7C7CF564190CEF41F91D282BF58A9F。历史 0.3.11 日志含 0.13.4.170388。当前未在测试设备上运行该 APK |

API 监听 TCP 18443，现有 MuMu 配置公告 10.17.43.177:25575，本机 WLAN 持有该 IPv4。借用旧 RoomServer 在该 UDP 地址限时监听，并关闭管理 HTTP 与原始包日志。第一次监听窗口内恢复启动了 0.1.8。第二次额外监听 90 秒，两次均未收到 connect。无法据此区分客户端未发起发现、未发送 UDP 或网络路径未送达。

日志位于本项目 logs/room-server。Windows 回环结果见 stage-1-loopback-events-20261123.log，两次实机窗口见 stage-1-mumu-0108-events-20261123.log 和 stage-1-mumu-0108-manual-20261123.log。监听由探针定时回收。

## 选型状态

sky-enet 作为已通过回环的参考基线。客户端构建日志不能决定客户端线上 UDP 格式或 Go 侧实现。0.1.8 已验证握手、收发及服务重启后的重连。0.3.11 仍需实际 APK 验证，再结合候选的 Windows/Linux 构建、运行依赖和线上兼容性选型。保留传输选型待定。

## 0.1.8 进场后复测

2026-11-23 19:45（北京时间），用户确认已进场后重新监听 10.17.43.177:25575。约 0.4 秒后收到真实客户端 ENet connect。channel 0 收到 1931 字节 JoinGame，旧服务识别为 sky0134 / mqtt-284-542，向客户端发送 41 字节 EnterGame。随后解析了 LevelUpdate、GameMsg 和 channel 1 的 NetTimePing，并观察到服务端持续发送 UDP 响应。日志为 logs/room-server/stage-1-mumu-0108-ingame-20261123.log。

这次实测确立了该 APK 的 ENet 线上兼容基线。测试使用旧房间服务，其 JoinGame 解析和响应不构成 Go API 入房认证。首次收到 LevelUpdate 也不构成自然过图验收。本窗口未采集原始载荷，不能生成黄金字节样本。

19:46:31 重启同地址的监听，19:46:55 客户端重新连接并再次发送 1931 字节 JoinGame。旧服务再次发出 EnterGame，随后解析了 LevelUpdate、GameMsg 和 NetTimePing。期间未重启客户端。此结果验证服务重启后的恢复，未测客户端主动断开回调。日志为 logs/room-server/stage-1-mumu-0108-reconnect-20261123.log。第二次临时监听于 19:47:06 退出。
