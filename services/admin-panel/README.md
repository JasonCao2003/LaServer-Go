# 独立管理面板

使用 Go 1.26 或更新版本。从本目录复制示例配置并填写上游地址。通过本地环境分别提供 `LASERVER_PANEL_PASSWORD` 和 `LASERVER_PANEL_API_TOKEN`。好友管理所用令牌须与 API 的 `admin_token` 匹配。

```sh
cp -n config/admin-panel.example.json config/admin-panel.json
go run ./cmd/admin-panel -config config/admin-panel.json
```

默认访问 `http://127.0.0.1:18080/login`。浏览器地址须使用配置中相同的回环 IP 和端口。面板独立启动，登录后可查看 API 健康、房间健康与房间就绪状态。会话有效期为 8 小时，重启面板后需重新登录。从概览进入好友管理，按用户 ID 查询，支持增加、移除、恢复和清空。清空时重新输入用户 ID。遇到写入超时，请先重新查询确认结果。

在本目录运行 `go test -race ./...`、`go vet ./...` 和 `go build ./...`。

从仓库根目录可运行 `sh scripts/verify/admin-panel.sh`，日志写到根目录 `logs/`。联调使用临时 SQLite 和本机回环端口。从仓库根目录运行以下命令：

```sh
mkdir -p tmp
(cd services/api-server && go build -o ../../tmp/admin-task4-api ./cmd/server)
(cd services/admin-panel && go build -o ../../tmp/admin-task4-panel ./cmd/admin-panel)
python3 scripts/admin-panel/smoke.py tmp/admin-task4-api tmp/admin-task4-panel
```

有 Node.js 和 Google Chrome 时，可在最后一条命令追加 `--browser`，运行键盘登录、好友查询、清空确认与 360px 窄屏检查。`CHROME_BINARY` 可指定兼容 Chromium 的可执行路径。
