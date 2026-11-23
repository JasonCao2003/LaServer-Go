# 实际 APK 兼容矩阵

基线日期：2026-11-11。版本含义依据 Package 对应归档文档，摘要已对当前交付文件重新计算。0.3.11 已安装到 ADB 连接的 arm64 设备并连入开发机局域网；按用户要求，后续接口迁移期间暂停客户端和登录验证。

| 项目 | 0.1.8 适配包 | 0.3.11 适配包 |
| --- | --- | --- |
| 当前文件 | Package/0108/Sky 0.1.8_4图_fix.apk | Package/0311/Sky 0.3.11_7图_LaServer_返程与HubReveal修复版.apk |
| 资源版本（归档证据） | iOS 0.1.8 移植地图 | 0.3.11 七图 |
| 引擎/上报（归档证据） | Android 0.13.4 / 0.13.4.170388 | 上报 0.13.4.170388；引擎 Build ID 待核对 |
| 包名（归档证据） | com.tgc.sky.androidl18 | com.tgc.sky.android311 |
| SHA-256（当前文件实测） | 48f6eaae7d93c0c7424d82b95e3c0bc1683ea329abcac3e43e39c23a75b21cdb | 6827a22f07e398eb5c81bb632a9391cdbbe0d236a53e3177dc8c8378e16931b4 |
| HTTPS 地址（归档证据） | https://192.168.123.136:18443 | https://192.168.123.136:18443 |
| ABI / 设备 / 安装 | 待验证 | arm64 设备；安装成功 |
| Go 服务 TLS / 创建账号 / 登录 | 未执行 | 曾执行早期本地排查；接口迁移完成后再回归 |
| 资源拉取 / 存档读写 / 重启恢复 | 未执行 | 未执行 |
| 房间发现 / UDP 进房 / 双端同步 / 过图 | 未执行 | 未执行 |

归档来源：旧工作区 Package/0108/0.1.8.md、Package/0311/0.3.11.md。这些历史说明不证明 Go 服务已兼容。

0.3.11 包曾成功访问本机测试 API；旧 Go 路由尚未迁移完整时，登录链路返回 `ServerError`，最后观察到的 501 路由为 `/account/get_forge_rates`。该接口现已接入，但尚未重新验证。测试 API 已停止，测试账号和应用数据保留。

## 2026-11-23：MuMu 0.1.8 与 Go API

- 测试源包为 `Package/0108/Sky-0.1.8-hidden-menu-schedule-hook-fix-20261119-aligned-debugSigned.apk`。仅重建 `classes.dex`，把旧 API 地址 `192.168.137.1:8443` 改为当前开发机 `10.17.43.177:18443`；签名及 ZIP 对齐验证通过。最终测试包为 `Package/0108/Sky-0.1.8-go-api-10.17.43.177-rebuilt-aligned-debugSigned.apk`，SHA-256 `4928d3a1d30d54719976a2708cd53b711d0c523ed643579d40d4042afe75a0f7`。这是网络地址测试包，原始交付包未覆盖。
- MuMu ADB `127.0.0.1:16384`，ABI `x86_64,arm64-v8a,x86`；保留数据覆盖安装成功。Go API 使用独立 `services/api-server/data/mumu-0108-20261123/api.db`，TLS 监听 `18443`。模拟器 TCP 连接及 `/healthz` 成功，真实客户端启动、账号创建、存档和多项资源请求到达 Go API，日志为 `logs/mumu-0108-api-20261123.log`。
- 首次登录资源同步在 `/account/get_unlocks` 后触发客户端原生断言 `Json.cpp (97): Unrecognized JSON escape character \\u` 并崩溃重启。Go 默认 JSON 编码会把 `&` 等字符写成 `\\u0026`；旧配置全量解锁模板包含 `reveal friend & family finish`。修复 Go HTTP 响应编码时关闭 HTML 转义，新增回归用例先失败后通过；`go test ./...` 通过。修复后直接请求 `/account/get_unlocks` 返回 200、1466 项，响应中无 `\\uXXXX`。
- 用户确认已进入游戏。Go 日志证实 `auth/login`、存档读取、完整登录资源序列、`get_unlocks`、`get_shop`、`get_spirit_shops`、收藏品、魔法、消息、解锁状态、房间发现及 `set_checkpoint` 已到达，客户端进程在观察时保持运行。首次调用 `get_account_world_quests`、`get_invites`、`find_previous_or_empty` 和 `add_unlock` 偶有 500，随后同路径重试为 200；原因尚未定位，不能将该链路标为无错误。现有日志把非业务错误类型固定写为 `internal`，不足以判断是否 SQLite 锁或请求取消。`find_previous_or_empty` 返回 200 仅说明房间地址下发，不代表 UDP 已连通。
- 首次直接修改 DEX 字符串的候选包因字符串表排序错误在应用启动前崩溃，已改用 apktool 重建。该失败包及中间包是候选清理项，不属于交付包。上述测试未验证房间 UDP、存档写入或地图加载。

## 设备验收记录要求

- 每次测试记录 APK 摘要、实际版本、上报版本、包名、ABI、设备和时间；设备标识只留本地脱敏报告。
- 安装前核对目标包是否存在，不自动卸载或清理应用数据。用户已授权使用 ADB 从 Package 安装本轮测试包。
- 使用独立 Go 数据库和测试账号，确认测试端口空闲、地址与证书配置正确；不接入旧服务真实实例。
- 按冷启动 → TLS → 认证 → resource_list 拉取 → 存档保存/重启 → 房间发现记录首个失败点；room-server 未实现，不将房间地址下发写为 UDP 进房通过。
- 不截图确认状态；需要界面信息时先询问用户，日志放 logs/，报告放 json/。
