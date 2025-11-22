package enrichment

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// JSONMap handles the JSONB interaction with Postgres
type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(JSONMap)
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, m)
}

// =================================================================
// NEW: 3-Layer Progressive Data Collection Structure
// Layer 1: Immediate (<2s) - Basic metadata
// Layer 2: Structured (3-6s) - Legal/business data
// Layer 3: AI Inference (6-10s) - Strategic intelligence
// =================================================================

// Layer1Data - Immediate data collection (<2s)
type Layer1Data struct {
	DomainMetadata DomainMetadata `json:"domainMetadata"`
	IPLocation     IPLocation     `json:"ipLocation"`
	BasicInfo      BasicInfo      `json:"basicInfo"`
	CollectedAt    string         `json:"collectedAt"`
	Sources        []string       `json:"sources"` // whois, metadata scraper, ip-api
}

type DomainMetadata struct {
	Domain      string   `json:"domain"`
	Registrar   string   `json:"registrar,omitempty"`
	CreatedDate string   `json:"createdDate,omitempty"`
	ExpiresDate string   `json:"expiresDate,omitempty"`
	DomainAge   int      `json:"domainAge,omitempty"` // in years
	NameServers []string `json:"nameServers,omitempty"`
}

type IPLocation struct {
	IP       string `json:"ip,omitempty"`
	Country  string `json:"country"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
	ISP      string `json:"isp,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type BasicInfo struct {
	Title       string   `json:"title,omitempty"`       // From meta tags
	Description string   `json:"description,omitempty"` // From meta tags
	Keywords    []string `json:"keywords,omitempty"`    // From meta tags
	Language    string   `json:"language,omitempty"`
}

// Layer2Data - Structured business data (3-6s)
type Layer2Data struct {
	LegalInfo    LegalInfo    `json:"legalInfo"`
	BusinessInfo BusinessInfo `json:"businessInfo"`
	LocationInfo LocationInfo `json:"locationInfo"`
	CollectedAt  string       `json:"collectedAt"`
	Sources      []string     `json:"sources"` // ReceitaWS, OpenCNPJ, Google Places
}

type LegalInfo struct {
	LegalName    string `json:"legalName"`
	CNPJ         string `json:"cnpj,omitempty"`
	TradeName    string `json:"tradeName,omitempty"`
	LegalStatus  string `json:"legalStatus,omitempty"`  // Active, Inactive, etc.
	LegalNature  string `json:"legalNature,omitempty"`  // SA, LTDA, etc.
	Founded      string `json:"founded,omitempty"`
	TaxSituation string `json:"taxSituation,omitempty"`
}

type BusinessInfo struct {
	Sector           string   `json:"sector"`                     // Primary CNAE
	SecondarySectors []string `json:"secondarySectors,omitempty"` // Secondary CNAEs
	EmployeeCount    string   `json:"employeeCount,omitempty"`    // Range or exact
	RevenueRange     string   `json:"revenueRange,omitempty"`     // Estimated
	CompanySize      string   `json:"companySize,omitempty"`      // MEI, ME, EPP, etc.
}

type LocationInfo struct {
	Address     string  `json:"address"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	ZipCode     string  `json:"zipCode,omitempty"`
	Phone       string  `json:"phone,omitempty"`
	Email       string  `json:"email,omitempty"`
	Coordinates *LatLng `json:"coordinates,omitempty"`
}

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Layer3Data - AI-powered strategic inference (6-10s)
type Layer3Data struct {
	BusinessSummary  BusinessSummary  `json:"businessSummary"`
	DigitalMaturity  DigitalMaturity  `json:"digitalMaturity"`
	CompetitiveIntel CompetitiveIntel `json:"competitiveIntel"`
	StrategicGaps    StrategicGaps    `json:"strategicGaps"`
	CollectedAt      string           `json:"collectedAt"`
	Sources          []string         `json:"sources"` // LLM analysis sources
}

type BusinessSummary struct {
	Description      string   `json:"description"`             // AI-generated business description
	ValueProposition string   `json:"valueProposition"`        // Core value prop
	TargetMarket     string   `json:"targetMarket"`            // Primary customer segment
	KeyProducts      []string `json:"keyProducts,omitempty"`   // Main offerings
	BrandTone        string   `json:"brandTone"`               // Professional, innovative, etc.
	UniqueFactors    []string `json:"uniqueFactors,omitempty"` // Differentiators
}

type DigitalMaturity struct {
	OverallScore       int      `json:"overallScore"`              // 1-10
	WebsiteQuality     int      `json:"websiteQuality"`            // 1-10
	SEOPresence        int      `json:"seoPresence"`               // 1-10
	SocialMedia        int      `json:"socialMedia"`               // 1-10
	OnlineReviews      int      `json:"onlineReviews"`             // 1-10
	EcommerceCapability int      `json:"ecommerceCapability"`       // 1-10
	Observations       []string `json:"observations,omitempty"`    // Key findings
}

type CompetitiveIntel struct {
	Industry       string   `json:"industry"`               // Detailed industry classification
	Competitors    []string `json:"competitors"`            // 3-5 main competitors
	MarketPosition string   `json:"marketPosition"`         // Leader, challenger, niche
	Differentiator string   `json:"keyDifferentiator"`      // Main competitive advantage
	ThreatLevel    string   `json:"threatLevel,omitempty"`  // High, medium, low
}

type StrategicGaps struct {
	OperationalGaps []string `json:"operationalGaps"`  // Process/capability gaps
	TechnologyGaps  []string `json:"technologyGaps"`   // Tech stack gaps
	MarketingGaps   []string `json:"marketingGaps"`    // Marketing/brand gaps
	TalentGaps      []string `json:"talentGaps,omitempty"` // HR/skill gaps
	Opportunities   []string `json:"opportunities"`    // Quick win opportunities
	PriorityActions []string `json:"priorityActions"`  // Recommended actions
}

// StrategicProfile - Combined view of all 3 layers
type StrategicProfile struct {
	Layer1 Layer1Data `json:"layer1"` // Immediate data
	Layer2 Layer2Data `json:"layer2"` // Structured data
	Layer3 Layer3Data `json:"layer3"` // AI inference

	// Metadata
	ProfileVersion  string `json:"profileVersion"`  // e.g., "3.0"
	TotalDuration   int    `json:"totalDuration"`   // milliseconds
	CompletedLayers int    `json:"completedLayers"` // 1, 2, or 3
}

// =================================================================
// Database Entity (Unchanged from Schema 004)
// =================================================================

type Enrichment struct {
	ID           uuid.UUID `json:"id" db:"id"`
	SubmissionID uuid.UUID `json:"submissionId" db:"submission_id"`

	Status      Status `json:"status" db:"status"`
	Progress    int    `json:"progress" db:"progress"`
	CurrentStep string `json:"currentStep" db:"current_step"`

	IsLocked bool `json:"isLocked" db:"is_locked"`

	// We map the StrategicProfile into this JSONMap for storage
	SourcesStatus JSONMap `json:"sourcesStatus,omitempty" db:"sources_status"`
	EnrichedData  JSONMap `json:"enrichedData,omitempty" db:"enriched_data"`

	StartedAt    *time.Time `json:"startedAt,omitempty" db:"started_at"`
	CompletedAt  *time.Time `json:"completedAt,omitempty" db:"completed_at"`
	ErrorMessage string     `json:"errorMessage,omitempty" db:"error_message"`
	RetryCount   int        `json:"retryCount" db:"retry_count"`
	MaxRetries   int        `json:"maxRetries" db:"max_retries"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusFinished   Status = "finished"
	StatusFailed     Status = "failed"
	StatusApproved   Status = "approved"
)

// Constructors and Methods (Start, UpdateProgress, etc.) remain identical...
// (Omitting strictly for brevity, paste previous standard methods here)
func NewEnrichment(submissionID uuid.UUID) *Enrichment {
	now := time.Now()
	return &Enrichment{
		ID:            uuid.New(),
		SubmissionID:  submissionID,
		Status:        StatusPending,
		Progress:      0,
		CurrentStep:   "Queued for enrichment",
		IsLocked:      false,
		SourcesStatus: make(JSONMap),
		EnrichedData:  make(JSONMap),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// ... Add Start(), UpdateProgress(), Complete(), Fail() from previous chat ...
func (e *Enrichment) Start() {
	e.Status = StatusProcessing
	n := time.Now()
	e.StartedAt = &n
	e.UpdatedAt = n
}

func (e *Enrichment) Finish() {
	e.Status = StatusFinished
	n := time.Now()
	e.CompletedAt = &n
	e.Progress = 100
	e.UpdatedAt = n
}

func (e *Enrichment) Approve() {
	e.Status = StatusApproved
	e.UpdatedAt = time.Now()
}

func (e *Enrichment) Fail(err error) {
	e.Status = StatusFailed
	e.ErrorMessage = err.Error()
	e.UpdatedAt = time.Now()
}

func (e *Enrichment) UpdateProgress(step string, pct int) {
	e.CurrentStep = step
	e.Progress = pct
	e.UpdatedAt = time.Now()
}
