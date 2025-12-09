package adapters

import (
	"context"
	"encoding/json"

	"backend_v3/domain/analysis"
	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/domain/submission"

	"github.com/google/uuid"
)

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// convertViaJSON converts between structs with matching JSON field names.
// Uses JSON marshal/unmarshal to avoid manual field-by-field mapping.
func convertViaJSON(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// =============================================================================
// COMPANY SERVICE ADAPTERS
// =============================================================================

// CompanyServiceAdapterForSubmission implements submission.CompanyServiceInterface
type CompanyServiceAdapterForSubmission struct {
	Svc *company.Service
}

func NewCompanyServiceAdapterForSubmission(svc *company.Service) *CompanyServiceAdapterForSubmission {
	return &CompanyServiceAdapterForSubmission{Svc: svc}
}

func (a CompanyServiceAdapterForSubmission) CreateFromSubmission(ctx context.Context, input submission.CompanyCreateInput) (submission.CompanyResult, error) {
	var companyInput company.CreateFromSubmissionInput
	if err := convertViaJSON(input, &companyInput); err != nil {
		return submission.CompanyResult{}, err
	}
	comp, err := a.Svc.CreateFromSubmission(ctx, companyInput)
	if err != nil {
		return submission.CompanyResult{}, err
	}
	return submission.CompanyResult{
		ID:   comp.ID,
		Name: comp.Name,
	}, nil
}

func (a CompanyServiceAdapterForSubmission) SetOwnerFromSubmission(ctx context.Context, submissionID, userID uuid.UUID) error {
	return a.Svc.SetOwnerFromSubmission(ctx, submissionID, userID)
}

func (a CompanyServiceAdapterForSubmission) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	return a.Svc.Delete(ctx, id)
}

// CompanyServiceAdapterForAnalysis implements analysis.CompanyServiceInterface
type CompanyServiceAdapterForAnalysis struct {
	Svc *company.Service
}

func NewCompanyServiceAdapterForAnalysis(svc *company.Service) *CompanyServiceAdapterForAnalysis {
	return &CompanyServiceAdapterForAnalysis{Svc: svc}
}

// GetByID implements wizard.CompanyServiceInterface (v2: fetch by company ID directly)
func (a CompanyServiceAdapterForAnalysis) GetByID(ctx context.Context, companyID uuid.UUID) (*analysis.AnalysisCompanyData, error) {
	comp, err := a.Svc.GetByID(ctx, companyID)
	if err != nil || comp == nil {
		return nil, err
	}

	return convertCompanyToAnalysisData(comp), nil
}

// GetBySubmissionID still available for analysis.AnalysisCompanyServiceInterface
func (a CompanyServiceAdapterForAnalysis) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*analysis.AnalysisCompanyData, error) {
	comp, err := a.Svc.GetBySubmissionID(ctx, submissionID)
	if err != nil || comp == nil {
		return nil, err
	}

	return convertCompanyToAnalysisData(comp), nil
}

// convertCompanyToAnalysisData converts company.Company to analysis.AnalysisCompanyData
func convertCompanyToAnalysisData(comp *company.Company) *analysis.AnalysisCompanyData {
	result := &analysis.AnalysisCompanyData{
		Name:              comp.Name,
		CNPJ:              comp.CNPJ,
		Website:           comp.Website,
		Industry:          comp.Industry,
		Sector:            comp.Sector,
		CompanySize:       comp.CompanySize,
		Location:          comp.Location,
		TargetMarket:      comp.TargetMarket,
		FundingStage:      comp.FundingStage,
		FoundationYear:    comp.FoundationYear,
		Headquarters:      comp.Headquarters,
		TargetAudience:    comp.TargetAudience,
		ValueProposition:  comp.ValueProposition,
		EmployeesRange:    comp.EmployeesRange,
		RevenueEstimate:   comp.RevenueEstimate,
		BusinessModel:     comp.BusinessModel,
		MarketShareStatus: comp.MarketShareStatus,
		DigitalMaturity:   comp.DigitalMaturity,
		// Enrichment v3 scalar fields
		PricingModel:         comp.PricingModel,
		CompanyHistory:       comp.CompanyHistory,
		CompetitiveAdvantage: comp.CompetitiveAdvantage,
		MarketShare:          comp.MarketShare,
		TAMEstimate:          comp.TAMEstimate,
		SAMEstimate:          comp.SAMEstimate,
		SOMEstimate:          comp.SOMEstimate,
	}

	// Convert slices (original fields)
	if comp.Competitors != nil {
		result.Competitors = []string(comp.Competitors)
	}
	if comp.Strengths != nil {
		result.Strengths = []string(comp.Strengths)
	}
	if comp.Weaknesses != nil {
		result.Weaknesses = []string(comp.Weaknesses)
	}

	// Convert slices (v3 enrichment fields)
	if comp.MainProducts != nil {
		result.MainProducts = []string(comp.MainProducts)
	}
	if comp.CustomerSegments != nil {
		result.CustomerSegments = []string(comp.CustomerSegments)
	}
	if comp.UniqueSellingPoints != nil {
		result.UniqueSellingPoints = []string(comp.UniqueSellingPoints)
	}
	if comp.RecentNews != nil {
		result.RecentNews = []string(comp.RecentNews)
	}
	if comp.KeyExecutives != nil {
		result.KeyExecutives = []string(comp.KeyExecutives)
	}
	if comp.Opportunities != nil {
		result.Opportunities = []string(comp.Opportunities)
	}
	if comp.Threats != nil {
		result.Threats = []string(comp.Threats)
	}
	if comp.StrategicChallenges != nil {
		result.StrategicChallenges = []string(comp.StrategicChallenges)
	}
	if comp.CompetitorDetails != nil {
		result.CompetitorDetails = []string(comp.CompetitorDetails)
	}

	return result
}

// =============================================================================
// CHALLENGE SERVICE ADAPTER
// =============================================================================

// ChallengeServiceAdapterForSubmission implements submission.ChallengeServiceInterface
type ChallengeServiceAdapterForSubmission struct {
	Svc *challenge.Service
}

func NewChallengeServiceAdapterForSubmission(svc *challenge.Service) *ChallengeServiceAdapterForSubmission {
	return &ChallengeServiceAdapterForSubmission{Svc: svc}
}

func (a ChallengeServiceAdapterForSubmission) CreateFromInput(ctx context.Context, input submission.ChallengeCreateInput) (uuid.UUID, error) {
	// Create challenge entity from input (no contact info - stays on submission)
	ch := challenge.NewChallenge(
		input.CompanyID,
		challenge.ChallengeCategory(input.ChallengeCategory),
		challenge.ChallengeType(input.ChallengeType),
		input.BusinessChallenge,
	)

	created, err := a.Svc.Create(ctx, ch)
	if err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

func (a ChallengeServiceAdapterForSubmission) DeleteChallenge(ctx context.Context, id uuid.UUID) error {
	// Delegate to the challenge service's existing Delete method
	return a.Svc.Delete(ctx, id)
}

func (a ChallengeServiceAdapterForSubmission) ValidateCategory(category string) bool {
	return a.Svc.ValidateCategory(category)
}

func (a ChallengeServiceAdapterForSubmission) ValidateType(category, challengeType string) bool {
	return a.Svc.ValidateType(category, challengeType)
}

// =============================================================================
// CHALLENGE REPOSITORY ADAPTER
// =============================================================================

// ChallengeRepositoryAdapterForAnalysis adapts challenge.Repository to analysis.ChallengeRepository
type ChallengeRepositoryAdapterForAnalysis struct {
	Repo challenge.Repository
}

func NewChallengeRepositoryAdapterForAnalysis(repo challenge.Repository) *ChallengeRepositoryAdapterForAnalysis {
	return &ChallengeRepositoryAdapterForAnalysis{Repo: repo}
}

// GetByID implements analysis.ChallengeRepository
func (a ChallengeRepositoryAdapterForAnalysis) GetByID(ctx context.Context, id uuid.UUID) (interface{}, error) {
	return a.Repo.GetByID(ctx, id)
}
