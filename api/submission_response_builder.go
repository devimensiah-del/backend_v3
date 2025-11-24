package api

import (
	"context"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"
	"github.com/google/uuid"
)

type EnrichmentGetter interface {
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*enrichment.Enrichment, error)
}

type AnalysisGetter interface {
	GetBySubmissionID(ctx context.Context, submissionID string) (*analysis.Analysis, error)
}

type ReportGetter interface {
	GetBySubmissionID(ctx context.Context, submissionID string) (*report.Report, error)
}

// SubmissionResponseBuilder holds service dependencies for building submission detail responses.
type SubmissionResponseBuilder struct {
	EnrichmentService EnrichmentGetter
	AnalysisService   AnalysisGetter
	ReportService     ReportGetter
}

// NewSubmissionResponseBuilder creates a new SubmissionResponseBuilder.
func NewSubmissionResponseBuilder(
	enrichmentSvc EnrichmentGetter,
	analysisSvc AnalysisGetter,
	reportSvc ReportGetter,
) *SubmissionResponseBuilder {
	return &SubmissionResponseBuilder{
		EnrichmentService: enrichmentSvc,
		AnalysisService:   analysisSvc,
		ReportService:     reportSvc,
	}
}

// buildSubmissionDetailResponse creates a comprehensive SubmissionDetailResponse
// This helper avoids code duplication between GetSubmission, GetSubmissionAdmin, and UpdateSubmissionStatus
func (b *SubmissionResponseBuilder) buildSubmissionDetailResponse(
	sub *submission.Submission,
	ctx context.Context,
	submissionID string,
) SubmissionDetailResponse {
	// Convert UserID to string pointer
	var userIDStr *string
	if sub.UserID != nil {
		uid := sub.UserID.String()
		userIDStr = &uid
	}

	resp := SubmissionDetailResponse{
		ID:                sub.ID.String(),
		UserID:            userIDStr,
		CompanyName:       sub.CompanyName,
		CNPJ:              sub.CNPJ,
		CompanyWebsite:    sub.CompanyWebsite,
		CompanyIndustry:   sub.CompanyIndustry,
		CompanySize:       sub.CompanySize,
		CompanyLocation:   sub.CompanyLocation,
		ContactName:       sub.ContactName,
		ContactEmail:      sub.ContactEmail,
		ContactPhone:      sub.ContactPhone,
		ContactPosition:   sub.ContactPosition,
		TargetMarket:      sub.TargetMarket,
		AnnualRevenueMin:  sub.AnnualRevenueMin,
		AnnualRevenueMax:  sub.AnnualRevenueMax,
		FundingStage:      sub.FundingStage,
		BusinessChallenge: sub.BusinessChallenge,
		AdditionalNotes:   sub.AdditionalNotes,
		LinkedInURL:       sub.LinkedInURL,
		TwitterHandle:     sub.TwitterHandle,
		Status:            string(sub.Status),
		CreatedAt:         sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         sub.UpdatedAt.Format(time.RFC3339),
	}

	// Add relationship IDs (optional)
	if b.EnrichmentService != nil {
		if enrich, err := b.EnrichmentService.GetBySubmissionID(ctx, sub.ID); err == nil && enrich != nil {
			enrichID := enrich.ID.String()
			resp.EnrichmentID = &enrichID
		}
	}
	if b.AnalysisService != nil {
		if anal, err := b.AnalysisService.GetBySubmissionID(ctx, submissionID); err == nil && anal != nil {
			resp.AnalysisID = &anal.ID
		}
	}
	if b.ReportService != nil {
		if rep, err := b.ReportService.GetBySubmissionID(ctx, submissionID); err == nil && rep != nil {
			resp.ReportID = &rep.ID
			resp.PDFURL = &rep.PDFURL
		}
	}

	return resp
}
