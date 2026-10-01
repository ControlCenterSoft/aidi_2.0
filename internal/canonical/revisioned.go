package canonical

// Revisioned pairs a mutable canonical object's current payload (Value)
// with its current Revision, per SPEC §4.2/§8.2 and FOUNDATION.md
// "Optimistic concurrency": object revisions are mandatory and
// last-write-wins is forbidden for critical state. It is the generic
// "object revision model" built purely on the Revision/
// CheckExpectedRevision/NextRevision primitives (revision.go), mirroring
// the Rule[T]/RuleSet[T] generic-extraction convention (invariant.go):
// no existing entity (Task, ChangeSet, specification.Requirement) is
// rewired onto Revisioned[T] in this slice.
type Revisioned[T any] struct {
	// Revision is the current monotonic revision of Value. It starts at 1
	// (NewRevisioned) and strictly increases by exactly one on every
	// successful Update, mirroring the "Version >= 1" contract already
	// enforced for specification.Requirement.
	Revision Revision
	// Value is the current payload: the mutable canonical object itself.
	Value T
}

// NewRevisioned wraps value at the initial Revision (1), the first
// committed revision of a newly created canonical object.
func NewRevisioned[T any](value T) Revisioned[T] {
	return Revisioned[T]{Revision: 1, Value: value}
}

// Update applies mutate to the current Value and returns the resulting
// Revisioned[T] at the next monotonic Revision, but only if expected
// matches the receiver's current Revision.
//
// Contract (SPEC §4.2 optimistic concurrency / FOUNDATION.md "Optimistic
// concurrency"):
//   - A stale expected Revision is rejected via CheckExpectedRevision,
//     returning a *RevisionConflictError (wrapping ErrRevisionConflict)
//     naming both the expected and actual Revision; mutate is never
//     invoked and the receiver is left unchanged. Last-write-wins is not
//     reachable through this method.
//   - On a matching expected Revision, mutate is invoked with the current
//     Value. If mutate returns an error, Update returns the zero
//     Revisioned[T] and that error unchanged; the receiver is left
//     unchanged (Revisioned[T] is a value type) and the Revision is not
//     advanced.
//   - If mutate succeeds, the Revision advances by exactly one
//     (NextRevision); monotonic exhaustion (ErrRevisionExhausted) is
//     surfaced instead of wrapping back to a lower/zero Revision.
func (r Revisioned[T]) Update(expected Revision, mutate func(T) (T, error)) (Revisioned[T], error) {
	if err := CheckExpectedRevision(expected, r.Revision); err != nil {
		return Revisioned[T]{}, err
	}
	next, err := NextRevision(r.Revision)
	if err != nil {
		return Revisioned[T]{}, err
	}
	updated, err := mutate(r.Value)
	if err != nil {
		return Revisioned[T]{}, err
	}
	return Revisioned[T]{Revision: next, Value: updated}, nil
}
