package api

import (
	"context"
	"testing"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/domain/submission"

	"github.com/google/uuid"
)

type fakeAnalysisService struct {
	item *analysis.Analysis
	err  error
}

func (f *fakeAnalysisService) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) (*analysis.Analysis, error) {
	return f.item, f.err
}

type fakeCompanyService struct {
	item *company.Company
	err  error
}

func (f *fakeCompanyService) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*company.Company, error) {
	return f.item, f.err
}

type fakeChallengeService struct {
	items []*challenge.Challenge
	err   error
}

func (f *fakeChallengeService) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*challenge.Challenge, error) {
	return f.items, f.err
}

func TestSubmissionResponseBuilder_BuildsContractShape(t *testing.T) {
	subID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	sub := &submission.Submission{
		ID:               subID,
		CompanyName:      "Test Co",
		CNPJ:             stringToPtr("12.345.678/0001-90"),
		CompanyWebsite:   stringToPtr("https://test.co"),
		CompanyIndustry:  stringToPtr("Tech"),
		CompanySize:      stringToPtr("10-50"),
		CompanyLocation:  stringToPtr("NYC"),
		ContactName:      "Alice",
		ContactEmail:     "alice@test.co",
		ContactPhone:     stringToPtr("+123"),
		ContactPosition:  stringToPtr("CEO"),
		TargetMarket:     stringToPtr("USA"),
		AnnualRevenueMin: floatPtr(1_000_000),
		AnnualRevenueMax: floatPtr(2_000_000),
		FundingStage:     stringToPtr("Seed"),
		AdditionalNotes:  stringToPtr("Notes"),
		LinkedInURL:      stringToPtr("https://linkedin.com/company/test"),
		TwitterHandle:    stringToPtr("@test"),
		UserID:           &userID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	companyID := uuid.New()
	challengeID := uuid.New()
	builder := &SubmissionResponseBuilder{
		AnalysisService:  &fakeAnalysisService{item: &analysis.Analysis{ID: "analysis-1", ChallengeID: challengeID, Status: "completed"}},
		CompanyService:   &fakeCompanyService{item: &company.Company{ID: companyID, EnrichmentStatus: "completed"}},
		ChallengeService: &fakeChallengeService{items: []*challenge.Challenge{{ID: challengeID, CompanyID: companyID, ChallengeCategory: "growth", ChallengeType: "growth_organic", BusinessChallenge: "Grow the business"}}},
	}

	resp := builder.buildSubmissionDetailResponse(sub, context.Background(), subID.String())

	if resp.ID != subID.String() || resp.CompanyName != "Test Co" {
		t.Fatalf("basic fields mismatch: %+v", resp)
	}
	// EnrichmentID removed - enrichment is now inline in company
	if resp.AnalysisID == nil || *resp.AnalysisID != "analysis-1" {
		t.Fatalf("expected analysis id, got %+v", resp.AnalysisID)
	}
	// PDF URL removed in v2_013 schema cleanup
	if resp.ContactName != "Alice" || resp.ContactEmail != "alice@test.co" {
		t.Fatalf("contact fields mismatch: %+v", resp)
	}
	if resp.CreatedAt == "" || resp.UpdatedAt == "" {
		t.Fatalf("expected timestamps set: %+v", resp)
	}
}

func floatPtr(v float64) *float64 { return &v }
