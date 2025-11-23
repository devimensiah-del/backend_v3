package testutils

// =================================================================
// Mock LLM Response Fixtures
// These are comprehensive JSON responses for all 11 analysis frameworks
// =================================================================

func getDefaultEnrichmentResponse() string {
	return `{
  "profile_overview": {
    "legal_name": "Acme Tech Solutions LTDA",
    "website": "https://acme-tech.com.br",
    "foundation_year": "2019",
    "headquarters": "São Paulo, SP, Brazil"
  },
  "market_position": {
    "sector": "Cloud Infrastructure & DevOps",
    "target_audience": "Enterprise B2B - Tech companies with 100+ developers",
    "value_proposition": "Kubernetes-native platform with 99.9% uptime SLA and auto-scaling"
  },
  "financials": {
    "employees_range": "50-200",
    "revenue_estimate": "R$ 8-12M ARR (2024)",
    "business_model": "Subscription SaaS - Monthly recurring revenue"
  },
  "competitive_landscape": {
    "competitors": [
      "AWS Elastic Kubernetes Service",
      "Google Cloud Platform GKE",
      "Azure Kubernetes Service",
      "DigitalOcean Kubernetes"
    ],
    "market_share_status": "Emerging player with <2% market share in Brazil"
  },
  "strategic_assessment": {
    "digital_maturity": 8,
    "strengths": [
      "Strong technical team with Kubernetes expertise",
      "Excellent product-market fit in Brazilian market",
      "High customer retention (92% annual)",
      "Fast deployment times (avg 5 minutes)"
    ],
    "weaknesses": [
      "Limited brand recognition vs global players",
      "Small sales and marketing team",
      "Dependency on single market (Brazil)",
      "Limited multi-cloud support"
    ]
  },
  "macro_context": {
    "economic_indicators": {
      "country": "Brazil",
      "gdp_growth": "+2.3% (2025 forecast)",
      "inflation_rate": "IPCA: 4.5% a.a.",
      "interest_rate": "SELIC: 11.25%",
      "exchange_rate": "USD/BRL: 5.15",
      "unemployment_rate": "7.8%",
      "political_stability": "Moderate - ongoing fiscal reforms",
      "economic_outlook": "Cautious growth with focus on tech sector investment",
      "recent_policy_changes": ["Tax reform 2025", "Digital services taxation"]
    },
    "industry_trends": {
      "industry_sector": "Cloud Computing & Infrastructure",
      "growth_rate": "+22% CAGR (2024-2028)",
      "key_trends": ["Kubernetes adoption surge", "Multi-cloud strategies", "Edge computing"],
      "technology_adoption": "High - 68% of enterprises adopting cloud-native",
      "market_concentration": "Concentrated - AWS 32%, Azure 21%, GCP 9%",
      "barriers_to_entry": "High - requires significant capital and technical expertise",
      "mergers_acquisitions": ["AWS acquired CloudEndure (2024)"]
    },
    "regulatory_landscape": {
      "recent_regulations": ["LGPD enforcement strengthened", "Cloud data residency requirements"],
      "upcoming_changes": ["Carbon tax proposal", "AI governance framework"],
      "compliance_requirements": "LGPD, ISO 27001, SOC 2 Type II",
      "industry_standards": ["ISO 27001", "SOC 2", "PCI-DSS"]
    },
    "market_signals": {
      "commodity_prices": ["Cloud compute costs stable", "Bandwidth costs down 8% YoY"],
      "supply_chain_status": "Normal - no major disruptions",
      "consumer_sentiment": "Cautiously optimistic about tech sector",
      "competitor_activity": ["AWS opened new Brazil region"],
      "emerging_threats": ["Aggressive pricing from Chinese cloud providers"]
    },
    "data_sources": ["web_search", "industry_reports", "competitor_analysis"],
    "last_updated": "2025-01-22"
  }
}`
}

func getDefaultPESTELResponse() string {
	return `{
  "political": ["Stable democracy with periodic reforms", "Pro-tech government initiatives"],
  "economic": ["Moderate GDP growth (+2.3%)", "High interest rates (11.25%)"],
  "social": ["Growing tech workforce", "Increasing cloud adoption"],
  "technological": ["Kubernetes ecosystem maturation", "5G rollout expanding"],
  "environmental": ["Pressure for green datacenters", "Carbon footprint regulations proposed"],
  "legal": ["LGPD enforcement strengthened", "Data residency requirements"],
  "summary": "Favorable macro environment for cloud infrastructure with regulatory tailwinds"
}`
}

func getDefaultPorterResponse() string {
	return `{
  "competitive_rivalry": "Intense competition from AWS, Azure, GCP",
  "supplier_power": "Moderate - dependent on datacenter providers",
  "buyer_power": "High - enterprise customers have leverage",
  "threat_new_entrants": "Medium - high capital requirements",
  "threat_substitutes": "Medium - open-source alternatives exist",
  "power_partnerships_ecosystems": "High - Kubernetes ecosystem partnerships critical",
  "disruption_ai_data": "High - AI-driven infrastructure optimization emerging",
  "competitive_rivalry_intensity": "Alta",
  "supplier_power_intensity": "Média",
  "buyer_power_intensity": "Alta",
  "threat_new_entrants_intensity": "Média",
  "threat_substitutes_intensity": "Média",
  "power_partnerships_ecosystems_intensity": "Alta",
  "disruption_ai_data_intensity": "Alta",
  "strategic_implications": [
    "Differentiate on service quality",
    "Build ecosystem partnerships",
    "Invest in AI automation",
    "Focus on niche verticals"
  ],
  "overall_attractiveness": "Moderately attractive - high growth but intense competition",
  "summary": "Challenging competitive landscape requiring differentiation strategy"
}`
}

func getDefaultSWOTResponse() string {
	return `{
  "strengths": [
    {"content": "Technical excellence in Kubernetes", "confidence": "Alta", "source": "fato"},
    {"content": "High customer retention (92%)", "confidence": "Alta", "source": "fato"},
    {"content": "Fast deployment (5 min avg)", "confidence": "Alta", "source": "fato"}
  ],
  "weaknesses": [
    {"content": "Limited brand vs AWS/Azure", "confidence": "Alta", "source": "análise de mercado"},
    {"content": "Small marketing team", "confidence": "Média", "source": "estimativa"}
  ],
  "opportunities": [
    {"content": "Cloud market growing 22% CAGR", "confidence": "Alta", "source": "análise de mercado"},
    {"content": "LGPD driving local cloud demand", "confidence": "Alta", "source": "fato"}
  ],
  "threats": [
    {"content": "AWS aggressive pricing", "confidence": "Alta", "source": "análise de mercado"},
    {"content": "Chinese cloud providers entering", "confidence": "Média", "source": "estimativa"}
  ],
  "summary": "Strong technical foundation with growth opportunities if marketing improves"
}`
}

func getDefaultTamSamSomResponse() string {
	return `{
  "tam": "R$ 45B (Brazil cloud market 2025)",
  "sam": "R$ 8B (Kubernetes infrastructure segment)",
  "som": "R$ 120M (achievable in 3 years with 1.5% market share)",
  "assumptions": [
    "22% CAGR cloud growth",
    "Enterprise adoption rate 35%",
    "Customer acquisition cost R$ 25k"
  ],
  "cagr": "22% (2024-2028)",
  "data_quality": "complete",
  "next_steps": [],
  "proxy_indicators": [],
  "expected_outputs": [],
  "methodological_note": "Based on Gartner cloud forecasts and local market research",
  "summary": "Large addressable market with realistic 3-year revenue target"
}`
}

func getDefaultBlueOceanResponse() string {
	return `{
  "eliminate": ["Complex pricing models", "Vendor lock-in constraints"],
  "reduce": ["Setup complexity", "Support response times"],
  "raise": ["Deployment speed", "Developer experience"],
  "create": ["Transparent flat-rate pricing", "Brazilian Portuguese support"],
  "new_value_curve": "Developer-first platform with Brazilian focus",
  "summary": "Differentiate on simplicity and local market expertise"
}`
}

func getDefaultOKRResponse() string {
	return `{
  "quarters": [
    {
      "quarter": "Q1 2025",
      "objective": "Expand customer base",
      "key_results": [
        "Acquire 25 new enterprise customers",
        "Achieve 95% customer satisfaction",
        "Launch automated onboarding"
      ],
      "investment": "R$ 180k",
      "timeline": "3 months"
    },
    {
      "quarter": "Q2 2025",
      "objective": "Improve platform reliability",
      "key_results": [
        "Achieve 99.95% uptime",
        "Reduce incident response to <5min",
        "Complete SOC 2 certification"
      ],
      "investment": "R$ 220k",
      "timeline": "3 months"
    },
    {
      "quarter": "Q3 2025",
      "objective": "Scale infrastructure",
      "key_results": [
        "Support 10k concurrent deployments",
        "Launch multi-region support",
        "Reduce costs by 15%"
      ],
      "investment": "R$ 350k",
      "timeline": "3 months"
    }
  ],
  "summary": "Focus on growth, reliability, and scale"
}`
}

func getDefaultBSCResponse() string {
	return `{
  "financial": ["Increase ARR by 40%", "Reduce churn to <5%"],
  "customer": ["NPS >50", "95% satisfaction"],
  "internal_processes": ["Deploy in <5min", "99.95% uptime"],
  "learning_growth": ["Train 100% team on new features", "Hire 15 engineers"],
  "summary": "Balanced growth across all perspectives"
}`
}

func getDefaultBenchmarkingResponse() string {
	return `{
  "competitors_analyzed": ["AWS EKS", "Azure AKS", "GCP GKE"],
  "performance_gaps": ["Brand recognition", "Global presence", "Sales reach"],
  "best_practices": [
    "AWS pricing transparency",
    "Azure developer tools",
    "GCP automation"
  ],
  "summary": "Learn from global leaders, execute with local focus"
}`
}

func getDefaultGrowthHackingResponse() string {
	return `{
  "leap_loop": {
    "name": "LEAP Loop",
    "type": "acquisition",
    "steps": ["Land", "Engage", "Activate", "Propagate"],
    "metrics": ["CAC: R$ 25k", "Conversion: 12%", "Time-to-value: 5 days"],
    "bottleneck": "Low brand awareness - need content marketing"
  },
  "scale_loop": {
    "name": "SCALE Loop",
    "type": "monetization",
    "steps": ["Satisfy", "Convert", "Loop Back", "Expand"],
    "metrics": ["NRR: 115%", "Upsell rate: 25%", "Churn: 8%"],
    "bottleneck": "Limited upsell opportunities - need premium tier"
  },
  "summary": "Strong retention, need better acquisition"
}`
}

func getDefaultScenarioResponse() string {
	return `{
  "optimistic": {
    "name": "Cenário Otimista",
    "probability": 20,
    "description": "Market grows 30% annually, we capture 3% share",
    "required_actions": ["Aggressive hiring", "Expand to LATAM"]
  },
  "realist": {
    "name": "Cenário Realista",
    "probability": 60,
    "description": "Market grows 22%, we capture 1.5% share",
    "required_actions": ["Focus on customer success", "Build partnerships"]
  },
  "pessimistic": {
    "name": "Cenário Pessimista",
    "probability": 20,
    "description": "Economic downturn, market grows 10%",
    "required_actions": ["Cost optimization", "Focus on retention"]
  },
  "mitigation_tactics": ["Diversify revenue streams", "Build cash reserves"],
  "early_warning_signals": ["Customer acquisition slowing", "Churn increasing"],
  "summary": "Prepare for realistic scenario, monitor for pessimistic"
}`
}

func getDefaultDecisionMatrixResponse() string {
	return `{
  "alternatives": ["Expand LATAM", "Build enterprise features", "Partner with AWS"],
  "criteria": ["ROI", "Time to market", "Risk", "Strategic fit"],
  "final_recommendation": "Focus on enterprise features first, then LATAM expansion",
  "recommended_option": "Build enterprise features",
  "score": "8.2/10",
  "score_comparison": "+18% above second option",
  "priority_recommendations": [
    {
      "priority": 1,
      "title": "Enterprise tier launch",
      "description": "Build SOC 2, SSO, RBAC",
      "timeline": "6 months",
      "budget": "R$ 450k"
    },
    {
      "priority": 2,
      "title": "LATAM expansion",
      "description": "Mexico and Argentina markets",
      "timeline": "9 months",
      "budget": "R$ 320k"
    },
    {
      "priority": 3,
      "title": "Strategic partnerships",
      "description": "AWS/Azure marketplace",
      "timeline": "12 months",
      "budget": "R$ 180k"
    }
  ],
  "review_cycle": {
    "frequency": "Trimestral",
    "extraordinary_triggers": ["Major competitor moves", "Economic shifts", "Funding rounds"]
  },
  "monitoring_metrics": ["ARR growth", "Customer acquisition", "Churn rate", "NPS"],
  "summary": "Enterprise features critical for revenue growth"
}`
}

func getDefaultSynthesisResponse() string {
	return `{
  "executive_summary": "Acme Tech has strong technical foundation in growing market. Key priority: enhance enterprise features and improve brand awareness.",
  "central_challenge": "Scale revenue 3x while maintaining technical excellence and customer satisfaction",
  "main_findings": [
    "Strong: Technical excellence, high retention",
    "Weak: Brand awareness, marketing reach",
    "Opportunity: 22% CAGR market growth",
    "Threat: AWS aggressive pricing"
  ],
  "important_notes": [
    "Customer retention (92%) is exceptional - protect this",
    "Marketing is critical bottleneck - needs investment",
    "SOC 2 certification unlocks enterprise deals"
  ],
  "key_findings": [
    "Technical product is strong",
    "Market is growing rapidly",
    "Need to invest in sales/marketing"
  ],
  "strategic_priorities": [
    "Build enterprise features (SOC 2, SSO)",
    "Invest in content marketing",
    "Expand to LATAM markets"
  ],
  "roadmap": [
    "Q1: Enterprise tier + marketing",
    "Q2: SOC 2 certification",
    "Q3: LATAM expansion prep"
  ],
  "overall_recommendation": "Focus on enterprise segment with strong marketing support"
}`
}
