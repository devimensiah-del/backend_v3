package api

import (
	"context"
	"testing"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"

	"github.com/google/uuid"
)

type fakeEnrichmentService struct {
	item *enrichment.Enrichment
	err  error
}

func (f *fakeEnrichmentService) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*enrichment.Enrichment, error) {
	return f.item, f.err
}

type fakeAnalysisService struct {
	item *analysis.Analysis
	err  error
}

func (f *fakeAnalysisService) GetBySubmissionID(ctx context.Context, submissionID string) (*analysis.Analysis, error) {
	return f.item, f.err
}

type fakeReportService struct {
	item *report.Report
	err  error
}

func (f *fakeReportService) GetBySubmissionID(ctx context.Context, submissionID string) (*report.Report, error) {
	return f.item, f.err
}

func TestSubmissionResponseBuilder_BuildsContractShape(t *testing.T) {
	subID := uuid.New()
	userID := uuid.New()
	enrichID := uuid.New()
	now := time.Now().UTC()

	sub := &submission.Submission{
		ID:                subID,
		CompanyName:       "Test Co",
		CNPJ:              stringToPtr("12.345.678/0001-90"),
		CompanyWebsite:    stringToPtr("https://test.co"),
		CompanyIndustry:   stringToPtr("Tech"),
		CompanySize:       stringToPtr("10-50"),
		CompanyLocation:   stringToPtr("NYC"),
		ContactName:       "Alice",
		ContactEmail:      "alice@test.co",
		ContactPhone:      stringToPtr("+123"),
		ContactPosition:   stringToPtr("CEO"),
		TargetMarket:      stringToPtr("USA"),
		AnnualRevenueMin:  floatPtr(1_000_000),
		AnnualRevenueMax:  floatPtr(2_000_000),
		FundingStage:      stringToPtr("Seed"),
		BusinessChallenge: "Grow",
		AdditionalNotes:   stringToPtr("Notes"),
		LinkedInURL:       stringToPtr("https://linkedin.com/company/test"),
		TwitterHandle:     stringToPtr("@test"),
		Status:            submission.StatusReceived,
		UserID:            &userID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	builder := &SubmissionResponseBuilder{
		EnrichmentService: &fakeEnrichmentService{item: &enrichment.Enrichment{ID: enrichID}},
		AnalysisService:   &fakeAnalysisService{item: &analysis.Analysis{ID: "analysis-1"}},
		ReportService:     &fakeReportService{item: &report.Report{ID: "report-1", PDFURL: "http://example.com/report.pdf"}},
	}

	resp := builder.buildSubmissionDetailResponse(sub, context.Background(), subID.String())

	if resp.ID != subID.String() || resp.CompanyName != "Test Co" || resp.Status != string(submission.StatusReceived) {
		t.Fatalf("basic fields mismatch: %+v", resp)
	}
	if resp.EnrichmentID == nil || *resp.EnrichmentID != enrichID.String() {
		t.Fatalf("expected enrichment id, got %+v", resp.EnrichmentID)
	}
	if resp.AnalysisID == nil || *resp.AnalysisID != "analysis-1" {
		t.Fatalf("expected analysis id, got %+v", resp.AnalysisID)
	}
	if resp.ReportID == nil || *resp.ReportID != "report-1" || resp.PDFURL == nil {
		t.Fatalf("expected report id/pdf url, got report=%+v pdf=%+v", resp.ReportID, resp.PDFURL)
	}
	if resp.ContactName != "Alice" || resp.ContactEmail != "alice@test.co" {
		t.Fatalf("contact fields mismatch: %+v", resp)
	}
	if resp.CreatedAt == "" || resp.UpdatedAt == "" {
		t.Fatalf("expected timestamps set: %+v", resp)
	}
}

func floatPtr(v float64) *float64 { return &v }
