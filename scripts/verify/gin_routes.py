"""Read method/path declarations and POST service names from the Go router for reports."""
import re
from pathlib import Path


def registered_routes():
    handler_dir = (Path(__file__).resolve().parents[2] /
                   "services/api-server/internal/handler")
    sources = [handler_dir / "router.go"] + sorted(handler_dir.glob("routes*.go"))
    methods = {}
    handlers = {}
    for source_path in sources:
        source = source_path.read_text(encoding="utf-8")
        for method, path, callback in re.findall(r'^\s*r\.(GET|POST|OPTIONS)\("([^"]+)", (.+)$', source, re.M):
            methods.setdefault(path, []).append(method)
            if method == "POST":
                match = re.fullmatch(r'post\(s\.(\w+)\)\)', callback)
                if not match:
                    raise ValueError(f"Unknown service callback for POST {path}: {callback}")
                handlers[path] = match.group(1)
        methods.setdefault(path, []).append(method)
        if method == "POST":
            match = re.fullmatch(r'post\(s\.(\w+)\)\)', callback)
            if not match:
                raise ValueError(f"Unknown service callback for POST {path}: {callback}")
            handlers[path] = match.group(1)
    if not handlers or "/ugc/*subpath" not in methods or "/healthz" not in methods:
        raise ValueError("Incomplete Gin route inventory")
    return handlers, methods


def gin_path(legacy_path):
    return "/ugc/*subpath" if legacy_path == "/ugc/<path:subpath>" else legacy_path
