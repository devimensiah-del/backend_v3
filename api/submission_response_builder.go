package api

import (
	"context"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/domain/submission"

	"github.com/google/uuid"
)

// AnalysisGetter retrieves analysis by challenge ID
type AnalysisGetter interface {
	GetByChallengeID(ctx context.Context, challengeID uuid.UUID) (*analysis.Analysis, error)
}

// CompanyGetter retrieves company by submission ID
type CompanyGetter interface {
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*company.Company, error)
}

// ChallengeGetter retrieves challenges by company ID
type ChallengeGetter interface {
	ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*challenge.Challenge, error)
}

// SubmissionResponseBuilder holds service dependencies for building submission detail responses.
type SubmissionResponseBuilder struct {
	AnalysisService  AnalysisGetter
	CompanyService   CompanyGetter
	ChallengeService ChallengeGetter
}

// NewSubmissionResponseBuilder creates a new SubmissionResponseBuilder.
func NewSubmissionResponseBuilder(
	analysisSvc AnalysisGetter,
	companySvc CompanyGetter,
	challengeSvc ChallengeGetter,
) *SubmissionResponseBuilder {
	return &SubmissionResponseBuilder{
		AnalysisService:  analysisSvc,
		CompanyService:   companySvc,
		ChallengeService: challengeSvc,
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
		ID:               sub.ID.String(),
		UserID:           userIDStr,
		CompanyName:      sub.CompanyName,
		CNPJ:             sub.CNPJ,
		CompanyWebsite:   sub.CompanyWebsite,
		CompanyIndustry:  sub.CompanyIndustry,
		CompanySize:      sub.CompanySize,
		CompanyLocation:  sub.CompanyLocation,
		ContactName:      sub.ContactName,
		ContactEmail:     sub.ContactEmail,
		ContactPhone:     sub.ContactPhone,
		ContactPosition:  sub.ContactPosition,
		TargetMarket:     sub.TargetMarket,
		AnnualRevenueMin: sub.AnnualRevenueMin,
		AnnualRevenueMax: sub.AnnualRevenueMax,
		FundingStage:     sub.FundingStage,
		AdditionalNotes:  sub.AdditionalNotes,
		LinkedInURL:      sub.LinkedInURL,
		TwitterHandle:    sub.TwitterHandle,
		CreatedAt:        sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        sub.UpdatedAt.Format(time.RFC3339),
		Status:           "pending", // Default status
	}

	// Fetch company data and derive status
	var companyData *company.Company
	if b.CompanyService != nil {
		if comp, err := b.CompanyService.GetBySubmissionID(ctx, sub.ID); err == nil && comp != nil {
			companyData = comp
			companyIDStr := comp.ID.String()
			resp.CompanyID = &companyIDStr

			// Derive initial status from enrichment
			resp.Status = deriveStatus(comp.EnrichmentStatus, "")
		}
	}

	// Fetch challenge data (get first/primary challenge for the company)
	if companyData != nil && b.ChallengeService != nil {
		if challenges, err := b.ChallengeService.ListByCompany(ctx, companyData.ID); err == nil && len(challenges) > 0 {
			// Use the first (most recent) challenge
			ch := challenges[0]
			challengeIDStr := ch.ID.String()
			resp.ChallengeID = &challengeIDStr

			category := string(ch.ChallengeCategory)
			challengeType := string(ch.ChallengeType)
			resp.ChallengeCategory = &category
			resp.ChallengeType = &challengeType
			resp.BusinessChallenge = &ch.BusinessChallenge
		}
	}

	// Fetch analysis data and update status (via challenge ID)
	if b.AnalysisService != nil && resp.ChallengeID != nil {
		challengeUUID, parseErr := uuid.Parse(*resp.ChallengeID)
		if parseErr == nil {
			if anal, err := b.AnalysisService.GetByChallengeID(ctx, challengeUUID); err == nil && anal != nil {
				resp.AnalysisID = &anal.ID

				// Update status based on analysis
				if companyData != nil {
					resp.Status = deriveStatus(companyData.EnrichmentStatus, anal.Status)
				} else {
					resp.Status = deriveStatus("", anal.Status)
				}
			}
		}
	}

	return resp
}

// deriveStatus derives a user-friendly status from enrichment and analysis statuses
func deriveStatus(enrichmentStatus, analysisStatus string) string {
	// Priority: analysis status > enrichment status
	if analysisStatus != "" {
		switch analysisStatus {
		case "completed", "approved", "sent":
			return "completed"
		case "processing":
			return "analyzing"
		case "failed":
			return "failed"
		case "pending":
			// Fall through to check enrichment
		}
	}

	// Check enrichment status
	switch enrichmentStatus {
	case "completed":
		if analysisStatus == "" || analysisStatus == "pending" {
			return "enriched" // Enrichment done, analysis pending
		}
		return "analyzing"
	case "processing":
		return "enriching"
	case "failed":
		return "failed"
	case "pending", "":
		return "pending"
	}

	return "pending"
}
