#!/usr/bin/env python3
"""Check existing files for local inline Markdown links in repository documents.

External URLs, anchors and reference-style links are intentionally not validated.
Uses Git's file list to avoid scanning ignored dependencies, builds and data.
"""
from pathlib import Path
import re
import subprocess
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parent.parent
LINK = re.compile(r"!?\[[^\]\n]*\]\(([^)\n]+)\)")


def main() -> int:
    result = subprocess.run(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--", "*.md"],
        cwd=ROOT, check=True, capture_output=True,
    )
    paths = sorted(set(result.stdout.decode("utf-8").strip("\0").split("\0")) - {""})
    errors = []
    links = 0
    for name in paths:
        source = ROOT / name
        if not source.is_file():
            continue  # A tracked document may have been deleted in this change.
        # Code samples are not document links.
        markdown = re.sub(r"^```[^\n]*\n.*?^```[^\n]*$", "", source.read_text(encoding="utf-8"), flags=re.M | re.S)
        for match in LINK.finditer(markdown):
            target = match.group(1).strip()
            if target.startswith("<"):
                target = target[1:target.index(">")]
            else:
                target = target.split()[0] if target else ""
            parsed = urlsplit(target)
            if parsed.scheme or parsed.netloc or not parsed.path:
                continue
            links += 1
            path = unquote(parsed.path)
            resolved = ROOT / path.lstrip("/") if path.startswith("/") else source.parent / path
            if not resolved.exists():
                errors.append(f"{name}: missing target {target}")
    for error in errors:
        print(error)
    print(f"Checked {len(paths)} Markdown files and {links} local links; {len(errors)} missing targets.")
    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
