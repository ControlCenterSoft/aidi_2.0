package integrity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
	"github.com/ControlCenterSoft/aidi_2.0/internal/lease"
)

var ErrLeaseDetectorUnavailable = errors.New("lease integrity detector unavailable")

type LeaseFindingCode string

const (
	LeaseFindingStale  LeaseFindingCode = "stale_lease"
	LeaseFindingOrphan LeaseFindingCode = "orphan_lease"
)

type LeaseFinding struct {
	Code       LeaseFindingCode
	Ref        canonical.ObjectRef
	Owner      string
	Generation uint64
	ExpiresAt  time.Time
}

type LeaseDetectionReport struct {
	Findings []LeaseFinding
}

func (r LeaseDetectionReport) Healthy() bool {
	return len(r.Findings) == 0
}

func (r LeaseDetectionReport) Has(code LeaseFindingCode) bool {
	for _, finding := range r.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

// LeaseReader is intentionally read-only. A LeaseDetector can observe persisted
// ownership but cannot Acquire, Renew, Release, or otherwise mutate lease state.
type LeaseReader interface {
	Get(ctx context.Context, ref canonical.ObjectRef) (lease.Lease, error)
}

// OwnerActiveFunc must return false only when the caller can authoritatively
// confirm that owner is not live. Unknown/unavailable liveness should be
// returned as an error so the detector does not manufacture orphan findings.
type OwnerActiveFunc func(ctx context.Context, owner string) (bool, error)

type LeaseDetector struct {
	reader      LeaseReader
	ownerActive OwnerActiveFunc
}

func NewLeaseDetector(reader LeaseReader, ownerActive OwnerActiveFunc) *LeaseDetector {
	return &LeaseDetector{reader: reader, ownerActive: ownerActive}
}

// Detect reports persisted lease states that need recovery attention without
// mutating them. A stale finding is a previously acquired lease that is no
// longer held at now. An orphan finding is a still-held lease whose owner is
// authoritatively reported as not live.
func (d *LeaseDetector) Detect(ctx context.Context, now time.Time, refs ...canonical.ObjectRef) (LeaseDetectionReport, error) {
	var report LeaseDetectionReport
	if d == nil || d.reader == nil || d.ownerActive == nil {
		return report, ErrLeaseDetectorUnavailable
	}

	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return report, fmt.Errorf("lease integrity check %s: %w", formatRef(ref), err)
		}

		observed, err := d.reader.Get(ctx, ref)
		if err != nil {
			return report, fmt.Errorf("lease integrity check %s: %w", formatRef(ref), err)
		}
		if err := observed.Validate(); err != nil {
			return report, fmt.Errorf("lease integrity check %s: %w", formatRef(ref), err)
		}
		if observed.Generation == 0 {
			continue
		}

		finding := LeaseFinding{
			Ref:        observed.Ref,
			Owner:      observed.Owner,
			Generation: observed.Generation,
			ExpiresAt:  observed.ExpiresAt,
		}
		if !observed.Held(now) {
			finding.Code = LeaseFindingStale
			report.Findings = append(report.Findings, finding)
			continue
		}

		active, err := d.ownerActive(ctx, observed.Owner)
		if err != nil {
			return report, fmt.Errorf("lease owner liveness %q: %w", observed.Owner, err)
		}
		if !active {
			finding.Code = LeaseFindingOrphan
			report.Findings = append(report.Findings, finding)
		}
	}

	return report, nil
}
