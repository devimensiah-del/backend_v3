package submission

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memRepo struct {
	last *Submission
}

func (m *memRepo) Create(ctx context.Context, s *Submission) error                { m.last = s; return nil }
func (m *memRepo) GetByID(ctx context.Context, id uuid.UUID) (*Submission, error) { return m.last, nil }
func (m *memRepo) GetByEmail(ctx context.Context, email string) ([]*Submission, error) {
	return nil, nil
}
func (m *memRepo) List(ctx context.Context, opts *ListOptions) ([]*Submission, int, error) {
	return nil, 0, nil
}
func (m *memRepo) Update(ctx context.Context, s *Submission) error    { m.last = s; return nil }
func (m *memRepo) Delete(ctx context.Context, id uuid.UUID) error     { return nil }
func (m *memRepo) GetTotalCount(ctx context.Context) (int, error)     { return 0, nil }
func (m *memRepo) GetActiveCount(ctx context.Context) (int, error)    { return 0, nil }
func (m *memRepo) GetCompletedCount(ctx context.Context) (int, error) { return 0, nil }
func (m *memRepo) ListWithAnalytics(ctx context.Context, opts *ListOptions) ([]*Submission, int, error) {
	return nil, 0, nil
}
func (m *memRepo) GetEnrichmentStatus(ctx context.Context, submissionID uuid.UUID) (*EnrichmentStatusRow, error) {
	return nil, nil
}
func (m *memRepo) ReserveEnrichment(ctx context.Context, submissionID uuid.UUID) (bool, error) {
	return true, nil
}
func (m *memRepo) GetAnonymousByEmail(ctx context.Context, email string) ([]*Submission, error) {
	return nil, nil
}
func (m *memRepo) UpdateUserID(ctx context.Context, submissionID, userID uuid.UUID) error {
	return nil
}

func TestCreateSetsReceivedStatus(t *testing.T) {
	repo := &memRepo{}
	svc := NewService(repo, nil)

	sub := &Submission{
		CompanyName:       "Test Co",
		ContactName:       "Alice",
		ContactEmail:      "alice@test.co",
		BusinessChallenge: "Grow",
		CreatedAt:         time.Time{},
		UpdatedAt:         time.Time{},
	}

	created, err := svc.Create(context.Background(), sub, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.Status != StatusReceived {
		t.Fatalf("expected status 'received', got %s", created.Status)
	}
	if created.ID == uuid.Nil {
		t.Fatalf("expected ID set")
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps set")
	}
}
