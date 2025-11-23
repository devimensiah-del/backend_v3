package testutils

import (
	"encoding/json"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"

	"github.com/google/uuid"
)

// =================================================================
// Fixture Data Generators
// =================================================================

// NewTestSubmission creates a valid submission for testing
func NewTestSubmission() *submission.Submission {
	now := time.Now()
	website := "https://acme-tech.com.br"
	industry := "Technology"
	size := "50-200"
	location := "São Paulo, SP, Brazil"
	phone := "+55 11 98765-4321"
	position := "CTO"
	targetMarket := "B2B SaaS - Enterprise"
	revenueMin := 5000000.0
	revenueMax := 15000000.0
	fundingStage := "Series A"
	notes := "Looking to expand to Latin American markets"
	linkedin := "https://linkedin.com/company/acme-tech"
	twitter := "@acmetech"
	cnpj := "12.345.678/0001-90"

	return &submission.Submission{
		ID:                uuid.New(),
		CompanyName:       "Acme Tech Solutions",
		CNPJ:              &cnpj,
		CompanyWebsite:    &website,
		CompanyIndustry:   &industry,
		CompanySize:       &size,
		CompanyLocation:   &location,
		ContactName:       "John Silva",
		ContactEmail:      "john.silva@acme-tech.com.br",
		ContactPhone:      &phone,
		ContactPosition:   &position,
		TargetMarket:      &targetMarket,
		AnnualRevenueMin:  &revenueMin,
		AnnualRevenueMax:  &revenueMax,
		FundingStage:      &fundingStage,
		BusinessChallenge: "We need to scale our cloud infrastructure platform to support 10x user growth while maintaining 99.9% uptime. Current challenges include database scalability and multi-region deployment complexity.",
		AdditionalNotes:   &notes,
		LinkedInURL:       &linkedin,
		TwitterHandle:     &twitter,
		Status:            submission.StatusReceived,
		UserID:            nil, // Public submission
		CreatedAt:         now,
		UpdatedAt:         now,
		DeletedAt:         nil,
	}
}

// NewTestEnrichment creates an enrichment with UnifiedProfile data
func NewTestEnrichment(submissionID uuid.UUID) *enrichment.Enrichment {
	now := time.Now()
	profile := enrichment.UnifiedProfile{
		ProfileOverview: enrichment.ProfileOverview{
			LegalName:      "Acme Tech Solutions LTDA",
			Website:        "https://acme-tech.com.br",
			FoundationYear: "2019",
			Headquarters:   "São Paulo, SP, Brazil",
		},
		MarketPosition: enrichment.MarketPosition{
			Sector:           "Cloud Infrastructure & DevOps",
			TargetAudience:   "Enterprise B2B - Tech companies with 100+ developers",
			ValueProposition: "Kubernetes-native platform with 99.9% uptime SLA and auto-scaling",
		},
		Financials: enrichment.Financials{
			EmployeesRange:  "50-200",
			RevenueEstimate: "R$ 8-12M ARR (2024)",
			BusinessModel:   "Subscription SaaS - Monthly recurring revenue",
		},
		CompetitiveLandscape: enrichment.CompetitiveLandscape{
			Competitors: []string{
				"AWS Elastic Kubernetes Service",
				"Google Cloud Platform GKE",
				"Azure Kubernetes Service",
				"DigitalOcean Kubernetes",
			},
			MarketShareStatus: "Emerging player with <2% market share in Brazil",
		},
		StrategicAssessment: enrichment.StrategicAssessment{
			DigitalMaturity: 8,
			Strengths: []string{
				"Strong technical team with Kubernetes expertise",
				"Excellent product-market fit in Brazilian market",
				"High customer retention (92% annual)",
				"Fast deployment times (avg 5 minutes)",
			},
			Weaknesses: []string{
				"Limited brand recognition vs global players",
				"Small sales and marketing team",
				"Dependency on single market (Brazil)",
				"Limited multi-cloud support",
			},
		},
		MacroContext: &enrichment.MacroContext{
			EconomicIndicators: enrichment.EconomicIndicators{
				Country:             "Brazil",
				GDPGrowth:           "+2.3% (2025 forecast)",
				InflationRate:       "IPCA: 4.5% a.a.",
				InterestRate:        "SELIC: 11.25%",
				ExchangeRate:        "USD/BRL: 5.15",
				UnemploymentRate:    "7.8%",
				PoliticalStability:  "Moderate - ongoing fiscal reforms",
				EconomicOutlook:     "Cautious growth with focus on tech sector investment",
				RecentPolicyChanges: []string{"Tax reform 2025", "Digital services taxation", "Cloud sovereignty regulations"},
			},
			IndustryTrends: enrichment.IndustryTrends{
				IndustrySector:      "Cloud Computing & Infrastructure",
				GrowthRate:          "+22% CAGR (2024-2028)",
				KeyTrends:           []string{"Kubernetes adoption surge", "Multi-cloud strategies", "Edge computing", "FinOps practices"},
				TechnologyAdoption:  "High - 68% of enterprises adopting cloud-native",
				MarketConcentration: "Concentrated - AWS 32%, Azure 21%, GCP 9%",
				BarriersToEntry:     "High - requires significant capital and technical expertise",
				MergersAcquisitions: []string{"AWS acquired CloudEndure (2024)", "Google Cloud expanded Brazil presence"},
			},
			RegulatoryLandscape: enrichment.RegulatoryLandscape{
				RecentRegulations:      []string{"LGPD enforcement strengthened", "Cloud data residency requirements"},
				UpcomingChanges:        []string{"Carbon tax proposal", "AI governance framework"},
				ComplianceRequirements: "LGPD, ISO 27001, SOC 2 Type II",
				IndustryStandards:      []string{"ISO 27001", "SOC 2", "PCI-DSS"},
			},
			MarketSignals: enrichment.MarketSignals{
				CommodityPrices:    []string{"Cloud compute costs stable", "Bandwidth costs down 8% YoY"},
				SupplyChainStatus:  "Normal - no major disruptions",
				ConsumerSentiment:  "Cautiously optimistic about tech sector",
				CompetitorActivity: []string{"AWS opened new Brazil region", "Azure expanding São Paulo datacenter"},
				EmergingThreats:    []string{"Aggressive pricing from Chinese cloud providers", "Open-source Kubernetes alternatives"},
			},
			DataSources: []string{"web_search", "industry_reports", "competitor_analysis"},
			LastUpdated: "2025-01-22",
		},
	}

	profileBytes, _ := json.Marshal(profile)
	enrichedData := make(enrichment.JSONMap)
	json.Unmarshal(profileBytes, &enrichedData)

	sourcesStatus := make(enrichment.JSONMap)
	sourcesStatus["web_scraping"] = "completed"
	sourcesStatus["linkedin"] = "completed"
	sourcesStatus["crunchbase"] = "completed"

	completedAt := now.Add(45 * time.Second)

	return &enrichment.Enrichment{
		ID:            uuid.New(),
		SubmissionID:  submissionID,
		Status:        enrichment.StatusCompleted,
		Progress:      100,
		CurrentStep:   "Enrichment completed",
		IsLocked:      false,
		SourcesStatus: sourcesStatus,
		EnrichedData:  enrichedData,
		StartedAt:     &now,
		CompletedAt:   &completedAt,
		ErrorMessage:  "",
		RetryCount:    0,
		MaxRetries:    3,
		CreatedAt:     now,
		UpdatedAt:     completedAt,
	}
}

// NewTestAnalysis creates an analysis with all 11 frameworks populated
func NewTestAnalysis(submissionID, enrichmentID string) *analysis.Analysis {
	now := time.Now()
	completedAt := now.Add(8 * time.Second)

	return &analysis.Analysis{
		ID:               uuid.New().String(),
		SubmissionID:     submissionID,
		EnrichmentID:     enrichmentID,
		Status:           string(analysis.StatusCompleted),
		ErrorMessage:     nil,
		ProcessingTimeMs: 8245,
		CreatedAt:        now,
		UpdatedAt:        completedAt,
		CompletedAt:      &completedAt,
		Version:          1,
		ParentAnalysisID: nil,
		IsLatest:         true,
		ApprovedAt:       nil,
		ApprovedBy:       nil,
		SentAt:           nil,
		SentTo:           nil,
		DeletedAt:        nil,

		// 11 Frameworks
		PESTEL:         getTestPESTEL(),
		Porter:         getTestPorter(),
		SWOT:           getTestSWOT(),
		TamSamSom:      getTestTamSamSom(),
		BlueOcean:      getTestBlueOcean(),
		OKRs:           getTestOKRs(),
		BSC:            getTestBSC(),
		Benchmarking:   getTestBenchmarking(),
		GrowthHacking:  getTestGrowthHacking(),
		Scenarios:      getTestScenarios(),
		DecisionMatrix: getTestDecisionMatrix(),
		Synthesis:      getTestSynthesis(),
	}
}

// =================================================================
// Sample UUIDs and Constants
// =================================================================

var (
	TestSubmissionID = uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	TestEnrichmentID = uuid.MustParse("b1fecba0-ad1c-5fg9-cc7e-7cca6e491b22")
	TestAnalysisID   = uuid.MustParse("c2gfddbb-be2d-6gh0-dd8f-8ddb7f502c33")
	TestUserID       = uuid.MustParse("d3hgeecc-cf3e-7hi1-ee9g-9eec8g613d44")
)

const (
	TestEmail    = "test@example.com"
	TestPassword = "SecureP@ssw0rd123!"
	TestAPIKey   = "test-api-key-123456789"
)

// =================================================================
// Internal Helper Functions for Framework Fixtures
// =================================================================

func getTestPESTEL() analysis.PESTELAnalysis {
	return analysis.PESTELAnalysis{
		Political:     []string{"Stable democracy with periodic reforms", "Pro-tech government initiatives"},
		Economic:      []string{"Moderate GDP growth (+2.3%)", "High interest rates (11.25%)"},
		Social:        []string{"Growing tech workforce", "Increasing cloud adoption"},
		Technological: []string{"Kubernetes ecosystem maturation", "5G rollout expanding"},
		Environmental: []string{"Pressure for green datacenters", "Carbon footprint regulations proposed"},
		Legal:         []string{"LGPD enforcement strengthened", "Data residency requirements"},
		Summary:       "Favorable macro environment for cloud infrastructure with regulatory tailwinds",
	}
}

func getTestPorter() analysis.PorterAnalysis {
	return analysis.PorterAnalysis{
		CompetitiveRivalry:                   "Intense competition from AWS, Azure, GCP",
		SupplierPower:                        "Moderate - dependent on datacenter providers",
		BuyerPower:                           "High - enterprise customers have leverage",
		ThreatNewEntrants:                    "Medium - high capital requirements",
		ThreatSubstitutes:                    "Medium - open-source alternatives exist",
		PowerPartnershipsEcosystems:          "High - Kubernetes ecosystem partnerships critical",
		DisruptionAIData:                     "High - AI-driven infrastructure optimization emerging",
		CompetitiveRivalryIntensity:          "Alta",
		SupplierPowerIntensity:               "Média",
		BuyerPowerIntensity:                  "Alta",
		ThreatNewEntrantsIntensity:           "Média",
		ThreatSubstitutesIntensity:           "Média",
		PowerPartnershipsEcosystemsIntensity: "Alta",
		DisruptionAIDataIntensity:            "Alta",
		StrategicImplications:                []string{"Differentiate on service quality", "Build ecosystem partnerships", "Invest in AI automation", "Focus on niche verticals"},
		OverallAttractiveness:                "Moderately attractive - high growth but intense competition",
		Summary:                              "Challenging competitive landscape requiring differentiation strategy",
	}
}

func getTestSWOT() analysis.SWOTAnalysis {
	return analysis.SWOTAnalysis{
		Strengths: []analysis.SWOTItem{
			{Content: "Technical excellence in Kubernetes", Confidence: "Alta", Source: "fato"},
			{Content: "High customer retention (92%)", Confidence: "Alta", Source: "fato"},
			{Content: "Fast deployment (5 min avg)", Confidence: "Alta", Source: "fato"},
		},
		Weaknesses: []analysis.SWOTItem{
			{Content: "Limited brand vs AWS/Azure", Confidence: "Alta", Source: "análise de mercado"},
			{Content: "Small marketing team", Confidence: "Média", Source: "estimativa"},
		},
		Opportunities: []analysis.SWOTItem{
			{Content: "Cloud market growing 22% CAGR", Confidence: "Alta", Source: "análise de mercado"},
			{Content: "LGPD driving local cloud demand", Confidence: "Alta", Source: "fato"},
		},
		Threats: []analysis.SWOTItem{
			{Content: "AWS aggressive pricing", Confidence: "Alta", Source: "análise de mercado"},
			{Content: "Chinese cloud providers entering", Confidence: "Média", Source: "estimativa"},
		},
		Summary: "Strong technical foundation with growth opportunities if marketing improves",
	}
}

func getTestTamSamSom() analysis.TamSamSomAnalysis {
	return analysis.TamSamSomAnalysis{
		TAM:                "R$ 45B (Brazil cloud market 2025)",
		SAM:                "R$ 8B (Kubernetes infrastructure segment)",
		SOM:                "R$ 120M (achievable in 3 years with 1.5% market share)",
		Assumptions:        []string{"22% CAGR cloud growth", "Enterprise adoption rate 35%", "Customer acquisition cost R$ 25k"},
		CAGR:               "22% (2024-2028)",
		DataQuality:        "complete",
		NextSteps:          []string{},
		ProxyIndicators:    []string{},
		ExpectedOutputs:    []string{},
		MethodologicalNote: "Based on Gartner cloud forecasts and local market research",
		Summary:            "Large addressable market with realistic 3-year revenue target",
	}
}

func getTestBlueOcean() analysis.BlueOceanAnalysis {
	return analysis.BlueOceanAnalysis{
		Eliminate:     []string{"Complex pricing models", "Vendor lock-in constraints"},
		Reduce:        []string{"Setup complexity", "Support response times"},
		Raise:         []string{"Deployment speed", "Developer experience"},
		Create:        []string{"Transparent flat-rate pricing", "Brazilian Portuguese support"},
		NewValueCurve: "Developer-first platform with Brazilian focus",
		Summary:       "Differentiate on simplicity and local market expertise",
	}
}

func getTestOKRs() analysis.OKRAnalysis {
	return analysis.OKRAnalysis{
		Quarters: []analysis.QuarterlyOKR{
			{
				Quarter:   "Q1 2025",
				Objective: "Expand customer base",
				KeyResults: []string{
					"Acquire 25 new enterprise customers",
					"Achieve 95% customer satisfaction",
					"Launch automated onboarding",
				},
				Investment: "R$ 180k",
				Timeline:   "3 months",
			},
			{
				Quarter:   "Q2 2025",
				Objective: "Improve platform reliability",
				KeyResults: []string{
					"Achieve 99.95% uptime",
					"Reduce incident response to <5min",
					"Complete SOC 2 certification",
				},
				Investment: "R$ 220k",
				Timeline:   "3 months",
			},
			{
				Quarter:   "Q3 2025",
				Objective: "Scale infrastructure",
				KeyResults: []string{
					"Support 10k concurrent deployments",
					"Launch multi-region support",
					"Reduce costs by 15%",
				},
				Investment: "R$ 350k",
				Timeline:   "3 months",
			},
		},
		Summary: "Focus on growth, reliability, and scale",
	}
}

func getTestBSC() analysis.BalancedScorecardAnalysis {
	return analysis.BalancedScorecardAnalysis{
		Financial:      []string{"Increase ARR by 40%", "Reduce churn to <5%"},
		Customer:       []string{"NPS >50", "95% satisfaction"},
		Internal:       []string{"Deploy in <5min", "99.95% uptime"},
		LearningGrowth: []string{"Train 100% team on new features", "Hire 15 engineers"},
		Summary:        "Balanced growth across all perspectives",
	}
}

func getTestBenchmarking() analysis.BenchmarkingAnalysis {
	return analysis.BenchmarkingAnalysis{
		Competitors:     []string{"AWS EKS", "Azure AKS", "GCP GKE"},
		PerformanceGaps: []string{"Brand recognition", "Global presence", "Sales reach"},
		BestPractices:   []string{"AWS pricing transparency", "Azure developer tools", "GCP automation"},
		Summary:         "Learn from global leaders, execute with local focus",
	}
}

func getTestGrowthHacking() analysis.GrowthHackingAnalysis {
	return analysis.GrowthHackingAnalysis{
		LeapLoop: analysis.GrowthLoop{
			Name:       "LEAP Loop",
			Type:       "acquisition",
			Steps:      []string{"Land", "Engage", "Activate", "Propagate"},
			Metrics:    []string{"CAC: R$ 25k", "Conversion: 12%", "Time-to-value: 5 days"},
			Bottleneck: "Low brand awareness - need content marketing",
		},
		ScaleLoop: analysis.GrowthLoop{
			Name:       "SCALE Loop",
			Type:       "monetization",
			Steps:      []string{"Satisfy", "Convert", "Loop Back", "Expand"},
			Metrics:    []string{"NRR: 115%", "Upsell rate: 25%", "Churn: 8%"},
			Bottleneck: "Limited upsell opportunities - need premium tier",
		},
		Summary: "Strong retention, need better acquisition",
	}
}

func getTestScenarios() analysis.ScenarioAnalysis {
	return analysis.ScenarioAnalysis{
		Optimistic: analysis.Scenario{
			Name:            "Cenário Otimista",
			Probability:     20,
			Description:     "Market grows 30% annually, we capture 3% share",
			RequiredActions: []string{"Aggressive hiring", "Expand to LATAM"},
		},
		Realist: analysis.Scenario{
			Name:            "Cenário Realista",
			Probability:     60,
			Description:     "Market grows 22%, we capture 1.5% share",
			RequiredActions: []string{"Focus on customer success", "Build partnerships"},
		},
		Pessimistic: analysis.Scenario{
			Name:            "Cenário Pessimista",
			Probability:     20,
			Description:     "Economic downturn, market grows 10%",
			RequiredActions: []string{"Cost optimization", "Focus on retention"},
		},
		MitigationTactics:   []string{"Diversify revenue streams", "Build cash reserves"},
		EarlyWarningSignals: []string{"Customer acquisition slowing", "Churn increasing"},
		Summary:             "Prepare for realistic scenario, monitor for pessimistic",
	}
}

func getTestDecisionMatrix() analysis.DecisionMatrixAnalysis {
	return analysis.DecisionMatrixAnalysis{
		Alternatives:        []string{"Expand LATAM", "Build enterprise features", "Partner with AWS"},
		Criteria:            []string{"ROI", "Time to market", "Risk", "Strategic fit"},
		FinalRecommendation: "Focus on enterprise features first, then LATAM expansion",
		RecommendedOption:   "Build enterprise features",
		Score:               "8.2/10",
		ScoreComparison:     "+18% above second option",
		PriorityRecommendations: []analysis.PriorityRecommendation{
			{Priority: 1, Title: "Enterprise tier launch", Description: "Build SOC 2, SSO, RBAC", Timeline: "6 months", Budget: "R$ 450k"},
			{Priority: 2, Title: "LATAM expansion", Description: "Mexico and Argentina markets", Timeline: "9 months", Budget: "R$ 320k"},
			{Priority: 3, Title: "Strategic partnerships", Description: "AWS/Azure marketplace", Timeline: "12 months", Budget: "R$ 180k"},
		},
		ReviewCycle: analysis.ReviewCycle{
			Frequency:             "Trimestral",
			ExtraordinaryTriggers: []string{"Major competitor moves", "Economic shifts", "Funding rounds"},
		},
		MonitoringMetrics: []string{"ARR growth", "Customer acquisition", "Churn rate", "NPS"},
		Summary:           "Enterprise features critical for revenue growth",
	}
}

func getTestSynthesis() analysis.AnalysisSynthesis {
	return analysis.AnalysisSynthesis{
		ExecutiveSummary: "Acme Tech has strong technical foundation in growing market. Key priority: enhance enterprise features and improve brand awareness.",
		CentralChallenge: "Scale revenue 3x while maintaining technical excellence and customer satisfaction",
		MainFindings: []string{
			"Strong: Technical excellence, high retention",
			"Weak: Brand awareness, marketing reach",
			"Opportunity: 22% CAGR market growth",
			"Threat: AWS aggressive pricing",
		},
		ImportantNotes: []string{
			"Customer retention (92%) is exceptional - protect this",
			"Marketing is critical bottleneck - needs investment",
			"SOC 2 certification unlocks enterprise deals",
		},
		KeyFindings: []string{
			"Technical product is strong",
			"Market is growing rapidly",
			"Need to invest in sales/marketing",
		},
		StrategicPriorities: []string{
			"Build enterprise features (SOC 2, SSO)",
			"Invest in content marketing",
			"Expand to LATAM markets",
		},
		Roadmap: []string{
			"Q1: Enterprise tier + marketing",
			"Q2: SOC 2 certification",
			"Q3: LATAM expansion prep",
		},
		OverallRecommendation: "Focus on enterprise segment with strong marketing support",
	}
}
