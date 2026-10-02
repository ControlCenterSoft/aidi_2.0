package integrity

import (
	"context"
	"errors"
	"fmt"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
	"github.com/ControlCenterSoft/aidi_2.0/internal/repository"
)

var ErrIntegrityViolation = errors.New("state integrity violation")
var ErrCheckerUnavailable = errors.New("state integrity checker unavailable")

type ViolationCode string

const (
	ViolationMissingObject       ViolationCode = "missing_object"
	ViolationInvalidRequestedRef ViolationCode = "invalid_requested_ref"
	ViolationInvalidStoredRef    ViolationCode = "invalid_stored_ref"
	ViolationReferenceMismatch   ViolationCode = "reference_mismatch"
	ViolationZeroRevision        ViolationCode = "zero_revision"
	ViolationPayloadInvariant    ViolationCode = "payload_invariant"
)

type Violation struct {
	Code         ViolationCode
	RequestedRef canonical.ObjectRef
	StoredRef    canonical.ObjectRef
	Revision     canonical.Revision
	Cause        error
}

func (v Violation) Error() string {
	return fmt.Sprintf("%v: %s requested=%s stored=%s revision=%d: %v", ErrIntegrityViolation, v.Code, formatRef(v.RequestedRef), formatRef(v.StoredRef), v.Revision, v.Cause)
}

func (v Violation) Unwrap() error {
	return ErrIntegrityViolation
}

type Report struct {
	Violations []Violation
}

func (r Report) Healthy() bool {
	return len(r.Violations) == 0
}

func (r Report) Has(code ViolationCode) bool {
	for _, violation := range r.Violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}

type PayloadValidator func(ref canonical.ObjectRef, payload []byte) error

type Checker struct {
	repo             repository.Repository
	payloadValidator PayloadValidator
}

func NewChecker(repo repository.Repository, payloadValidator PayloadValidator) *Checker {
	return &Checker{repo: repo, payloadValidator: payloadValidator}
}

func (c *Checker) Check(ctx context.Context, refs ...canonical.ObjectRef) (Report, error) {
	var report Report
	if c == nil || c.repo == nil {
		return report, ErrCheckerUnavailable
	}
	for _, requested := range refs {
		if err := requested.Validate(); err != nil {
			report.Violations = append(report.Violations, Violation{Code: ViolationInvalidRequestedRef, RequestedRef: requested, Cause: err})
			continue
		}
		stored, err := c.repo.Get(ctx, requested)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				report.Violations = append(report.Violations, Violation{Code: ViolationMissingObject, RequestedRef: requested, Cause: err})
				continue
			}
			return report, fmt.Errorf("state integrity check %s: %w", formatRef(requested), err)
		}
		if err := stored.Ref.Validate(); err != nil {
			report.Violations = append(report.Violations, Violation{Code: ViolationInvalidStoredRef, RequestedRef: requested, StoredRef: stored.Ref, Revision: stored.Revision, Cause: err})
		} else if stored.Ref != requested {
			report.Violations = append(report.Violations, Violation{Code: ViolationReferenceMismatch, RequestedRef: requested, StoredRef: stored.Ref, Revision: stored.Revision, Cause: errors.New("repository returned different canonical identity")})
		}
		if stored.Revision == 0 {
			report.Violations = append(report.Violations, Violation{Code: ViolationZeroRevision, RequestedRef: requested, StoredRef: stored.Ref})
		}
		if c.payloadValidator != nil {
			if err := c.payloadValidator(stored.Ref, stored.Payload); err != nil {
				report.Violations = append(report.Violations, Violation{Code: ViolationPayloadInvariant, RequestedRef: requested, StoredRef: stored.Ref, Revision: stored.Revision, Cause: err})
			}
		}
	}
	return report, nil
}

func formatRef(ref canonical.ObjectRef) string {
	return fmt.Sprintf("%s/%s", ref.Kind, ref.ID)
}
