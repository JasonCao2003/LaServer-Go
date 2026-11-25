# 独立管理面板

使用 Go 1.26 或更新版本。从本目录复制示例配置并填写上游地址。通过本地环境分别提供 `LASERVER_PANEL_PASSWORD` 和 `LASERVER_PANEL_API_TOKEN`。好友管理所用令牌须与 API 的 `admin_token` 匹配。

```sh
cp -n config/admin-panel.example.json config/admin-panel.json
go run ./cmd/admin-panel -config config/admin-panel.json
```

默认访问 `http://127.0.0.1:18080/login`。浏览器地址须使用配置中相同的回环 IP 和端口。面板独立启动，登录后可查看 API 健康、房间健康与房间就绪状态。会话有效期为 8 小时，重启面板后需重新登录。从概览进入好友管理，按用户 ID 查询，支持增加、移除、恢复和清空。清空时重新输入用户 ID。遇到写入超时，请先重新查询确认结果。

## IP 封禁

登录后从概览打开 IP 封禁页，可维护 IPv4、IPv6 地址和 CIDR 网段。规则保存在 API 实例目录的 banned_ips.json，修改立即生效并跨重启保留。游戏请求命中时返回 HTTP 405。维护规则需要配置 API 管理令牌。

默认按直接连接地址判断。若游戏 API 在反向代理后部署，可将明确受信的代理地址或网段以 CIDR 格式加入 API 的 ip_ban_trusted_proxies。应由受信代理追加实际来源到 X-Forwarded-For。该配置仅影响 IP 封禁判断。健康检查、后台和内部接口保留原有访问规则。

在本目录运行 `go test -race ./...`、`go vet ./...` 和 `go build ./...`。

从仓库根目录可运行 `sh scripts/verify/admin-panel.sh`，日志写到根目录 `logs/`。联调使用临时 SQLite 和本机回环端口。从仓库根目录运行以下命令：

```sh
mkdir -p tmp
(cd services/api-server && go build -o ../../tmp/admin-task4-api ./cmd/server)
(cd services/admin-panel && go build -o ../../tmp/admin-task4-panel ./cmd/admin-panel)
python3 scripts/admin-panel/smoke.py tmp/admin-task4-api tmp/admin-task4-panel
```

有 Node.js 和 Google Chrome 时，可在最后一条命令追加 `--browser`，运行键盘登录、好友查询、清空确认与 360px 窄屏检查。`CHROME_BINARY` 可指定兼容 Chromium 的可执行路径。

### 运营配置更新接收

在面板进程设置 `LASERVER_UPDATE_PUBLIC_KEY` 为可信 Ed25519 发布公钥（32 字节，标准 Base64）。`LASERVER_UPDATE_LISTEN` 默认取 `127.0.0.1:9527`。登录面板后在「更新接收」启动监听。发送方 POST 到该监听地址的 `/updates`，在 `X-Update-Signature` 中传入原始请求体签名的标准 Base64。请求体示例如下，revision 取自目标配置的当前读取结果。

```json
{"section":"events","revision":"<当前配置的 SHA-256>","value":{}}
```

接收端以内容摘要识别重复批次，在面板配置文件旁的 `update-inbox/` 暂存。管理员预览后可应用或拒绝，版本冲突保留待审状态。此协议支持运营页列出的六类 API 配置。

### HTTP 与 UDP 收发查看

在房间进程中设置与面板一致的 `LASERVER_PANEL_API_TOKEN`，面板通过配置的房间管理地址读取记录。两个服务各在内存保留最近 128 条记录，重启后重新累计。HTTP 显示请求序号、路径、状态码、耗时、字节数及允许公开的 JSON 字段。UDP 按 peer、通道、方向、类型编号和长度记录实际收发，广播产生多条发送记录。登录后可按关键词筛选并刷新。
