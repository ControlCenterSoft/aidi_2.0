# Build and version provenance

Every Release A executable exposes build provenance with:

```text
<binary> --version
```

The command writes one JSON object with four fields:

- `version` — AIDI release/build version;
- `commit` — source commit SHA;
- `build_time` — UTC build timestamp;
- `provenance` — reference to the build source/run that produced the binary.

The release Makefile accepts `RELEASE_VERSION`, `RELEASE_COMMIT`,
`RELEASE_BUILD_TIME`, and `RELEASE_PROVENANCE`. All four values are
injected through Go linker metadata into `aidi-control`,
`aidi-node-agent`, `aidi-installer`, and `aidi-admin`.

Metadata values are explicit inputs rather than implicit wall-clock state so
that repeated builds remain byte-for-byte reproducible when invoked with the
same provenance inputs.

GitHub CI builds every Foundation binary with commit, timestamp, and workflow
provenance, executes `--version`, validates every field, and publishes the
resulting binaries as qualification evidence.
