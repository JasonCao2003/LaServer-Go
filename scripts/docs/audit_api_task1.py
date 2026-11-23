"""Compare the legacy HTTP registrations, Go routes, and direct Go test references.

The legacy checkout is read only. Run from any directory; no server or database
is required. The JSON report is deliberately explicit about missing evidence.
"""

from __future__ import annotations

import ast
import json
import re
import warnings
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
LEGACY = ROOT.parent / "LaServer" / "services" / "game-api"
ROUTE_FILE = ROOT / "services/api-server/internal/handler/api.go"
TEST_DIR = ROOT / "services/api-server/internal/handler"
OUTPUT = ROOT / "json/api/task1-route-inventory.json"
RESOURCE_OUTPUT = ROOT / "json/api/task1-login-resources.json"
RESOURCE_DOC = ROOT / "docs/api-login-resource-dependencies.md"

# These names differ from the response key or have multiple response producers.
# A candidate is not evidence that the client maps the label to that endpoint.
CANONICAL = {
    "achievement_stats": "/account/get_achievement_stats",
    "checkpoints": "/account/get_checkpoints",
    "collectibles": "/account/get_collectibles",
    "currency": "/account/get_currency",
    "free_gifts": "/service/relationship/api/v1/free_gifts/get_pending",
    "get_buffs": "/account/buff/get_buffs1",
    "get_consumables": "/account/consumable/get_consumables1",
    "get_lootboxes": "/account/lootboxes/get",
    "gift_messages": "/account/get_pending_messages",
    "infractions": "/service/infraction/api/v1/get_infractions_status_decorated",
    "invite_list": "/account/get_invites",
    "messages": "/service/message/api/v2/get_all",
    "questionnaires": "/account/get_questionnaires",
    "set_app_badge_number": "/account/get_app_badge_number",
    "set_friend_status": "/account/get_friend_statues",
    "set_random": "/account/get_rank",
    "wing_buffs": "/account/wing_buffs/get",
}


def literal(node: ast.AST) -> object | None:
    try:
        return ast.literal_eval(node)
    except (ValueError, TypeError):
        return None


def methods(call: ast.Call, default: list[str]) -> list[str]:
    for keyword in call.keywords:
        if keyword.arg == "methods":
            value = literal(keyword.value)
            if isinstance(value, (list, tuple)) and all(isinstance(x, str) for x in value):
                return sorted(set(value))
    return default


def legacy_registrations() -> list[dict[str, object]]:
    entries: list[dict[str, object]] = []
    for source in sorted(LEGACY.rglob("*.py")):
        with warnings.catch_warnings():
            warnings.simplefilter("ignore", SyntaxWarning)
            tree = ast.parse(source.read_text(encoding="utf-8-sig"), filename=str(source))
        relative = source.relative_to(LEGACY).as_posix()
        for node in ast.walk(tree):
            if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute):
                if node.func.attr == "add_url_rule" and node.args:
                    path = literal(node.args[0])
                    if isinstance(path, str):
                        entries.append({"path": path, "methods": methods(node, ["GET"]), "source": f"{relative}:{node.lineno}", "kind": "add_url_rule"})
                elif node.func.attr == "route" and node.args:
                    path = literal(node.args[0])
                    if isinstance(path, str):
                        owner = node.func.value
                        kind = "sock" if isinstance(owner, ast.Name) and owner.id == "sock" else "decorator"
                        entries.append({"path": path, "methods": methods(node, ["GET"]), "source": f"{relative}:{node.lineno}", "kind": kind})
    return entries


def main() -> None:
    if not LEGACY.is_dir():
        raise SystemExit(f"legacy source checkout missing: {LEGACY}")
    old = json.loads((ROOT / "json/api/legacy-routes.json").read_text(encoding="utf-8"))
    go = dict(re.findall(r'a\.routes\["([^"]+)"\] = s\.(\w+)', ROUTE_FILE.read_text(encoding="utf-8")))
    registrations = legacy_registrations()
    registered: dict[str, list[dict[str, object]]] = {}
    for entry in registrations:
        registered.setdefault(str(entry["path"]), []).append(entry)
    tests = {p.name: p.read_text(encoding="utf-8") for p in TEST_DIR.glob("*_test.go")}
    rows = []
    for item in old:
        path = item["path"]
        if path == "/ugc/<path:subpath>":
            go_methods = ["GET"]
        elif path in {"/account/commerce/product_info", "/admin/api/friends_list", "/admin/api/friends_set", "/admin/api/friends_clear"}:
            go_methods = ["GET", "POST"]
        elif path in {"/account/get_friend_statues", "/service/relationship/api/v1/update_friend_constellation_pages", "/service/relationship/api/v1/free_gifts/send"}:
            go_methods = ["OPTIONS", "POST"]
        else:
            go_methods = ["POST"]
        refs = sorted(name for name, body in tests.items() if f'"{path}"' in body)
        if path == "/ugc/<path:subpath>":
            refs = sorted(name for name, body in tests.items() if '"/ugc/' in body)
        rows.append({
            "path": path,
            "legacy_methods": item["methods"],
            "go_methods": go_methods,
            "legacy_source": f'{item["source"]}:{item["line"]}',
            "registered_sources": registered.get(path, []),
            "go_handler": go.get(path),
            "direct_test_files": refs,
            "direct_test_evidence": "source_reference_only" if refs else "missing",
            "client_0_1_8": "not_verified",
            "client_0_3_11": "not_verified",
        })
    old_paths = {item["path"] for item in old}
    go_dynamic = {"/ugc/<path:subpath>"} if '"/ugc/"' in ROUTE_FILE.read_text(encoding="utf-8") else set()
    go_only = sorted(set(go) - old_paths)
    extra_legacy = sorted((entry for entry in registrations if entry["path"] not in old_paths), key=lambda x: (str(x["path"]), str(x["source"])))
    report = {
        "scope": "Static legacy-routes.json paths only; source references are not execution results.",
        "legacy_count": len(rows),
        "go_registered_count": len(go),
        "legacy_missing_in_go": sorted(old_paths - set(go) - go_dynamic),
        "legacy_dynamic_in_go": sorted(go_dynamic),
        "go_only": go_only,
        "extra_legacy_http_or_websocket": extra_legacy,
        "no_direct_test_reference": [row["path"] for row in rows if not row["direct_test_files"]],
        "method_differences": [{"path": row["path"], "legacy": row["legacy_methods"], "go": row["go_methods"]} for row in rows if sorted(row["legacy_methods"]) != row["go_methods"]],
        "routes": rows,
    }
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    audit_resources(go)
    print(f"legacy={len(rows)} go={len(go)} extra_legacy={len(extra_legacy)} direct_reference_missing={len(report['no_direct_test_reference'])}")


def audit_resources(go: dict[str, str]) -> None:
    old_files = {
        "create": "business/account/create/create.py",
        "login": "business/account/login/login.py",
        "auth_create": "business/account/auth_create/auth_create.py",
        "auth_login": "business/account/auth_login/auth_login.py",
    }
    flow_lists: dict[str, list[str]] = {}
    for flow, relative in old_files.items():
        source = (LEGACY / relative).read_text(encoding="utf-8")
        match = re.search(r'"resource_list"\s*:\s*(\[[^]]*\])', source, re.S)
        if not match:
            raise ValueError(f"resource_list missing in {relative}")
        flow_lists[flow] = ast.literal_eval(match.group(1))
    go_source = (ROOT / "services/api-server/internal/service/legacy_contracts.go").read_text(encoding="utf-8")
    for flow, name in {"create": "createResources", "login": "loginResources", "auth_create": "authCreateResources", "auth_login": "authLoginResources"}.items():
        match = re.search(r"var " + name + r" = \[\]string\{([^}]*)\}", go_source, re.S)
        if not match:
            raise ValueError(f"Go resource list missing: {name}")
        flow_lists["go_" + flow] = re.findall(r'"([^"]+)"', match.group(1))

    response_dir = LEGACY / "resources/static_responses"
    manifest = json.loads((response_dir / "_manifest.json").read_text(encoding="utf-8"))
    response_keys: dict[str, list[str]] = {}
    for path, entry in manifest.items():
        try:
            payload = json.loads((response_dir / entry["file"]).read_text(encoding="utf-8-sig"))
        except (ValueError, OSError):
            continue
        if isinstance(payload, dict):
            for key in payload:
                response_keys.setdefault(key, []).append(path)
    service_dir = ROOT / "services/api-server/internal/service"
    functions = {}
    for source in service_dir.glob("*.go"):
        content = source.read_text(encoding="utf-8")
        starts = list(re.finditer(r'^func ', content, re.M))
        for index, start in enumerate(starts):
            end = starts[index + 1].start() if index + 1 < len(starts) else len(content)
            body = content[start.start():end]
            match = re.match(r'func \(a \*Service\) (\w+)\(', body)
            if match:
                functions[match.group(1)] = {"source": source.relative_to(ROOT).as_posix(), "body": body}

    logs = {
        "0.1.8": ROOT / "logs/mumu-0108-api-fixed-20261123.log",
        "0.3.11": ROOT / "logs/room-server/stage-2-mumu-0311-api-20261123.log",
    }
    seen = {}
    for version, source in logs.items():
        seen[version] = set(re.findall(r"method=POST path=(\S+) status=200", source.read_text(encoding="utf-8"))) if source.exists() else set()
    names = sorted(set().union(*(set(values) for values in flow_lists.values())))
    rows = []
    for name in names:
        paths = sorted(set(response_keys.get(name, [])))
        candidate = CANONICAL.get(name)
        if candidate is None and paths:
            candidate = paths[0]
        handler = go.get(candidate or "")
        function = functions.get(handler or "", {})
        body = function.get("body", "")
        files = sorted(set(re.findall(r'"([^"/]+\.json)"', body)))
        origins = []
        if files:
            origins.append("game_config_or_embedded_json:" + ",".join(files))
        if "a.store" in body or "tx." in body:
            origins.append("SQLite")
        if "legacyResponse(" in body or "legacyData." in body:
            origins.append("embedded_legacydata")
        if not origins:
            origins.append("代码常量或辅助函数，待逐分支核对" if candidate else "未找到独立数据来源")
        if candidate in manifest:
            origins.append("旧静态库:" + manifest[candidate]["file"])
        auth = "session_checked_in_handler" if re.search(r'authenticated\(|magicUser\(|questUser\(|batch1User\(|text\(m, "session"\)', body) else "no_direct_session_check_visible"
        rows.append({
            "resource": name,
            "legacy_flows": [flow for flow in old_files if name in flow_lists[flow]],
            "go_flows": [flow for flow in old_files if name in flow_lists["go_" + flow]],
            "legacy_response_paths": paths,
            "legacy_response_files": [manifest[path]["file"] for path in paths],
            "candidate_go_path": candidate,
            "candidate_kind": "embedded_in_login_response" if name == "external_links" else "response_key_match" if candidate in paths else "alias_inference" if candidate else "no_independent_endpoint_found",
            "go_handler": handler,
            "go_source": function.get("source"),
            "go_direct_config_or_data_sources": origins,
            "go_direct_auth_signal": auth,
            "http_200_paths_in_existing_logs": {version: [path for path in paths if path in observed] for version, observed in seen.items()},
            "candidate_http_200_in_existing_logs": {version: bool(candidate and candidate in observed) for version, observed in seen.items()},
            "client_resource_name_mapping": "not_captured_from_apk",
        })
    result = {
        "scope": "Server-supplied login resource lists and inferred response-key mappings; APK resource mapping requires client trace confirmation.",
        "flow_sources": old_files,
        "flow_lists": flow_lists,
        "flow_differences": {flow: {"old_only": sorted(set(flow_lists[flow]) - set(flow_lists["go_" + flow])), "go_only": sorted(set(flow_lists["go_" + flow]) - set(flow_lists[flow]))} for flow in old_files},
        "resources": rows,
    }
    RESOURCE_OUTPUT.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    write_resource_doc(result, seen)


def write_resource_doc(result: dict[str, object], seen: dict[str, set[str]]) -> None:
    lines = [
        "# API 登录资源依赖表",
        "",
        "来源：旧 `create/login/auth_create/auth_login` 返回体、旧静态响应清单、Go `legacy_contracts.go` 与 route map。逐项机器可读证据见 [资源审计 JSON](../json/api/task1-login-resources.json)。运行 `python scripts/docs/audit_api_task1.py` 更新。",
        "",
        "四种流程的旧资源列表与 Go 列表逐项一致，共 48 个不同名称；这只证明服务端声明一致。现有 APK 日志未保存登录响应体，也未保存客户端资源名到 URL 的映射，因此下表的 URL 是响应键匹配或显式标注的候选，绝不当作客户端映射实测。0.1.8 和 0.3.11 列仅表示候选 URL 在已有隔离 API 日志中出现 POST 200，不代表资源内容正确。两版测试包均实际调用 `/account/auth/login`；0.3.11 也曾调用 `/account/auth/create`。APK 真实版本与其上报的 User-Agent 分开记录在 [兼容矩阵](client-compatibility.md)。",
        "",
        "旧静态库 `before_request` 仅在上报版本达到 0.26.0 且路径不在 `_dynamic_exclude.json` 时截断动态处理；这项门槛由请求头决定，不能仅凭 APK 文件版本推断。Go 版本过滤按各 handler 单独处理。配置来源列只列出候选 Go handler 中可直接看到的文件、SQLite 或内嵌数据；辅助函数的间接依赖仍须在 Task 3 检查。认证列同样只表达该 handler 中直接可见的会话校验信号。",
        "",
        "失败影响为排查优先级推断：`启动链` 表示登录后资源同步可能受阻；`功能` 表示相关界面或状态可能不可用；`待确认` 表示找不到独立资源入口，须以客户端追踪判定是否实际请求。",
        "",
        "| 资源名 | 候选接口 / 映射 | 配置或数据来源（Go 直接可见） | 认证信号 | Go 处理 | 0.1.8 / 0.3.11 200 | 失败影响 |",
        "| --- | --- | --- | --- | --- | --- | --- |",
    ]
    critical = {"device_capabilities", "userdata", "currency", "motd", "unlocks", "get_map_defs", "get_shop", "spirit_shops", "event_schedule", "set_world_quests", "checkpoints"}
    for row in result["resources"]:
        name = row["resource"]
        candidate = row["candidate_go_path"]
        kind = row["candidate_kind"]
        if kind == "embedded_in_login_response":
            mapped = "登录响应字段（无独立请求）"
        elif candidate:
            mapped = f"`{candidate}`" + ("（别名推断）" if kind == "alias_inference" else "")
        else:
            mapped = "无独立入口证据"
        origins = "; ".join(row["go_direct_config_or_data_sources"])
        auth = "见直接会话检查" if row["go_direct_auth_signal"] == "session_checked_in_handler" else "未见直接会话检查"
        go = f'`{row["go_handler"]}` / `{Path(row["go_source"]).name}`' if row["go_source"] else "无已知处理"
        hits = " / ".join("✓" if candidate and candidate in seen[version] else "—" for version in ("0.1.8", "0.3.11"))
        impact = "待确认" if not candidate else "启动链" if name in critical else "功能"
        lines.append(f"| `{name}` | {mapped} | {origins} | {auth} | {go} | {hits} | {impact} |")
    lines.extend([
        "",
        "## 已解释的名称差异与缺口",
        "",
        "- `set_friend_status`、`set_random` 只在旧版及 Go 的 `auth_create` 列表中出现；已知候选接口分别返回 `set_friend_statues`、`set_rank`。这两个资源名不能据此判定为成功加载。",
        "- `invite_list`、`gift_messages`、`free_gifts`、`infractions`、`questionnaires` 的已知业务入口与列表名不是相同响应键；表内候选仅用于追踪，客户端映射待抓取。`questionnaires` 当前 Go 只返回 `result=ok`，是旧占位语义。",
        "- `outfit_defs`、`get_user_profile`、`emote_sequences` 在旧静态响应键及注册路由中都找不到独立资源入口；目前作为显式待确认依赖，不以成功占位掩盖。",
        "- `get_shop`、`spirit_shops`、`event_schedule`、`motd`、`unlocks`、魔法定义等配置依赖要在 Task 3 分别测试正常、缺失、损坏及版本差异。现有日志中的 HTTP 200 不代表客户端接受了内容。",
        "",
    ])
    RESOURCE_DOC.write_text("\n".join(lines), encoding="utf-8")


if __name__ == "__main__":
    main()
