# State Integrity and Lease Anomaly Detection (Release A / A11-001, A11-002)

The Release A state-integrity recovery primitives are read-only detectors for
supported canonical-state and persisted-lease anomalies. The implementations
are in internal/integrity.

## State Integrity Checker (A11-001)

### Boundary

The checker reads canonical objects only through the A2-003
repository.Repository contract. It does not enumerate or connect to a database
directly and has no dependency on PostgreSQL drivers, files, message queues,
workflow engines, local AIDI, Forgejo, VMs, or runners.

Callers supply the exact canonical ObjectRef values that are expected to exist.
This keeps storage-specific discovery outside the checker boundary.

### Supported foundation checks

For each requested object the checker reports:

- a missing expected canonical object;
- an invalid requested canonical reference before repository access;
- an invalid stored canonical reference;
- a stored canonical identity that differs from the requested identity;
- revision 0 on a persisted object;
- a domain-specific payload invariant rejected by an optional PayloadValidator.

Supported violations accumulate in one Report. Repository operational errors
other than repository.ErrNotFound are returned separately and are not
misclassified as canonical-state corruption.

The checker is strictly read-only and never calls Repository.Save.

## Orphan/stale Lease Detection (A11-002)

LeaseDetector layers read-only recovery detection over the A7-003 persisted
Lease contract. It receives exact canonical ObjectRef values, reads each Lease
through a deliberately narrow LeaseReader interface exposing only Get, and uses
a caller-supplied OwnerActiveFunc to classify owner liveness.

Supported findings are:

- stale_lease: a previously acquired Lease (Generation > 0) that is no longer
  held at the observation time because ExpiresAt is not in the future;
- orphan_lease: a Lease that is still held at the observation time but whose
  persisted Owner is authoritatively reported as not live.

Owner liveness must be conservative. OwnerActiveFunc returns false only when its
source can confirm that the owner is not live. Unknown or unavailable liveness
is returned as an error and is not converted into an orphan finding.

Detection is intentionally non-destructive. LeaseDetector has no
Acquire/Renew/Release capability and does not delete, rewrite, expire, or
reconcile persisted lease data. A finding is evidence for a later recovery
decision, not permission to mutate state. In particular, stale_lease is a
read-only observation of persisted lease metadata whose ExpiresAt is at or
before the observation time; downstream recovery logic remains responsible for
distinguishing the appropriate remediation path.

## Evidence

internal/integrity/checker_test.go deliberately injects the supported A11-001
violations through a test repository and verifies stable violation codes. It
also verifies a healthy repository object, fail-closed checker configuration,
and separation of operational repository failures from state corruption.

internal/integrity/lease_detector_test.go verifies A11-002 detection of stale
and orphan leases, healthy live-owner and never-acquired cases, conservative
handling of unknown owner liveness, fail-closed dependency handling, and that
persisted Lease owner/generation/expiry data remains unchanged after detection.
