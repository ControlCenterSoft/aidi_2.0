#!/usr/bin/env python3
"""Guard the minimum Release A CI quality-gate contract."""

from pathlib import Path

CI_PATH = Path(__file__).resolve().parents[1] / "workflows" / "ci.yml"

REQUIRED_SNIPPETS = {
    "formatting gate": 'run: test -z "$(gofmt -l .)"',
    "static-analysis gate": "run: go vet ./...",
    "Go compile gate": "run: go build ./...",
    "Go test gate": "run: go test -race -coverprofile=coverage.out ./...",
    "Python compile gate": "run: python -m compileall -q workers/src workers/tests",
    "Python test gate": "run: PYTHONPATH=workers/src python -m unittest discover -s workers/tests -v",
    "web build gate": "run: npm run build",
}

text = CI_PATH.read_text(encoding="utf-8")
missing = [name for name, snippet in REQUIRED_SNIPPETS.items() if snippet not in text]

if "continue-on-error: true" in text:
    raise SystemExit("quality-gate workflow must not contain continue-on-error: true")

if missing:
    raise SystemExit("missing required CI quality gates: " + ", ".join(sorted(missing)))

print("Release A CI quality-gate contract: PASS")
