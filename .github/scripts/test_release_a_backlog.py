#!/usr/bin/env python3
"""Regression tests for GitHub-only Release A backlog selection."""
from __future__ import annotations

import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


MODULE_PATH = Path(__file__).with_name("release_a_backlog.py")
SPEC = importlib.util.spec_from_file_location("release_a_backlog", MODULE_PATH)
assert SPEC and SPEC.loader
backlog = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = backlog
SPEC.loader.exec_module(backlog)


class GithubOnlyDependencyTests(unittest.TestCase):
    def test_external_blocker_does_not_deadlock_downstream_card(self) -> None:
        cards = [
            backlog.Card("A0-001", "I", (), "[A0-001] External prerequisite", 0),
            backlog.Card("A0-002", "I", ("A0-001",), "[A0-002] Downstream work", 1),
        ]
        registry = {
            "A0-001": {"state": "open", "number": 50},
            "A0-002": {"state": "open", "number": 51},
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            blocklist = Path(tmpdir) / "github-only-blocked.txt"
            blocklist.write_text("A0-001\tExternal infrastructure\n", encoding="utf-8")
            with patch.object(backlog, "GITHUB_ONLY_BLOCKED_PATH", blocklist):
                selected = backlog.select_ready(cards, registry)

        self.assertTrue(selected["ready"])
        self.assertEqual(selected["key"], "A0-002")
        self.assertEqual(selected["issue"], 51)
        self.assertEqual(
            selected["github_only_deferred_dependencies"],
            ["A0-001"],
        )

    def test_blocked_card_is_not_marked_canonically_complete(self) -> None:
        cards = [
            backlog.Card("A0-001", "I", (), "[A0-001] External prerequisite", 0),
        ]
        registry = {"A0-001": {"state": "open", "number": 50}}

        with tempfile.TemporaryDirectory() as tmpdir:
            blocklist = Path(tmpdir) / "github-only-blocked.txt"
            blocklist.write_text("A0-001\tExternal infrastructure\n", encoding="utf-8")
            with patch.object(backlog, "GITHUB_ONLY_BLOCKED_PATH", blocklist):
                selected = backlog.select_ready(cards, registry)

        self.assertFalse(selected["ready"])
        self.assertFalse(selected["implementation_complete"])
        self.assertTrue(selected["github_only_implementation_complete"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
