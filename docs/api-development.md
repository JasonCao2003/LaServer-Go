# 游戏 API 迁移进度与运行

## 当前范围

本阶段已有可运行的 game-api，使用 Go net/http 和 modernc.org/sqlite。不是完整旧服务替代品：静态扫描找到 business 下 201 条 add_url_rule 注册；当前实现其中 16 个路径，其余 185 个明确返回 501。扫描不包含装饰器路由、动态注册、Socket.IO 或 MQTT/WebSocket 协议，不能把这份数量视为整个旧服务的接口总数。

| 业务 | 已实现路径 |
| --- | --- |
| 账号创建 | /account/create、/account/auth/create |
| 登录 | /account/login、/account/auth/login |
| 存档 | /account/userdata、/account/sync_user_data |
| 货币、排名 | /account/get_currency、/account/get_rank |
| 房间发现 | /account/find_previous_or_empty |
| 神坛检查点 | /account/get_checkpoints、/account/set_checkpoint |
| 启动配置 | /account/get_latest_build_version、/account/device_capabilities/lookup、/account/get_vars |
| 公告与支持 | /account/get_motd、/account/support/has_reply |

所有游戏路径使用 POST，支持 Content-Type: text/plain 的 JSON，最大请求体 2 MiB；GET /healthz 为健康检查。JSON 错误不返回原始请求、会话或数据库细节。已注册旧路径未实现返回 501，未知路径返回 404。不能因为登录接口返回 200 就认为真实客户端已能进游戏：resource_list 的其余资源接口仍需迁移，WebSocket 和实时服务未实现。

## 环境与启动

Go 工具链已安装在 Ubuntu WSL 的 `~/.local/opt/go1.27.4/`，压缩包缓存位于 `~/.cache/laserver-go/`。项目 Go 最低版本为 1.26.0，驱动锁定 modernc.org/sqlite v1.62.0。编译和模块缓存保留在 WSL Linux 文件系统，源代码沿用 Windows 项目目录，没有复制工作区或修改系统工具链。

以下在项目根目录、Ubuntu WSL 执行。只为新开发实例操作，不指向旧数据库：

```sh
export PATH="$HOME/.local/opt/go1.27.4/bin:$PATH"
cp config/game-api.example.json config/game-api.json
# 修改 room_host 为客户端可达 IPv4，并按需设置 TLS 和自动授权。
go run ./cmd/game-api migrate --config config/game-api.json
go run ./cmd/game-api --config config/game-api.json
```

相对数据库和证书路径相对于配置文件目录解析。migrate 显式创建开发数据库目录，仅初始化空数据库，拒绝覆盖非空数据库。正常启动只检查 schema，不自动建表。重跑 migrate 会失败，正常重启直接使用启动命令。当前 schema 只有首批功能所需表，不能直接导入、复制或替换旧实例数据；旧 instance 和 legacy 数据必须分别迁移并验收。

示例仅监听 127.0.0.1，默认不自动授权、不自动补建丢失账号，未授权登录返回 406。在隔离开发实例中可将 auto_authorize 设为 true 创建可登录测试账号；已有账号不会因此被批量授权。TLS 证书和私钥必须成对配置；旧客户端 TLS 互通尚未验证。API 独立启动，不检查实时服务是否在线，room_host/room_port 只是下发配置，不代表 ENet 已实现或可达。

## 已知协议差异与边界

- 登录沿用旧服务可仅凭 user ID 换取会话的行为，尚未实现可验证的设备签名或恢复凭据认证。该行为不是安全认证，当前版本仅供隔离本地开发，不应据此开放公网。
- userdata 与 sync_user_data 均要求当前有效会话，并校验授权和封禁；旧 userdata 无严格校验、sync 仅警告会话不匹配的路径没有保留。
- 默认关闭丢失账号补建，需要显式设置 recreate_missing_account；尚未迁移旧面板的自动授权开始时间、全解锁和账号默认数据配置，初始货币改为显式 default_currency 配置。
- 季卡每日季蜡按 America/Los_Angeles 日期在 SQLite 事务中与领取标记一起提交，不再依赖旧旁路 JSON；重复和并发请求同日只发一次。旧每日标记尚未导入，不可直接切换真实账号。
- 货币表新增 user_id 唯一约束，避免新库同账号重复记录；旧重复行需在迁移时单独处理。
- 检查点采用旧 checkpoint_defs 的首条关卡顺序，保留数值 ID、更新已有关卡而不重复插入，并维护最远标记。
- get_vars 来自旧源码公开协议常量，不代表其中开关指向的功能已经实现。MOTD 使用无外部链接的开发默认公告，尚未接入旧公告配置。设备描述写入等附带行为尚未完全迁移。
- 登录 resource_list 和 websocket_url 保留旧协议字段，清单中其余资源及 websocket 实现仍待迁移；此处不保证完整客户端登录链路。
- 错误响应的详细文案、auth/login 特殊错误形状和部分表单兼容尚未逐分支对齐；目前测试验证核心状态、数据副作用及首批响应字段。

## 验证

用户已授权安装工具链和运行本地测试，后续可使用 `sh scripts/verify/game-api.sh`。日志集中在 logs/。测试使用临时数据库，不接触旧实例。单元/集成测试和交叉编译不等于 Windows 实机运行、TLS 互通或实际客户端验收。

## 后续迁移

依赖启动资源完整性继续实现装扮、基础定义、解锁、好友与邀请，然后完成购买/奖励事务、任务和社交内容接口。每个接口需依据旧逻辑加入数据隔离、错误和并发断言，再更新清单状态。完整迁移未完成，不把 501 路由计为成功接口。
