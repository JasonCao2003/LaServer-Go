# 第二轮：重构前的行为基线

2026-11-24，基于 f4bebf6 和当时工作区，Fedora 45、linux/amd64、Go 1.27.4、cgo。两个服务独立执行。测试使用临时 SQLite、随机回环端口和测试凭据。生产代码、数据库 schema 及 UDP 协议均无变更。

| 服务 | 原有测试 | 最终测试 | 最终 race | vet、Linux 构建、Windows amd64 交叉构建 |
| --- | --- | --- | --- | --- |
| api-server | 220 通过 | 221 通过 | 221 通过 | 通过 |
| room-server | 76 通过，2 跳过 | 82 通过，2 跳过 | 82 通过，2 跳过 | 通过 |

计数取 go test -json 中含 Test 字段的最终 pass/skip 事件，包含子测试。常规和 race 执行均使用 -count=1。结果日志位于根 logs/phase2-{api-server,room-server}-{before,final,race-before,race-final}.jsonl；vet 输出在对应的 vet-final.log。构建按退出状态核对。Windows 只交叉编译。race 只覆盖实际执行到的 Go 路径。

## 本次固定的行为

- API 的 Gin 路由清单、HTTP 方法及错误、账号鉴权、资产并发与故障回滚、SQLite 重开和 SIGTERM 由已有用例保护。新增 TestWingConversionRollbackAndRetry 使用临时 SQLite 触发器让声望更新失败，在 HTTP 500 后检查光翼未标记兑换，恢复后两次请求的兑换数依次为 1 和 0，最终声望为 1。
- UDP 继续覆盖 284/542、时钟与游戏消息、快照校验、旧世代 ACK、关卡切换和权威转移。新增 TestLateAdmissionCannotAffectReusedPeer 与 TestVerificationBackpressureAndShutdown，固定认证晚到、同 peer 重连、八槽耗尽和结果队列已满时取消的行为。
- TestFailedSnapshotDeliveryKeepsPeersAndSourceAck 固定当前可观察行为：源端收到 ACK，向接收端发送失败未导致断连，随后 ACK 和同关卡消息仍可处理。这条用例记录当前语义；改变重试或断连策略须单独评估。
- TestAdmissionAgainstRealAPI 启动独立编译的真实 API 子进程，用临时账号取得发现接口凭据，通过房间服务的真实 HTTP verifier 校验账号、到期时间，并确认错误签名被拒绝。过期响应和调用方 deadline 在专门的 HTTP 测试服务器上验证。
- TestNativeOpenAndPoll 与 TestNativeRoundTripWithSkyENet 本次缺少指定的真实桥接库及 Node SkyENet 对端，按条件跳过。真实 APK、双端同步、自然过图及原生 C 内存安全不在此次结论内。

## 复跑

以下从仓库根目录执行，适用于 Bash。需要 Go 模块缓存已准备、cgo、C 编译器和回环监听权限。

```bash
set -e
repo_root="$PWD"
mkdir -p tmp/go-build tmp/go-tmp logs
export GOPROXY=off GOTOOLCHAIN=local GIN_MODE=test
export GOCACHE="$repo_root/tmp/go-build" TMPDIR="$repo_root/tmp/go-tmp"
export SKYLA_TEST_API_BINARY="$repo_root/tmp/phase2-api"
(cd services/api-server && go build -o "$SKYLA_TEST_API_BINARY" ./cmd/server)
for svc in api-server room-server; do
  (cd "services/$svc"
   go test -count=1 -json ./... > "$repo_root/logs/phase2-$svc-final.jsonl"
   go vet ./... > "$repo_root/logs/phase2-$svc-vet-final.log" 2>&1
   go test -race -count=1 -json ./... > "$repo_root/logs/phase2-$svc-race-final.jsonl"
   go build ./...
   GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...)
done
```

SKYLA_TEST_API_BINARY 用于显式启用跨服务用例；未设置时房间服务仍可独立开发，该用例会显示跳过。下一轮以相同环境重新运行并逐项对照；测试失败需区分外部条件、兼容性变化和既有行为。
