package api

import (
	"context"
	"strings"
	"time"

	"backend_v3/domain/submission"

	"github.com/google/uuid"
)

// buildSubmissionDetailResponse creates a comprehensive SubmissionDetailResponse
// This helper avoids code duplication between GetSubmission, GetSubmissionAdmin, and UpdateSubmissionStatus
func buildSubmissionDetailResponse(
	sub *submission.Submission,
	h *Handler,
	ctx context.Context,
	subUUID uuid.UUID,
	submissionID string,
) SubmissionDetailResponse {
	// Convert UserID to string pointer
	var userIDStr *string
	if sub.UserID != nil {
		uid := sub.UserID.String()
		userIDStr = &uid
	}

	// Split BusinessChallenge back to strategic fields if needed
	strategicGoal := ""
	currentChallenges := ""
	competitivePosition := ""
	if sub.BusinessChallenge != "" {
		// Try to split by " | " separator
		parts := strings.Split(sub.BusinessChallenge, " | ")
		if len(parts) >= 3 {
			strategicGoal = parts[0]
			currentChallenges = parts[1]
			competitivePosition = parts[2]
		} else {
			currentChallenges = sub.BusinessChallenge
		}
	}

	resp := SubmissionDetailResponse{
		// IDs
		ID:     sub.ID.String(),
		UserID: userIDStr,

		// Company Information
		CompanyName: sub.CompanyName,
		CNPJ:        sub.CNPJ,
		Website:     sub.CompanyWebsite,
		Industry:    sub.CompanyIndustry,
		CompanySize: sub.CompanySize,

		// Strategic Context
		StrategicGoal:       stringToPtr(strategicGoal),
		CurrentChallenges:   stringToPtr(currentChallenges),
		CompetitivePosition: stringToPtr(competitivePosition),

		// Legacy fields
		ContactName:       stringToPtr(sub.ContactName),
		ContactEmail:      stringToPtr(sub.ContactEmail),
		ContactPhone:      sub.ContactPhone,
		ContactPosition:   sub.ContactPosition,
		TargetMarket:      sub.TargetMarket,
		AnnualRevenueMin:  sub.AnnualRevenueMin,
		AnnualRevenueMax:  sub.AnnualRevenueMax,
		FundingStage:      sub.FundingStage,
		BusinessChallenge: stringToPtr(sub.BusinessChallenge),
		AdditionalNotes:   sub.AdditionalNotes,
		LinkedInURL:       sub.LinkedInURL,
		TwitterHandle:     sub.TwitterHandle,
		Location:          sub.CompanyLocation,

		// Metadata
		Status:    string(sub.Status),
		CreatedAt: sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt: sub.UpdatedAt.Format(time.RFC3339),
	}

	// Add relationship IDs
	if enrich, err := h.enrichmentSvc.GetBySubmissionID(ctx, subUUID); err == nil {
		enrichID := enrich.ID.String()
		resp.EnrichmentID = &enrichID
	}
	if anal, err := h.analysisSvc.GetBySubmissionID(ctx, submissionID); err == nil {
		resp.AnalysisID = &anal.ID
	}
	if rep, err := h.reportSvc.GetBySubmissionID(ctx, submissionID); err == nil {
		resp.ReportID = &rep.ID
		resp.PDFURL = &rep.PDFURL
	}

	return resp
}
