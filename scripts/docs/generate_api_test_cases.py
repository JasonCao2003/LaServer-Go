"""Generate the per-route API test case inventory from registered routes."""

from __future__ import annotations

import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
HANDLER = ROOT / "services/api-server/internal/handler/api.go"
SERVICE = ROOT / "services/api-server/internal/service"
LOG = ROOT / "logs/mumu-0108-api-fixed-20261123.log"
OUTPUT = ROOT / "docs/api-test-cases.md"

routes = re.findall(r'a\.routes\["([^"]+)"\] = s\.(\w+)', HANDLER.read_text(encoding="utf-8"))
assert len(routes) == 200, f"registered API routes changed: {len(routes)}"
routes.append(("/healthz", "Healthz"))

functions: dict[str, str] = {}
for source in SERVICE.glob("*.go"):
    content = source.read_text(encoding="utf-8")
    matches = list(re.finditer(r'^func \(a \*Service\) (\w+)\([^\n]*\) \([^\n]*\) \{', content, re.M))
    all_functions = [m.start() for m in re.finditer(r'^func ', content, re.M)]
    for match in matches:
        end = next((position for position in all_functions if position > match.start()), len(content))
        functions[match.group(1)] = content[match.end() : end]

seen: set[str] = set()
if LOG.exists():
    seen = set(re.findall(r'method=POST path=(\S+) status=200', LOG.read_text(encoding="utf-8")))


def category(path: str) -> tuple[str, str]:
    p = path.lower()
    if p.startswith("/admin/"):
        return "管理", "在隔离管理端调用；无游戏内入口"
    if "/stage/" in p or "/reservation/" in p:
        return "舞台/预约", "进入舞台、公寓或预约界面；若该版本无入口，使用隔离接口请求"
    if any(x in p for x in ("/feed", "/recording", "/geonote")):
        return "内容", "打开对应留影、留言或社区界面，执行发布、读取或互动"
    if any(x in p for x in ("/message", "/chat", "/send", "/gift", "/translate")):
        return "消息/赠礼", "使用两个测试账号操作聊天、收件箱或赠礼界面"
    if any(x in p for x in ("friend", "relationship", "follow", "invite", "serendipity", "/star/")):
        return "好友/关系", "使用两个测试账号操作好友星盘、邀请或关系界面"
    if any(x in p for x in ("quest", "season", "pickup", "achievement")):
        return "任务/进度", "打开任务、季节或地图进度界面并执行对应操作"
    if any(x in p for x in ("shop", "purchase", "commerce", "lootbox", "currency", "forge", "redeem", "iap", "candle_wax", "/code/")):
        return "商店/货币", "打开商店或货币界面；写入接口仅用隔离测试账号"
    if any(x in p for x in ("consumable", "buff", "collectible", "unlock", "outfit")):
        return "物品/装扮", "打开魔法、光翼、收藏或装扮界面并执行对应操作"
    if any(x in p for x in ("auth", "login", "create", "userdata", "checkpoint", "find_previous")):
        return "登录/存档", "登录、重启或切图；核对请求与存档读回"
    return "启动资源/状态", "冷启动或打开对应资料界面；核对资源同步请求"


def fields(body: str) -> tuple[str, str]:
    required = set()
    for group in re.findall(r'require\(m,\s*([^)]*)\)', body):
        required.update(re.findall(r'"([a-zA-Z_][a-zA-Z_0-9]*)"', group))
    optional = set(re.findall(r'(?:text|number|integer|flag|boolean|array)\(m,\s*"([a-zA-Z_][a-zA-Z_0-9]*)"', body))
    optional.update(re.findall(r'm\["([a-zA-Z_][a-zA-Z_0-9]*)"\]', body))
    optional -= required
    optional -= {"user", "session"}
    request = "分支校验字段：" + ", ".join(f"`{x}`" for x in sorted(required)) if required else "必需字段待核实"
    if optional:
        request += "；代码涉及：" + ", ".join(f"`{x}`" for x in sorted(optional)[:12])
    response = set()
    for literal in re.findall(r'(?:return\s+(?:Object|object)\{|\b(?:Object|object)\{)([^\n{}]{0,400})', body):
        response.update(re.findall(r'"([a-zA-Z_][a-zA-Z_0-9]*)"\s*:', literal))
    return request, ", ".join(f"`{x}`" for x in sorted(response)[:12]) if response else "返回字段待核实（查函数分支/配置）"


def expectation(path: str, action: str) -> str:
    if path.startswith("/admin/"):
        return "授权调用返回预期数据或结果；未授权拒绝；状态写入后读回一致"
    if any(word in path for word in ("/get", "/lookup", "/list", "/check", "/auth", "/login", "/create")) and not any(word in path for word in ("/create", "/update", "/set", "/claim", "/send", "/delete")):
        return "正常请求返回协议约定的字段和类型；内容与该账号/配置一致；无效会话不得读取私有数据"
    if action in ("登录/存档", "启动资源/状态"):
        return "返回协议约定字段；重启后状态一致；缺字段或无效会话按实现拒绝"
    return "首次操作的状态变化与游戏一致；重复请求不得重复奖励或扣费；重登后读回一致，跨账号隔离"


lines = [
    "# API 逐路由测试用例（201 条）",
    "",
    "来源：`services/api-server/internal/handler/api.go` 的 200 条注册 API 路径，加独立处理的 `GET /healthz`，合计 201 条固定路径。动态 `GET /ugc/...` 不计入。每一行对应一条路径；同一处理函数的别名也单列。详细操作示例见 [接口测试手册](api-testing.md)。",
    "",
    "**读表规则**：除注册、登录、公开配置和管理接口外，请求通常附 `user` 与 `session`；‘分支校验字段’和‘代码涉及’是静态提取的候选字段，**不等于此路径必填字段**，响应键也可能来自不同成功/失败分支。‘待核实’表示源码分支、旧协议或客户端请求尚需人工确认。游戏操作是测试入口建议，若客户端版本没有该功能则标‘无入口，仅接口测试’。✅ 只表示固定 MuMu 0.1.8 日志中出现过该路径的 POST 200，**不表示功能通过**；— 表示该日志无记录。",
    "",
    "**逐条执行**：记录脱敏请求体、完整响应字段/类型、HTTP 状态、游戏界面结果、数据库或文件前后值；写入接口另测重复请求和重登读回，私有数据接口另测无效会话与跨账号。旧协议可能在 HTTP 200 的响应体中表达业务错误。",
    "",
    "| 序号 | 分组 | 接口名称 / 方法 | 传入参数（候选） | 传回参数（分支候选） | 游戏中如何测试 | 正确结果 | 200 记录 |",
    "| ---: | --- | --- | --- | --- | --- | --- | --- |",
]

for index, (path, handler) in enumerate(routes, 1):
    if path == "/healthz":
        lines.append(f"| {index:03d} | 健康检查 | `GET /healthz` | 无 | `status=ok` | 浏览器或终端请求；无游戏内入口 | HTTP 200 且状态为 ok；只能证明 API 进程响应 | — |")
        continue
    group, game = category(path)
    body = functions.get(handler, "")
    request, response = fields(body)
    if path.startswith("/admin/"):
        request = "管理鉴权 + 查询/请求字段；" + request
    elif not any(x in path for x in ("/create", "/login", "/auth/", "/get_latest_build_version", "/device_capabilities/lookup", "/get_vars", "/get_motd")):
        request = "`user`, `session`；" + request
    if path in ("/account/commerce/product_info", "/admin/api/friends_list", "/admin/api/friends_set", "/admin/api/friends_clear"):
        method = "POST/GET"
    else:
        method = "POST"
    if path in ("/service/relationship/api/v1/update_friend_constellation_pages", "/service/relationship/api/v1/free_gifts/send"):
        method += "/OPTIONS"
    lines.append(f"| {index:03d} | {group} | `{method} {path}` | {request} | {response} | {game} | {expectation(path, group)} | {'✅' if path in seen else '—'} |")

lines.extend(["", "## 记录模板", "", "`序号 / 客户端版本 / 请求体（脱敏） / 响应状态和字段 / 游戏操作 / 游戏结果 / 数据前后值 / 日志时间 / 结论（通过、失败、未执行、占位兼容）`", ""])
OUTPUT.write_text("\n".join(lines), encoding="utf-8")
print(f"wrote {len(routes)} routes, {sum(path in seen for path, _ in routes)} with HTTP 200 evidence")
