"""Exercise independently built panel and API binaries against an isolated SQLite database."""

import html
import http.cookiejar
import json
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
import tempfile
import time
from contextlib import ExitStack
from urllib.parse import urlencode
from urllib.request import HTTPCookieProcessor, Request, build_opener, urlopen


ROOT = Path(__file__).resolve().parents[2]


def reserve(stack):
    sock = stack.enter_context(socket.socket())
    sock.bind(("127.0.0.1", 0))
    return sock


def start(stack, argv, env=None, output=subprocess.DEVNULL):
    process = subprocess.Popen(argv, env=env, stdout=output, stderr=output)

    def stop():
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=7)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()

    stack.callback(stop)
    return process


def ready(process, url):
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"server exited: {process.returncode}")
        try:
            with urlopen(url, timeout=1) as response:
                if response.status == 200:
                    return
        except OSError:
            time.sleep(0.03)
    raise RuntimeError("startup timed out")


def run(api_binary, panel_binary, browser=False):
    with ExitStack() as stack:
        work = Path(stack.enter_context(tempfile.TemporaryDirectory(dir=ROOT / "tmp")))
        api_sock, panel_sock, room_sock = (reserve(stack) for _ in range(3))
        api, panel, room = [f"http://127.0.0.1:{s.getsockname()[1]}"
                            for s in (api_sock, panel_sock, room_sock)]
        token, password = secrets.token_hex(32), secrets.token_hex(32)
        api_config, panel_config = work / "api.json", work / "panel.json"
        api_config.write_text(json.dumps({
            "listen": api.removeprefix("http://"), "database": "data/test.db",
            "room_host": "127.0.0.1", "room_port": 25565, "login_mode": "legacy_dev",
            "auto_authorize": True, "admin_token": token,
        }))
        panel_config.write_text(json.dumps({
            "listen": panel.removeprefix("http://"), "api_url": api, "room_url": room,
        }))
        subprocess.run([api_binary, "migrate", "-config", api_config], check=True)
        api_sock.close()
        api_cmd = [api_binary, "-config", api_config]
        upstream = start(stack, api_cmd)
        ready(upstream, api + "/healthz")
        people = []
        for name in ["panel-test-a", "panel-test-b"]:
            req = Request(api + "/account/create", json.dumps({"device_name": name,
                          "language": "zh"}).encode(), {"Content-Type": "text/plain"})
            with urlopen(req, timeout=3) as response:
                people.append(json.load(response)["user"])
        user, friend = people
        panel_sock.close()
        panel_cmd = [panel_binary, "-config", panel_config]
        env = dict(os.environ, LASERVER_PANEL_PASSWORD=password, LASERVER_PANEL_API_TOKEN=token)
        panel_log = work / "panel.log"
        output = stack.enter_context(panel_log.open("wb"))
        process = start(stack, panel_cmd, env, output)
        ready(process, panel + "/login")
        opener = build_opener(HTTPCookieProcessor(http.cookiejar.CookieJar()))

        def page(path, form=None):
            data = None if form is None else urlencode(form).encode()
            req = Request(panel + path, data, {"Origin": panel})
            with opener.open(req, timeout=4) as result:
                body = result.read().decode()
                assert password not in body and token not in body
                return body

        overview = page("/login", {"password": password})
        assert "在线" in overview and "不可达" in overview
        query = "/friends?" + urlencode({"user": user})
        csrf = html.unescape(re.search(r'name="csrf" value="([^"]+)"', page(query))[1])
        for action, status in (("add", "active"), ("remove", "removed"), ("restore", "active")):
            body = page("/friends/set", dict(csrf=csrf, user=user, friend=friend,
                                           action=action))
            assert "操作成功" in body, action
            body = page(query)
            assert friend in body and f"<td>{status}</td><td>{status}</td>" in body, action
        if browser:
            subprocess.run(["node", ROOT / "scripts/admin-panel/browser-smoke.mjs"],
                           input=json.dumps(dict(origin=panel, password=password, user=user,
                                                 profile=str(work.resolve() / "chrome"))),
                           text=True, check=True, timeout=30)
        upstream.terminate()
        assert upstream.wait(timeout=7) == 0
        assert "API 结果" in page(query)
        upstream = start(stack, api_cmd)
        ready(upstream, api + "/healthz")
        assert "<td>active</td><td>active</td>" in page(query)
        body = page("/friends/clear", dict(csrf=csrf, user=user, confirm=user))
        assert "已移除 1" in body
        assert "<td>removed</td><td>removed</td>" in page(query)
        process.terminate()
        assert process.wait(timeout=7) == 0
        logged = panel_log.read_bytes()
        assert b"friend management" in logged
        assert password.encode() not in logged and token.encode() not in logged
        process = start(stack, panel_cmd, dict(env, LASERVER_PANEL_API_TOKEN=secrets.token_hex(32)))
        ready(process, panel + "/login")
        page("/login", {"password": password})
        assert "无法读取 API 结果" in page(query)


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("api", type=Path)
    parser.add_argument("panel", type=Path)
    parser.add_argument("--browser", action="store_true")
    args = parser.parse_args()
    run(args.api.resolve(), args.panel.resolve(), args.browser)
    print("Real API/panel: login, status, add/remove/restore, restart persistence, clear and wrong token passed.")
