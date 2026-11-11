"""Static declaration audit only: never imports or starts the legacy application."""
import ast
import json
from pathlib import Path

root = Path(__file__).resolve().parents[2]
source = root.parent / 'LaServer/services/game-api'
records = []
failures = []

def literal(node):
    try:
        return ast.literal_eval(node)
    except (ValueError, TypeError):
        return None

for path in sorted(source.rglob('*.py')):
    if any(part in {'var', 'logs', '__pycache__', 'bak'} for part in path.relative_to(source).parts):
        continue
    try:
        tree = ast.parse(path.read_text(encoding='utf-8-sig'))
    except (SyntaxError, UnicodeError) as exc:
        failures.append({'source': path.relative_to(source).as_posix(), 'error': type(exc).__name__})
        continue
    parents = {}
    for parent in ast.walk(tree):
        for child in ast.iter_child_nodes(parent):
            parents[child] = parent
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call) or not isinstance(node.func, ast.Attribute):
            continue
        attr = node.func.attr
        if attr not in {'route', 'add_url_rule', 'websocket', 'on', 'before_request', 'register_blueprint'}:
            continue
        ancestors = []
        current = node
        while current in parents:
            current = parents[current]
            if isinstance(current, (ast.FunctionDef, ast.AsyncFunctionDef)):
                ancestors.append({'function': current.name})
            elif isinstance(current, (ast.If, ast.Try, ast.For, ast.While)):
                ancestors.append({'conditional': type(current).__name__, 'line': current.lineno})
        kwargs = {k.arg: k.value for k in node.keywords}
        value = literal(node.args[0]) if node.args else None
        # Never emit arbitrary call expressions: they may contain operational data.
        records.append({
            'kind': attr,
            'receiver': ast.unparse(node.func.value),
            'path_or_event': value if isinstance(value, str) else None,
            'dynamic_argument': value is None,
            'methods': literal(kwargs['methods']) if 'methods' in kwargs else None,
            'source': path.relative_to(source).as_posix(),
            'line': node.lineno,
            'enclosing': ancestors,
            'runtime_enabled': 'unverified',
            'authentication': 'requires_handler_and_hook_review',
            'writes': 'requires_handler_and_helper_review',
            'version_range': 'requires_dispatch_review',
        })
report = {'scope': 'Python AST declarations; not runtime route discovery', 'records': records, 'parse_failures': failures}
output = root / 'json/api/legacy-declarations.json'
output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
print(f'{len(records)} declaration candidates; {len(failures)} parse failures')
