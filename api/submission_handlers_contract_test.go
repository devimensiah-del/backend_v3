package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// --- fakes for submission handler contract test ---

type fakeSubmissionRepo struct{ sub *submission.Submission }

func (f *fakeSubmissionRepo) Create(ctx context.Context, s *submission.Submission) error {
	panic("not used")
}
func (f *fakeSubmissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*submission.Submission, error) {
	return f.sub, nil
}
func (f *fakeSubmissionRepo) GetByEmail(ctx context.Context, email string) ([]*submission.Submission, error) {
	panic("not used")
}
func (f *fakeSubmissionRepo) List(ctx context.Context, opts *submission.ListOptions) ([]*submission.Submission, int, error) {
	panic("not used")
}
func (f *fakeSubmissionRepo) Update(ctx context.Context, s *submission.Submission) error {
	panic("not used")
}
func (f *fakeSubmissionRepo) Delete(ctx context.Context, id uuid.UUID) error     { panic("not used") }
func (f *fakeSubmissionRepo) GetTotalCount(ctx context.Context) (int, error)     { panic("not used") }
func (f *fakeSubmissionRepo) GetActiveCount(ctx context.Context) (int, error)    { panic("not used") }
func (f *fakeSubmissionRepo) GetCompletedCount(ctx context.Context) (int, error) { panic("not used") }

type noopEnrichmentGetter struct{}

func (noopEnrichmentGetter) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*enrichment.Enrichment, error) {
	return nil, nil
}

type noopAnalysisGetter struct{}

func (noopAnalysisGetter) GetBySubmissionID(ctx context.Context, submissionID string) (*analysis.Analysis, error) {
	return nil, nil
}

type noopReportGetter struct{}

func (noopReportGetter) GetBySubmissionID(ctx context.Context, submissionID string) (*report.Report, error) {
	return nil, nil
}

// --- test ---

func TestGetSubmissionContractShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	subID := uuid.New()
	now := time.Now().UTC()

	sub := &submission.Submission{
		ID:              subID,
		CompanyName:     "Test Co",
		CNPJ:            stringToPtr("12.345.678/0001-90"),
		CompanyWebsite:  stringToPtr("https://test.co"),
		CompanyIndustry: stringToPtr("Tech"),
		CompanySize:     stringToPtr("10-50"),
		CompanyLocation: stringToPtr("NYC"),
		ContactName:     "Alice",
		ContactEmail:    "alice@test.co",
		Status:          submission.StatusReceived,
		UserID:          &userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	repo := &fakeSubmissionRepo{sub: sub}
	subSvc := submission.NewService(repo, nil)

	builder := NewSubmissionResponseBuilder(
		noopEnrichmentGetter{}, noopAnalysisGetter{}, noopReportGetter{},
	)

	handler := &SubmissionHandlers{
		SubmissionService:         subSvc,
		EnrichmentService:         nil,
		AsynqClient:               nil,
		Logger:                    zerolog.Nop(),
		SubmissionResponseBuilder: builder,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/"+subID.String(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", userID.String())
	c.Set("userRole", "user")
	c.Params = gin.Params{{Key: "id", Value: subID.String()}}

	handler.GetSubmission(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	var submissionResp SubmissionDetailResponse
	if err := json.Unmarshal(payload["submission"], &submissionResp); err != nil {
		t.Fatalf("invalid submission payload: %v", err)
	}
	if submissionResp.ID != subID.String() || submissionResp.CompanyName != "Test Co" || submissionResp.Status != string(submission.StatusReceived) {
		t.Fatalf("unexpected submission payload: %+v", submissionResp)
	}
	if submissionResp.ContactName != "Alice" || submissionResp.ContactEmail != "alice@test.co" {
		t.Fatalf("contact fields mismatch: %+v", submissionResp)
	}
	if submissionResp.CreatedAt == "" || submissionResp.UpdatedAt == "" {
		t.Fatalf("timestamps missing: %+v", submissionResp)
	}
}
