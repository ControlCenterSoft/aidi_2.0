package integrity

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/canonical"
	"github.com/ControlCenterSoft/aidi_2.0/internal/repository"
)

func TestCheckerHealthyRepositoryObject(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemory()
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}
	if _, err := repo.Save(ctx, ref, 0, []byte("valid")); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	checker := NewChecker(repo, func(_ canonical.ObjectRef, payload []byte) error {
		if !bytes.Equal(payload, []byte("valid")) { return errors.New("unexpected payload") }
		return nil
	})
	report, err := checker.Check(ctx, ref)
	if err != nil { t.Fatalf("Check returned operational error: %v", err) }
	if !report.Healthy() { t.Fatalf("expected healthy report, got %+v", report.Violations) }
}

func TestCheckerDetectsInjectedFoundationViolations(t *testing.T) {
	ctx := context.Background()
	mismatch := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-mismatch"}
	invalidStored := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-invalid-stored"}
	missing := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-missing"}
	repo := &testRepository{objects: map[canonical.ObjectRef]repository.StoredObject{
		mismatch: {
			Ref: canonical.ObjectRef{Kind: canonical.KindProject, ID: "other-object"},
			Revision: 0,
			Payload: []byte("bad"),
		},
		invalidStored: {
			Ref: canonical.ObjectRef{Kind: canonical.Kind("not-a-kind"), ID: "bad"},
			Revision: 7,
			Payload: []byte("valid"),
		},
	}}
	checker := NewChecker(repo, func(_ canonical.ObjectRef, payload []byte) error {
		if bytes.Equal(payload, []byte("bad")) { return errors.New("payload invariant failure") }
		return nil
	})
	report, err := checker.Check(ctx, mismatch, invalidStored, missing)
	if err != nil { t.Fatalf("Check returned operational error: %v", err) }
	for _, code := range []ViolationCode{
		ViolationReferenceMismatch,
		ViolationZeroRevision,
		ViolationPayloadInvariant,
		ViolationInvalidStoredRef,
		ViolationMissingObject,
	} {
		if !report.Has(code) { t.Errorf("expected %q in %+v", code, report.Violations) }
	}
	for _, violation := range report.Violations {
		if !errors.Is(violation, ErrIntegrityViolation) {
			t.Errorf("%q does not unwrap to ErrIntegrityViolation", violation.Code)
		}
	}
}

func TestCheckerInvalidRequestedRefDoesNotReadRepository(t *testing.T) {
	repo := &testRepository{getErr: errors.New("Get must not be called")}
	report, err := NewChecker(repo, nil).Check(context.Background(), canonical.ObjectRef{Kind: canonical.KindTask})
	if err != nil { t.Fatalf("unexpected operational error: %v", err) }
	if !report.Has(ViolationInvalidRequestedRef) { t.Fatalf("missing invalid ref violation: %+v", report.Violations) }
	if repo.getCalls != 0 { t.Fatalf("Get called %d times", repo.getCalls) }
}

func TestCheckerPropagatesOperationalRepositoryFailure(t *testing.T) {
	boom := errors.New("repository unavailable")
	repo := &testRepository{getErr: boom}
	ref := canonical.ObjectRef{Kind: canonical.KindTask, ID: "task-1"}
	report, err := NewChecker(repo, nil).Check(context.Background(), ref)
	if !errors.Is(err, boom) { t.Fatalf("expected repository error, got %v", err) }
	if !report.Healthy() { t.Fatalf("operational failure misclassified: %+v", report.Violations) }
}

func TestCheckerWithoutRepositoryFailsClosed(t *testing.T) {
	report, err := NewChecker(nil, nil).Check(context.Background())
	if !errors.Is(err, ErrCheckerUnavailable) { t.Fatalf("expected ErrCheckerUnavailable, got %v", err) }
	if !report.Healthy() { t.Fatalf("unexpected violations: %+v", report.Violations) }
}

type testRepository struct {
	objects map[canonical.ObjectRef]repository.StoredObject
	getErr error
	getCalls int
}

func (r *testRepository) Get(_ context.Context, ref canonical.ObjectRef) (repository.StoredObject, error) {
	r.getCalls++
	if r.getErr != nil { return repository.StoredObject{}, r.getErr }
	if object, ok := r.objects[ref]; ok { return object, nil }
	return repository.StoredObject{}, repository.ErrNotFound
}

func (r *testRepository) Save(context.Context, canonical.ObjectRef, canonical.Revision, []byte) (canonical.Revision, error) {
	return 0, errors.New("test repository is read-only")
}

var _ repository.Repository = (*testRepository)(nil)
