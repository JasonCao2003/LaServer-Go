# API 登录认证与旧客户端兼容

## 证据边界

旧 Python `auth_login.py` 要求 `user`，从 JSON（包括 UTF-8 BOM）取值，源码有表单回退分支但标准表单能否进入该分支尚未实测；登录时轮换 `session` 和 `recovery_token`，但没有核验请求凭据。旧 `login.py` 接受旧客户端的 `device`／`key`／`language`，通过 `device` 反查账号；`key` 是创建时返回的全零值，未被核验。两条旧登录路径也允许仅凭 `user` 换取会话。`auth_create.py` 要求签名字段存在，但没有验证签名；它返回 `authinfo.recovery_token`。旧 `create.py` 为低版本客户端补齐缺少的字段，返回的 `key` 为全零。

已有 0.1.8、0.3.11 隔离环境日志证实两版均请求 `/account/auth/login`，0.3.11 还请求 `/account/auth/create`，并曾收到 200。日志没有保存登录请求体、响应体及凭据字段，**不能据此断言任一 APK 会提交 `recovery_token`**。旧服务对 0.1.8／0.3.15 的静态逆向笔记记录了 legacy `device`／`key`／`language` 形状，但不能替代 0.3.11 的实际请求记录。原版客户端是否能使用默认凭据模式，仍待隔离抓包或客户端侧字段记录确认。

## 四条路径

| 路径 | 旧服务输入与响应 | Go `credential`（默认） | Go `legacy_dev` |
| --- | --- | --- | --- |
| `/account/auth/create` | JSON，必需 `type`、`device_name`、`device_token`、`production`、`tos_version`、`device_key`、`sig_ts`、`sig`、`hashes`；返回 `authinfo`、`session`、`resource_list` | 同形状，返回随机 `recovery_token`；签名字段仍只检查存在，未验证签名 | 同左 |
| `/account/auth/login` | JSON/BOM，源码有表单回退但未实测；必需 `user`；不存在返回 200 `not_found`，未授权／封禁返回 406 | 必需 `user` 与在凭据模式签发的当前 `recovery_token`；错误凭据返回 401；成功原子轮换凭据及会话，旧会话失效 | 沿用仅凭 `user` 登录、200 `not_found`、406；响应 `authinfo.device` 使用长度至少 32 的请求值，否则生成 UUID；不改写库中的设备标识 |
| `/account/create` | 老版本只需 `device_name`／`language`；返回 `user`、`device`、全零 `key`、`session` | 403，避免创建无法用安全凭据再次登录的账号 | 保留旧创建与响应字段 |
| `/account/login` | 旧客户端以 `device`、全零 `key`、`language` 登录；也接受 `user`；成功轮换会话，按请求更新设备 | 403 | 保留旧行为；未知账号 404，未授权／封禁 406 |

Go 的四条路径接受 JSON，包括 `text/plain` 头与 UTF-8 BOM；仅 `/account/auth/login` 支持真正的 `application/x-www-form-urlencoded` 表单。多值表单字段拒绝为 400。格式与旧服务的全部错误文案尚未逐项对齐。

## 配置和部署

`login_mode` 省略时按 `credential` 处理。`legacy_dev` 必须显式配置，且 `listen` 必须是具体回环或私有 IP；`0.0.0.0`、`::`、域名和公网 IP 会在启动校验时拒绝。`recreate_missing_account` 仅允许在 `legacy_dev` 下开启。私有 IP 仍可被同一局域网访问，测试时需使用隔离网络及防火墙限制。不要将 `legacy_dev` 称为安全认证，也不要把 `credential` 模式的旧客户端兼容性视为已验证。非回环部署还需 TLS 或可信的 TLS 终止代理。

schema v18 用 `credential_mode_issued` 标记会话和恢复令牌的签发模式。v17 升级时现有账号默认为 0；`legacy_dev` 下创建或登录也写 0，即使原令牌字符串未变。切换到 `credential` 后，这些账号的旧会话和令牌均被拒绝，不能直接用旧账号登录；需要新账号，或后续建立经过验证的凭据重发流程。`recovery_token` 当前以明文存于 SQLite；数据库文件、备份和日志权限应限制，令牌不应写入请求日志。

## 回归覆盖与待验收

本地 Go 测试覆盖：缺失／错误凭据拒绝、跨账号令牌拒绝、失败时旧会话不变、成功轮换后旧会话失效、旧令牌重试失败、表单与 BOM JSON、旧 `auth/login` 设备响应和持久字段、旧登录的隔离模式、同库模式切换、v17 升级，以及不安全监听地址拒绝。实际 APK 请求体字段、`recovery_token` 持久化与二次登录行为仍未验证；待客户端测试恢复后，分别记录 0.1.8／0.3.11 的字段名称、Content-Type、状态与重启登录结果，记录时不得保存令牌值。
