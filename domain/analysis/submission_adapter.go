package analysis

import (
	"backend_v3/domain/submission"
	"context"

	"github.com/google/uuid"
)

// SubmissionRepositoryAdapter adapts the submission.Repository to the analysis.SubmissionRepository interface
type SubmissionRepositoryAdapter struct {
	repo submission.Repository
}

// NewSubmissionRepositoryAdapter creates a new adapter
func NewSubmissionRepositoryAdapter(repo submission.Repository) SubmissionRepository {
	return &SubmissionRepositoryAdapter{repo: repo}
}

// GetByID fetches submission and converts it to SubmissionData needed by analysis
func (a *SubmissionRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*SubmissionData, error) {
	sub, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert submission.Submission to analysis.SubmissionData
	return &SubmissionData{
		CompanyName:       sub.CompanyName,
		CompanyWebsite:    sub.CompanyWebsite,
		CompanyIndustry:   sub.CompanyIndustry,
		CompanySize:       sub.CompanySize,
		CompanyLocation:   sub.CompanyLocation,
		BusinessChallenge: sub.BusinessChallenge,
		TargetMarket:      sub.TargetMarket,
		AnnualRevenueMin:  sub.AnnualRevenueMin,
		AnnualRevenueMax:  sub.AnnualRevenueMax,
		FundingStage:      sub.FundingStage,
	}, nil
}
