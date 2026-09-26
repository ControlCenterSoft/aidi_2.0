package canonical

import (
	"errors"
	"fmt"
	"math"
)

type Revision uint64

var (
	ErrRevisionConflict  = errors.New("revision conflict")
	ErrRevisionExhausted = errors.New("revision exhausted")
)

type RevisionConflictError struct {
	Expected Revision
	Actual   Revision
}

func (e *RevisionConflictError) Error() string {
	return fmt.Sprintf("%s: expected=%d actual=%d", ErrRevisionConflict, e.Expected, e.Actual)
}

func (e *RevisionConflictError) Unwrap() error {
	return ErrRevisionConflict
}

func CheckExpectedRevision(expected, actual Revision) error {
	if expected != actual {
		return &RevisionConflictError{Expected: expected, Actual: actual}
	}
	return nil
}

func NextRevision(actual Revision) (Revision, error) {
	if actual == Revision(math.MaxUint64) {
		return 0, ErrRevisionExhausted
	}
	return actual + 1, nil
}
