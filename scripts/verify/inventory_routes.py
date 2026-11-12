"""Read legacy route registrations without importing application code."""
import ast
import json
import re
from pathlib import Path

root = Path(__file__).resolve().parents[2]
source = root.parent / 'LaServer/services/game-api'
routes = []
for path in sorted((source / 'business').rglob('*.py')):
    tree = ast.parse(path.read_text(encoding='utf-8-sig'))
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call) or not isinstance(node.func, ast.Attribute) or node.func.attr != 'add_url_rule':
            continue
        if not node.args or not isinstance(node.args[0], ast.Constant):
            raise ValueError(f'Dynamic route requires manual review: {path}:{node.lineno}')
        kw = {k.arg: k.value for k in node.keywords}
        routes.append({'path': node.args[0].value, 'methods': ast.literal_eval(kw['methods']) if 'methods' in kw else ['GET'], 'handler': ast.unparse(node.args[2]) if len(node.args) > 2 else ast.unparse(kw.get('view_func')), 'source': str(path.relative_to(source)).replace('\\', '/'), 'line': node.lineno, 'status': 'pending'})
out = root / 'json/api/legacy-routes.json'
handler = (root / 'services/api-server/internal/handler/api.go').read_text(encoding='utf-8')
implemented = set(re.findall(r'a\.routes\["([^"]+)"\]', handler))
prefixes = set(re.findall(r'HasPrefix\(r\.URL\.Path, "(/[^"]*)"\)', handler))
previous = {}
if out.exists():
    previous = {row['path']: row.get('client_verified', False) for row in json.loads(out.read_text(encoding='utf-8'))}
for route in routes:
    path = route['path']
    route['status'] = 'implemented_local_only' if path in implemented or any(path.startswith(prefix) for prefix in prefixes) else 'pending'
    route['client_verified'] = bool(previous.get(path, False))
out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(json.dumps(routes, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(f'{len(routes)} registrations, {len({r["path"] for r in routes})} distinct paths')
