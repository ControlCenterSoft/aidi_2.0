# Release A canonical backlog contract

This directory is the GitHub execution projection of the approved AIDI v2.0.0 Release A tracker manifest.

## Authority and provenance

- Source artifact: `AIDI-2.0.0-Current-Project-Import-Bundle-20260925.zip`
- Source member: `release_a_tracker_package/aidi-release-a-gitea-manifest.json`
- Source SHA-256: `91f720101294c36b8243cbead1c064d67f8d27f33dd52386f256a9102c8393aa`
- Approved scope: 86 implementation cards + 15 gate cards = 101 cards.
- Approved hard-dependency edges: 263.
- Unresolved dependencies and dependency cycles in the approved manifest: zero.

`manifest.json` records provenance and policy. `cards.tsv` is the exact stable-key/dependency execution index used by GitHub automation.

## Execution invariants

1. `.github/workflows/autonomous-core.yml` MUST NOT invent Release A scope or create a replacement implementation Issue.
2. Product work may start only from a canonical GitHub Issue whose title exactly matches a stable key/title in `cards.tsv`.
3. An implementation card is READY only when every hard dependency is DONE (its canonical GitHub Issue is closed).
4. Gate cards are never product-write work and MUST NOT be selected by Autonomous Core.
5. Missing keys, duplicate keys, unknown dependencies, cycles, title drift, incomplete tracker import, or a non-canonical unresolved legacy cycle are fail-closed conditions.
6. `import.complete` is an activation fence. It may exist only after all 101 canonical GitHub Issues are present exactly once and registry validation passes.
7. Release A is not complete merely because all implementation cards are closed. All 15 gate cards must independently reach PASS/DONE with required evidence.

## Change control

The approved card graph may change only through a versioned SPEC/backlog change with a new source artifact/hash and an explicit migration. Editing `cards.tsv` ad hoc is prohibited.
