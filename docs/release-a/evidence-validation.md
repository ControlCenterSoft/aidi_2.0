# Release A evidence validation contract

Release A verification evidence is traceable to an exact requirement, source revision, artifact, validator and production timestamp.

The deterministic validator in `internal/verification` enforces the current Foundation-level envelope:

- requirement identifier is mandatory;
- source revision is an exact 40-character hexadecimal Git commit;
- artifact paths are repository-relative and cannot traverse to a parent directory;
- artifact SHA-256 is a complete hexadecimal digest;
- validator identity and production timestamp are mandatory;
- artifact bytes can be verified against the recorded SHA-256.

This contract supports later Acceptance Matrix and release-gate evidence. It does not by itself mark a product-level acceptance criterion as PASS.
