#!/usr/bin/env python3
"""Validate and select work from the approved AIDI 2.0 Release A backlog.

The approved Release A tracker manifest is authoritative. This helper never
creates scope. It only validates the committed dependency index and maps it to
GitHub Issues whose titles begin with the exact stable key, e.g. [A1-001].
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

KEY_RE = re.compile(r"^\[([A-Z0-9-]+)\]\s")
EXPECTED_IMPLEMENTATION = 86
EXPECTED_GATES = 15
EXPECTED_TOTAL = 101
EXPECTED_SOURCE_SHA256 = "91f720101294c36b8243cbead1c064d67f8d27f33dd52386f256a9102c8393aa"


class BacklogError(RuntimeError):
    pass


@dataclass(frozen=True)
class Card:
    key: str
    kind: str
    dependencies: tuple[str, ...]
    title: str
    order: int


def load_cards(path: Path) -> list[Card]:
    lines = path.read_text(encoding="utf-8").splitlines()
    if not lines or not lines[0].startswith("source_sha256\t"):
        raise BacklogError("missing source_sha256 header")
    source_sha = lines[0].split("\t", 1)[1].strip()
    if source_sha != EXPECTED_SOURCE_SHA256:
        raise BacklogError(
            f"unexpected source manifest sha256: {source_sha!r}; "
            f"expected {EXPECTED_SOURCE_SHA256}"
        )

    cards: list[Card] = []
    for order, raw in enumerate(lines[1:]):
        if not raw.strip():
            continue
        parts = raw.split("\t", 3)
        if len(parts) != 4:
            raise BacklogError(f"invalid TSV row {order + 2}: {raw!r}")
        key, kind, deps_raw, title = parts
        if kind not in {"I", "G"}:
            raise BacklogError(f"{key}: unknown card kind {kind!r}")
        deps = tuple(d for d in deps_raw.split(",") if d)
        cards.append(Card(key=key, kind=kind, dependencies=deps, title=title, order=order))
    return cards


def validate_cards(cards: list[Card]) -> dict[str, Card]:
    by_key: dict[str, Card] = {}
    for card in cards:
        if card.key in by_key:
            raise BacklogError(f"duplicate stable key: {card.key}")
        expected_prefix = f"[{card.key}] "
        if not card.title.startswith(expected_prefix):
            raise BacklogError(f"{card.key}: title does not begin with {expected_prefix!r}")
        by_key[card.key] = card

    impl = sum(c.kind == "I" for c in cards)
    gates = sum(c.kind == "G" for c in cards)
    if (impl, gates, len(cards)) != (
        EXPECTED_IMPLEMENTATION,
        EXPECTED_GATES,
        EXPECTED_TOTAL,
    ):
        raise BacklogError(
            f"card counts mismatch: implementation={impl}, gates={gates}, "
            f"total={len(cards)}"
        )

    missing = sorted(
        {dep for card in cards for dep in card.dependencies if dep not in by_key}
    )
    if missing:
        raise BacklogError(f"unresolved hard dependencies: {missing}")

    visiting: set[str] = set()
    visited: set[str] = set()

    def visit(key: str, stack: list[str]) -> None:
        if key in visited:
            return
        if key in visiting:
            start = stack.index(key)
            raise BacklogError(
                "dependency cycle: " + " -> ".join(stack[start:] + [key])
            )
        visiting.add(key)
        stack.append(key)
        for dep in by_key[key].dependencies:
            visit(dep, stack)
        stack.pop()
        visiting.remove(key)
        visited.add(key)

    for key in by_key:
        visit(key, [])

    return by_key


def load_issue_registry(path: Path, by_key: dict[str, Card]) -> dict[str, dict]:
    raw = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(raw, list):
        raise BacklogError("issues JSON must be a list")

    registry: dict[str, dict] = {}
    duplicate_keys: list[str] = []
    for issue in raw:
        if issue.get("pull_request"):
            continue
        title = issue.get("title") or ""
        match = KEY_RE.match(title)
        if not match:
            continue
        key = match.group(1)
        if key not in by_key:
            raise BacklogError(f"GitHub Issue uses unknown Release A stable key: {key}")
        if title != by_key[key].title:
            raise BacklogError(
                f"{key}: GitHub Issue title drift; got {title!r}, "
                f"expected {by_key[key].title!r}"
            )
        if key in registry:
            duplicate_keys.append(key)
            continue
        registry[key] = issue

    if duplicate_keys:
        raise BacklogError(
            "duplicate GitHub Issues for stable keys: " + ", ".join(sorted(set(duplicate_keys)))
        )

    missing = [key for key in by_key if key not in registry]
    if missing:
        raise BacklogError(
            f"canonical tracker import incomplete: {len(missing)} cards missing; "
            f"first={missing[:10]}"
        )

    return registry


def select_ready(cards: list[Card], registry: dict[str, dict]) -> dict:
    closed = {key for key, issue in registry.items() if issue.get("state") == "closed"}

    ready: list[Card] = []
    for card in cards:
        if card.kind != "I":
            continue
        issue = registry[card.key]
        if issue.get("state") != "open":
            continue
        if all(dep in closed for dep in card.dependencies):
            ready.append(card)

    if not ready:
        implementation_done = all(
            registry[c.key].get("state") == "closed" for c in cards if c.kind == "I"
        )
        return {
            "ready": False,
            "implementation_complete": implementation_done,
            "closed_count": len(closed),
            "total_count": len(cards),
        }

    card = min(ready, key=lambda c: c.order)
    issue = registry[card.key]
    return {
        "ready": True,
        "key": card.key,
        "issue": issue["number"],
        "title": card.title,
        "dependencies": list(card.dependencies),
        "closed_count": len(closed),
        "total_count": len(cards),
    }


def main(argv: Iterable[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("validate", "select"))
    parser.add_argument(
        "--cards",
        default=".automation/backlog/release-a/cards.tsv",
        type=Path,
    )
    parser.add_argument("--issues", type=Path)
    args = parser.parse_args(argv)

    try:
        cards = load_cards(args.cards)
        by_key = validate_cards(cards)
        if args.command == "validate":
            print(
                json.dumps(
                    {
                        "valid": True,
                        "implementation": EXPECTED_IMPLEMENTATION,
                        "gates": EXPECTED_GATES,
                        "total": EXPECTED_TOTAL,
                        "source_sha256": EXPECTED_SOURCE_SHA256,
                    },
                    sort_keys=True,
                )
            )
            return 0

        if args.issues is None:
            raise BacklogError("--issues is required for select")
        registry = load_issue_registry(args.issues, by_key)
        print(json.dumps(select_ready(cards, registry), ensure_ascii=False, sort_keys=True))
        return 0
    except (BacklogError, json.JSONDecodeError, OSError, KeyError, TypeError) as exc:
        print(f"release-a-backlog: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
