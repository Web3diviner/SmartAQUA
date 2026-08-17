"""Static check: every `from app...`/`from ingestion...` import resolves.

Catches drift between modules without needing the runtime dependencies
installed. Run from the aquadoc/ directory:

    python tools/check_imports.py
"""

from __future__ import annotations

import ast
import pathlib
import sys

ROOTS = ("app", "ingestion")


def module_name(path: pathlib.Path) -> str:
    parts = list(path.with_suffix("").parts)
    if parts and parts[-1] == "__init__":
        parts.pop()
    return ".".join(parts)


def collect_defined() -> dict[str, set[str]]:
    """Top-level names each module exposes."""
    defined: dict[str, set[str]] = {}
    for path in sorted(pathlib.Path(".").rglob("*.py")):
        if "__pycache__" in path.parts or path.parts[0] not in ROOTS:
            continue
        tree = ast.parse(path.read_text(encoding="utf-8"))
        names: set[str] = set()
        for node in tree.body:
            if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef | ast.ClassDef):
                names.add(node.name)
            elif isinstance(node, ast.Assign):
                names.update(t.id for t in node.targets if isinstance(t, ast.Name))
            elif isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name):
                names.add(node.target.id)
            elif isinstance(node, ast.Import | ast.ImportFrom):
                names.update(alias.asname or alias.name.split(".")[0] for alias in node.names)
        defined[module_name(path)] = names
    return defined


def main() -> int:
    defined = collect_defined()
    problems: list[str] = []

    for path in sorted(pathlib.Path(".").rglob("*.py")):
        if "__pycache__" in path.parts or path.parts[0] not in ROOTS:
            continue
        tree = ast.parse(path.read_text(encoding="utf-8"))
        for node in ast.walk(tree):
            if not isinstance(node, ast.ImportFrom):
                continue
            target = node.module
            if not target or target.split(".")[0] not in ROOTS:
                continue
            if target not in defined:
                problems.append(f"{path}:{node.lineno}  missing module: {target}")
                continue
            for alias in node.names:
                if alias.name == "*":
                    continue
                # A name may be either a top-level symbol or a submodule.
                if alias.name not in defined[target] and f"{target}.{alias.name}" not in defined:
                    problems.append(f"{path}:{node.lineno}  from {target} import {alias.name}")

    if problems:
        print("\n".join(problems))
        print(f"\n{len(problems)} unresolved import(s)")
        return 1

    print(f"OK: all internal imports resolve across {len(defined)} modules")
    return 0


if __name__ == "__main__":
    sys.exit(main())
