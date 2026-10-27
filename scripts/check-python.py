#!/usr/bin/env python3
"""Syntax check service and helper scripts without imports, pyc or data writes."""
import ast
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
paths = sorted((ROOT / "python_ai_service/app").rglob("*.py"))
paths += sorted((ROOT / "scripts").glob("*.py"))
for path in paths:
    ast.parse(path.read_text(encoding="utf-8"), filename=str(path.relative_to(ROOT)))
print(f"Parsed {len(paths)} Python files; runtime dependencies and behavior were not checked.")
