#!/usr/bin/env python3
"""A0-002 regression test for the canonical AIDI 2.0 monorepo layout."""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]

REQUIRED_DIRECTORIES = (
    "cmd/aidi",
    "internal",
    "workers/src",
    "workers/tests",
    "web/src",
    "docs",
    ".github/workflows",
    ".github/scripts",
    ".automation",
)

REQUIRED_FILES = (
    "go.mod",
    "Makefile",
    "README.md",
    "DEVELOPMENT.md",
    "workers/pyproject.toml",
    "web/package.json",
    "web/package-lock.json",
    "docs/MONOREPO_STRUCTURE.md",
)


def main() -> int:
    missing: list[str] = []

    for relative in REQUIRED_DIRECTORIES:
        path = ROOT / relative
        if not path.is_dir():
            missing.append(f"directory:{relative}")

    for relative in REQUIRED_FILES:
        path = ROOT / relative
        if not path.is_file():
            missing.append(f"file:{relative}")

    if missing:
        print("A0-002 monorepo structure verification failed:", file=sys.stderr)
        for item in missing:
            print(f" - {item}", file=sys.stderr)
        return 1

    print(
        "A0-002 monorepo structure verified: "
        f"{len(REQUIRED_DIRECTORIES)} directories, {len(REQUIRED_FILES)} files"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
