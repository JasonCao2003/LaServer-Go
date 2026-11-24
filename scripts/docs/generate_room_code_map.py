"""Generate a per-file and per-function code map for services/room-server."""

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SERVICE = ROOT / "services/room-server"
OUTPUT = ROOT / "docs/room-server-code-map.md"


def receiver_type(recv: str) -> str:
    cleaned = recv.strip()
    if not cleaned:
        return ""
    last = cleaned.split()[-1]
    return last.lstrip("*")


def functions(lines):
    result = []
    for i, line in enumerate(lines):
        if not line.startswith("func "):
            continue
        j = i - 1
        block = []
        while j >= 0 and lines[j].strip().startswith("//"):
            block.insert(0, lines[j].strip())
            j -= 1
        match = re.match(r"^func (?:\(([^)]*)\) )?([A-Za-z0-9_]+)\(", line)
        if not match:
            continue
        recv, name = match.group(1), match.group(2)
        rt = receiver_type(recv) if recv else ""
        display = f"{rt}.{name}" if rt else name
        comment = block[-1].lstrip("/ ").strip() if block else ""
        result.append((display, comment))
    return result


groups = {}
for path in sorted(SERVICE.rglob("*.go")):
    if path.name.endswith("_test.go"):
        continue
    rel = path.relative_to(SERVICE)
    lines = path.read_text(encoding="utf-8").splitlines()
    header = ""
    for line in lines:
        if line.startswith("package "):
            break
        if line.startswith("// ") and not line.startswith("//go:"):
            header = line.lstrip("/ ").strip()
            break
    groups.setdefault(str(rel.parent), []).append((rel, header, functions(lines)))

rows = ["# room-server 文件与函数说明", ""]
rows.append("> 由 `scripts/docs/generate_room_code_map.py` 从源码注释生成。")
rows.append("> 范围：`services/room-server` 下全部非测试 Go 文件；函数说明取自函数上方的一行中文注释。")
rows.append("> 测试文件（`_test.go`）未包含。相关文档见 [api-server 文件与函数说明](api-server-code-map.md)。")
rows.append("")

stats = []
total_files = 0
total_funcs = 0
for dirname, files in sorted(groups.items()):
    funcs = sum(len(f[2]) for f in files)
    stats.append((dirname, len(files), funcs))
    total_files += len(files)
    total_funcs += funcs

rows.append("## 统计")
rows.append("")
rows.append("| 目录 | 文件数 | 函数数 |")
rows.append("| --- | ---: | ---: |")
for dirname, files, funcs in stats:
    label = "." if dirname == "." else dirname
    rows.append(f"| `{label}` | {files} | {funcs} |")
rows.append(f"| **合计** | **{total_files}** | **{total_funcs}** |")
rows.append("")

for dirname, files in sorted(groups.items()):
    heading = "根目录" if dirname == "." else dirname
    rows.append(f"## {heading}")
    rows.append("")
    for rel, header, funcs in files:
        rows.append(f"### `{rel.as_posix()}`")
        rows.append("")
        rows.append(f"功能：{header or '（未提供文件说明）'}")
        rows.append("")
        if funcs:
            rows.append("| 函数 | 作用 |")
            rows.append("| --- | --- |")
            for name, comment in funcs:
                rows.append(f"| `{name}` | {comment or '（未提供函数说明）'} |")
        else:
            rows.append("无顶层函数。")
        rows.append("")

OUTPUT.write_text("\n".join(rows), encoding="utf-8")
print(f"wrote {OUTPUT} files={total_files} funcs={total_funcs}")
