# State Integrity Checker (Release A / A11-001)

The Release A State Integrity Checker is a read-only recovery primitive for
detecting supported canonical-state invariant violations. The implementation is
in internal/integrity.

## Boundary

The checker reads canonical objects only through the A2-003
repository.Repository contract. It does not enumerate or connect to a database
directly and has no dependency on PostgreSQL drivers, files, message queues,
workflow engines, local AIDI, Forgejo, VMs, or runners.

Callers supply the exact canonical ObjectRef values that are expected to exist.
This keeps storage-specific discovery outside the checker boundary.

## Supported foundation checks

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

## Evidence

internal/integrity/checker_test.go deliberately injects the supported
violations through a test repository and verifies stable violation codes.
It also verifies a healthy repository object, fail-closed checker
configuration, and separation of operational repository failures from state
corruption.
