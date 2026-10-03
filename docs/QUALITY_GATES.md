# Release A quality gates

AIDI Release A changes are accepted only when the GitHub-hosted `AIDI CI` workflow is green.

The Foundation gate currently enforces:

- Go formatting with `gofmt`;
- Go module integrity with `go mod verify`;
- Go static analysis with `go vet ./...`;
- compilation of all Go packages with `go build ./...`;
- Go unit tests with the race detector;
- compilation of Python worker sources/tests;
- Python worker contract tests;
- the web application build from the pinned npm lockfile.

The workflow does not use `continue-on-error: true` for these checks. Any failing command makes its CI job fail, so the autonomous promotion path must not merge that revision.

The repository also runs `.github/scripts/test_ci_quality_gates.py` in CI to detect accidental removal or weakening of the minimum Foundation quality-gate contract.
