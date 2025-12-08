# TypeScript Schema Definitions

**Complete frontend type definitions for IMENSIAH API**

All types are ready to copy directly into your TypeScript project.

---

## Table of Contents

- [Authentication](#authentication)
- [Submissions](#submissions)
- [Companies](#companies)
- [Challenges](#challenges)
- [Analysis](#analysis)
- [Frameworks](#frameworks)
- [Wizard](#wizard)
- [Admin](#admin)
- [Common Types](#common-types)

---

## Authentication

```typescript
// ============================================
// AUTH REQUEST/RESPONSE TYPES
// ============================================

export interface LoginRequest {
  email: string;
  password: string;
}

export interface SignupRequest {
  email: string;
  password: string;
  fullName: string;
}

export interface ForgotPasswordRequest {
  email: string;
}

export interface ResetPasswordRequest {
  token: string;
  newPassword: string;
}

export interface UpdatePasswordRequest {
  currentPassword: string;
  newPassword: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  token: string; // Alias for access_token (both provided for compatibility)
  token_type: "bearer";
  expires_in: number; // Seconds until expiration
  expires_at?: number; // Unix timestamp
}

export interface User {
  id: string; // UUID
  email: string;
  fullName?: string;
  role: UserRole;
  isActive: boolean;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

export type UserRole = "user" | "admin" | "super_admin" | "service_role";

export interface UserProfileResponse {
  user: User;
}

export interface UpdateProfileRequest {
  fullName: string;
}
```

---

## Submissions

```typescript
// ============================================
// SUBMISSION TYPES
// ============================================

export interface CreateSubmissionRequest {
  // Required
  companyName: string;
  challengeCategory: ChallengeCategory;
  challengeType: ChallengeType;
  businessChallenge: string;

  // Optional company data
  cnpj?: string;
  industry?: string;
  companySize?: string;
  website?: string;

  // Additional info (JSON string)
  additionalInfo?: string; // Stringified AdditionalInfoData
}

export interface AdditionalInfoData {
  // Required
  contactName: string;
  contactEmail: string;

  // Optional
  contactPhone?: string;
  contactPosition?: string;
  companyLocation?: string;
  targetMarket?: string;
  annualRevenueMin?: number;
  annualRevenueMax?: number;
  fundingStage?: string;
  additionalNotes?: string;
  linkedinUrl?: string;
  twitterHandle?: string;
}

export interface CreateSubmissionResponse {
  submission: {
    id: string; // UUID
    companyId: string; // UUID
    challengeId: string; // UUID
    createdAt: string; // ISO 8601
    updatedAt: string; // ISO 8601
  };
}

export interface SubmissionListResponse {
  data: SubmissionDetailResponse[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface SubmissionDetailResponse {
  // Core fields
  id: string; // UUID
  userId?: string; // UUID (null for anonymous submissions)

  // Company data (from submission)
  companyName: string;
  cnpj?: string;
  companyWebsite?: string;
  companyIndustry?: string;
  companySize?: string;
  companyLocation?: string;

  // Contact data
  contactName: string;
  contactEmail: string;
  contactPhone?: string;
  contactPosition?: string;

  // Business context
  targetMarket?: string;
  annualRevenueMin?: number;
  annualRevenueMax?: number;
  fundingStage?: string;
  additionalNotes?: string;
  linkedinUrl?: string;
  twitterHandle?: string;

  // Challenge data
  challengeCategory: ChallengeCategory;
  challengeType: ChallengeType;
  businessChallenge: string;

  // Related entity IDs
  companyId?: string; // UUID
  challengeId?: string; // UUID
  analysisId?: string; // UUID
  pdfUrl?: string;

  // Derived status
  status: SubmissionStatus;

  // Timestamps
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

export type SubmissionStatus =
  | "pending"      // Waiting for enrichment to start
  | "enriching"    // Company enrichment in progress
  | "enriched"     // Enrichment complete, awaiting analysis
  | "analyzing"    // Analysis job running
  | "completed"    // Analysis completed successfully
  | "failed";      // Enrichment or analysis failed

export type ChallengeCategory =
  | "growth"
  | "transform"
  | "transition"
  | "compete"
  | "funding";

// See Challenge Types section for full list
export type ChallengeType = string;
```

---

## Companies

```typescript
// ============================================
// COMPANY TYPES
// ============================================

export interface CreateCompanyRequest {
  // Required
  name: string;

  // Optional
  website?: string;
  cnpj?: string;
  industry?: string;
  company_size?: string;
  location?: string;
  target_market?: string;
  funding_stage?: string;
  annual_revenue_min?: number;
  annual_revenue_max?: number;
  linkedin_url?: string;
  twitter_handle?: string;
}

export interface CompanyResponse {
  id: string; // UUID
  name: string;
  cnpj?: string;
  website?: string;

  // Business context (from submission)
  industry?: string;
  companySize?: string;
  location?: string;
  targetMarket?: string;
  fundingStage?: string;
  annualRevenueMin?: number;
  annualRevenueMax?: number;

  // Enriched data (from Perplexity AI)
  foundationYear?: string;
  legalName?: string;
  headquarters?: string;
  sector?: string;
  targetAudience?: string;
  valueProposition?: string;
  employeesRange?: string;
  revenueEstimate?: string;
  businessModel?: string;
  competitors?: string[];
  marketShareStatus?: string;
  digitalMaturity?: number; // 1-10 scale
  strengths?: string[];
  weaknesses?: string[];

  // Social links
  linkedinUrl?: string;
  twitterHandle?: string;

  // Enrichment status
  enrichmentStatus: EnrichmentStatus;
  enrichmentCompletedAt?: string; // ISO 8601
  enrichmentError?: string;

  // Access control
  ownerId?: string; // UUID
  allowedUsers: string[]; // UUID[]

  // Timestamps
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

export type EnrichmentStatus =
  | "pending"      // Queued for enrichment
  | "processing"   // Enrichment in progress
  | "completed"    // Enrichment finished successfully
  | "failed";      // Enrichment failed

export interface CompanyListResponse {
  companies: CompanyResponse[];
  count: number;
}

export interface CompanyWithHistoryResponse extends CompanyResponse {
  analyses_history: AnalysisHistoryItem[];
}

export interface AnalysisHistoryItem {
  analysis_id: string; // UUID
  submission_id: string; // UUID
  status: AnalysisStatus;
  business_challenge: string;
  is_blurred: boolean;
  is_visible_to_user: boolean;
  is_public: boolean;
  access_code?: string;
  pdf_url?: string;
  completed_at?: string; // ISO 8601
  created_at: string; // ISO 8601
  updated_at: string; // ISO 8601
}

export interface ReAnalyzeCompanyRequest {
  challenge_category: ChallengeCategory;
  challenge_type: ChallengeType;
  business_challenge: string;
}

export interface ReAnalyzeCompanyResponse {
  message: string;
  data: {
    submission_id: string; // UUID
    company_id: string; // UUID
    challenge_id: string; // UUID
    challenge_category: ChallengeCategory;
    challenge_type: ChallengeType;
  };
}
```

---

## Challenges

```typescript
// ============================================
// CHALLENGE TYPES
// ============================================

export interface Challenge {
  id: string; // UUID
  company_id: string; // UUID
  challenge_category: ChallengeCategory;
  challenge_type: ChallengeType;
  business_challenge: string;
  created_at: string; // ISO 8601
  updated_at: string; // ISO 8601
  deleted_at?: string; // ISO 8601
}

// Challenge Categories and Types
export const ChallengeCategories = {
  growth: "growth",
  transform: "transform",
  transition: "transition",
  compete: "compete",
  funding: "funding"
} as const;

export const ChallengeTypesByCategory = {
  growth: [
    "growth_organic",
    "growth_geographic",
    "growth_segment",
    "growth_product",
    "growth_channel"
  ],
  transform: [
    "transform_digital",
    "transform_model",
    "transform_culture",
    "transform_operational"
  ],
  transition: [
    "transition_succession",
    "transition_exit",
    "transition_merger",
    "transition_turnaround"
  ],
  compete: [
    "compete_differentiate",
    "compete_defend",
    "compete_reposition"
  ],
  funding: [
    "funding_raise",
    "funding_debt",
    "funding_ipo"
  ]
} as const;

export type ChallengeType =
  // Growth
  | "growth_organic"
  | "growth_geographic"
  | "growth_segment"
  | "growth_product"
  | "growth_channel"
  // Transform
  | "transform_digital"
  | "transform_model"
  | "transform_culture"
  | "transform_operational"
  // Transition
  | "transition_succession"
  | "transition_exit"
  | "transition_merger"
  | "transition_turnaround"
  // Compete
  | "compete_differentiate"
  | "compete_defend"
  | "compete_reposition"
  // Funding
  | "funding_raise"
  | "funding_debt"
  | "funding_ipo";

export interface ChallengeTypesResponse {
  categories: ChallengeCategory[];
  types: Record<ChallengeCategory, ChallengeType[]>;
}
```

---

## Analysis

```typescript
// ============================================
// ANALYSIS TYPES
// ============================================

export interface AnalysisResponse {
  id: string; // UUID
  submission_id?: string; // UUID (nullable - historical reference)
  company_id?: string; // UUID
  challenge_id: string; // UUID
  status: AnalysisStatus;
  analysis: FrameworkResults;
  is_visible_to_user: boolean;
  is_blurred: boolean;
  is_public: boolean;
  access_code?: string;
  created_at: string; // ISO 8601
  updated_at: string; // ISO 8601
}

export type AnalysisStatus =
  | "pending"      // Queued, waiting for worker
  | "processing"   // Worker actively running
  | "completed"    // Worker finished successfully
  | "failed";      // Worker encountered error

export interface FrameworkResults {
  pestel?: PESTELAnalysis;
  porter?: PorterAnalysis;
  swot?: SWOTAnalysis;
  tam_sam_som?: TamSamSomAnalysis;
  benchmarking?: BenchmarkingAnalysis;
  blue_ocean?: BlueOceanAnalysis;
  growth_hacking?: GrowthHackingAnalysis;
  scenarios?: ScenarioAnalysis;
  okrs?: OKRAnalysis;
  bsc?: BalancedScorecardAnalysis;
  decision_matrix?: DecisionMatrixAnalysis;
  synthesis?: AnalysisSynthesis;
}

// ============================================
// FRAMEWORK RESULT SCHEMAS
// ============================================

export interface PESTELAnalysis {
  political: string[];
  economic: string[];
  social: string[];
  technological: string[];
  environmental: string[];
  legal: string[];
  summary: string;
}

export interface PorterAnalysis {
  // Traditional 5 Forces
  competitive_rivalry: string;
  supplier_power: string;
  buyer_power: string;
  threat_new_entrants: string;
  threat_substitutes: string;

  // +2 Modern Forces
  power_partnerships_ecosystems: string;
  disruption_ai_data: string;

  // Intensity ratings for each force
  competitive_rivalry_intensity: "Alta" | "Média" | "Baixa";
  supplier_power_intensity: "Alta" | "Média" | "Baixa";
  buyer_power_intensity: "Alta" | "Média" | "Baixa";
  threat_new_entrants_intensity: "Alta" | "Média" | "Baixa";
  threat_substitutes_intensity: "Alta" | "Média" | "Baixa";
  power_partnerships_ecosystems_intensity: "Alta" | "Média" | "Baixa";
  disruption_ai_data_intensity: "Alta" | "Média" | "Baixa";

  // Strategic implications
  strategic_implications: string[]; // 4 key actionable points
  overall_attractiveness: string;
  summary: string;
}

export interface SWOTItem {
  content: string;
  confidence: "Alta" | "Média" | "Baixa";
  source: "fato" | "análise de mercado" | "estimativa" | "feedback de clientes";
}

export interface SWOTAnalysis {
  strengths: SWOTItem[];
  weaknesses: SWOTItem[];
  opportunities: SWOTItem[];
  threats: SWOTItem[];
  summary: string;
}

export interface TamSamSomAnalysis {
  tam: string;
  sam: string;
  som: string;
  assumptions: string[];
  cagr: string;

  // Transparency fields
  confidence_level: number; // 0-100
  estimation_method: string;
  calculation_notes: string;
  data_sources_used?: {
    direct_data?: string[];
    proxy_calculations?: string[];
    benchmark_extrapolations?: string[];
  };

  // Caveat if confidence < 50
  caveat_message?: string;

  // 3-tier scenario modeling
  tam_scenarios?: {
    conservative: ScenarioEstimate;
    realistic: ScenarioEstimate;
    optimistic: ScenarioEstimate;
  };
  sam_scenarios?: {
    conservative: ScenarioEstimate;
    realistic: ScenarioEstimate;
    optimistic: ScenarioEstimate;
  };
  som_scenarios?: {
    conservative: ScenarioEstimate;
    realistic: ScenarioEstimate;
    optimistic: ScenarioEstimate;
  };
  baseline_recommendation?: "conservative" | "realistic";

  summary: string;
}

export interface ScenarioEstimate {
  value_range: string; // e.g., "R$ 500k-2M"
  market_share?: string; // e.g., "2-5%"
  probability: number; // 20, 30, 50
}

export interface BenchmarkingAnalysis {
  competitors_analyzed: string[];
  performance_gaps: string[];
  best_practices: string[];
  summary: string;
}

export interface BlueOceanAnalysis {
  eliminate: string[];
  reduce: string[];
  raise: string[];
  create: string[];
  new_value_curve: string;
  summary: string;
}

export interface GrowthLoop {
  name: string; // "LEAP Loop" or "SCALE Loop"
  type: "acquisition" | "monetization";
  steps: string[]; // 4 steps
  metrics: string[];
  bottleneck: string;
}

export interface GrowthHackingAnalysis {
  leap_loop: GrowthLoop; // Acquisition loop
  scale_loop: GrowthLoop; // Monetization loop
  summary: string;
}

export interface Scenario {
  name: string; // "Cenário Otimista", "Cenário Realista", "Cenário Pessimista"
  probability: number; // Percentage (e.g., 20, 60, 20)
  description: string; // Max 450 chars
  required_actions: string[];
}

export interface ScenarioAnalysis {
  optimistic: Scenario;
  realist: Scenario;
  pessimistic: Scenario;
  mitigation_tactics: string[];
  early_warning_signals: string[];
  summary: string;
}

export interface MonthlyOKR {
  month: string; // "Mês 1", "Mês 2", "Mês 3"
  focus: string; // "Fundação", "Crescimento", "Consolidação"
  objective: string;
  key_results: string[]; // Exactly 3 KRs
  investment: string;
  aligned_recommendation: string;
}

export interface ExecutionPhase {
  duration: string; // e.g., "Meses 1-3"
  focus: string; // e.g., "Estruturação"
  prerequisites?: string[];
  objectives: string[];
  key_results: string[];
  investment_range: string; // e.g., "R$ 50-100k"
  capacity_requirements?: {
    team: string;
    skills_needed: string[];
    dependencies: string[];
  };
}

export interface CapacityAssessment {
  team_readiness: "Alta" | "Média" | "Baixa";
  budget_adequacy: "Suficiente" | "Limitado" | "Insuficiente";
  stakeholder_alignment: "Alinhado" | "Parcial" | "Resistência";
  blockers?: string[];
}

export interface OKRAnalysis {
  // V2: 90-Day Plan format (preferred)
  plan_90_days?: MonthlyOKR[];
  total_investment?: string;
  success_metrics?: string[];

  // V3: Phased execution model
  execution_phases?: {
    foundation?: ExecutionPhase;
    validation?: ExecutionPhase;
    scale?: ExecutionPhase;
  };
  capacity_assessment?: CapacityAssessment;

  summary: string;
}

export interface BalancedScorecardAnalysis {
  financial: string[];
  customer: string[];
  internal_processes: string[];
  learning_growth: string[];
  summary: string;
}

export interface PriorityRecommendation {
  priority: 1 | 2 | 3;
  title: string;
  description: string;
  timeline: string; // e.g., "9-12 meses"
  budget: string; // e.g., "R$150-250k"
  legal_feasibility?: {
    risk_level: "Baixo" | "Médio" | "Alto" | "Crítico";
    requires_statutory_change: boolean;
    requires_legal_opinion: boolean;
    regulatory_dependencies?: string[];
    mitigation_plan?: string;
  };
}

export interface DecisionMatrixAnalysis {
  alternatives: string[];
  criteria: string[];
  final_recommendation: string;
  recommended_option: string;
  score: string; // e.g., "7.3/10"
  score_comparison: string; // e.g., "+23% above second"
  priority_recommendations: PriorityRecommendation[];
  review_cycle: {
    frequency: string; // "Trimestral", "Mensal"
    extraordinary_triggers: string[];
  };
  monitoring_metrics: string[];
  summary: string;
}

export interface ConsistencyValidation {
  financial_alignment: boolean;
  capacity_alignment: boolean;
  legal_alignment: boolean;
  overall_realism_score: number; // 1-10
  flags?: string[];
}

export interface AnalysisSynthesis {
  executive_summary: string;
  central_challenge: string;
  main_findings: string[]; // 4-point SWOT summary
  important_notes: string[];
  key_findings: string[];
  strategic_priorities: string[];
  roadmap: string[];
  overall_recommendation: string;
  consistency_validation?: ConsistencyValidation;
}

// ============================================
// ADMIN ANALYSIS OPERATIONS
// ============================================

export interface UpdateAnalysisRequest {
  // Any framework can be updated (partial update)
  pestel?: Partial<PESTELAnalysis>;
  porter?: Partial<PorterAnalysis>;
  swot?: Partial<SWOTAnalysis>;
  tam_sam_som?: Partial<TamSamSomAnalysis>;
  benchmarking?: Partial<BenchmarkingAnalysis>;
  blue_ocean?: Partial<BlueOceanAnalysis>;
  growth_hacking?: Partial<GrowthHackingAnalysis>;
  scenarios?: Partial<ScenarioAnalysis>;
  okrs?: Partial<OKRAnalysis>;
  bsc?: Partial<BalancedScorecardAnalysis>;
  decision_matrix?: Partial<DecisionMatrixAnalysis>;
  synthesis?: Partial<AnalysisSynthesis>;
}

export interface ToggleVisibilityRequest {
  visible: boolean;
}

export interface ToggleBlurRequest {
  blurred: boolean;
}

export interface TogglePublicRequest {
  public: boolean;
}

export interface GenerateAccessCodeResponse {
  access_code: string;
  shareable_url: string;
  message: string;
}

export interface PublicReportResponse {
  id: string; // UUID
  submission_id: string; // UUID
  status: AnalysisStatus;
  is_blurred: boolean;
  is_public: boolean;
  analysis: FrameworkResults;
  created_at: string; // ISO 8601
  is_admin_preview?: boolean; // Present when admin preview mode
}
```

---

## Frameworks

```typescript
// ============================================
// FRAMEWORK TYPES
// ============================================

export interface Framework {
  id: string; // UUID
  code: string; // Unique code (e.g., "pestel", "porter")
  name: string; // English name
  name_pt: string; // Portuguese name
  category: string;
  description?: string;
  layer_order: number;
  depends_on: string[]; // Array of framework codes
  is_active: boolean;
  created_at: string; // ISO 8601
  updated_at: string; // ISO 8601
}

export interface FrameworkListResponse {
  data: Framework[];
  meta: {
    total: number;
  };
}

export interface FrameworkResponse {
  data: Framework;
}

export interface CreateFrameworkRequest {
  code: string;
  name: string;
  name_pt: string;
  category: string;
  description?: string;
  layer_order: number;
  depends_on?: string[];
  is_active: boolean;
}

export interface UpdateFrameworkRequest {
  name?: string;
  name_pt?: string;
  category?: string;
  description?: string;
  layer_order?: number;
  depends_on?: string[];
  is_active?: boolean;
}
```

---

## Wizard

```typescript
// ============================================
// WIZARD (HUMAN-IN-THE-LOOP) TYPES
// ============================================

export interface WizardState {
  analysis_id: string; // UUID
  current_step: number; // 0-11
  total_steps: number; // 12
  framework_code: string;
  status: WizardStepStatus;
  output?: any; // Framework-specific output
  human_context?: string;
  human_answers?: Record<string, string>;
  steps_completed: string[]; // Array of completed framework codes
  iteration_count: number;
}

export type WizardStepStatus =
  | "pending"      // Not yet generated
  | "generating"   // AI is generating
  | "generated"    // Output ready for review
  | "approved"     // Human approved, moved to next step
  | "failed";      // Generation failed

export interface WizardResponse {
  state: WizardState;
  message?: string;
}

export interface GenerateStepRequest {
  human_context?: string;
  answers?: Record<string, string>;
}

export interface RefineStepRequest {
  additional_context: string;
  notes?: string;
}

export interface WizardSummaryResponse {
  framework_results: Partial<FrameworkResults>;
}

export interface FrameworkOrderResponse {
  frameworks: FrameworkStepInfo[];
  total_steps: number;
}

export interface FrameworkStepInfo {
  step: number;
  code: string;
  name: string;
  name_pt: string;
  description: string;
  questions?: ClarifyingQuestion[];
}

export interface ClarifyingQuestion {
  id: string;
  question: string;
  type?: "text" | "select" | "multiselect";
  options?: string[];
}
```

---

## Admin

```typescript
// ============================================
// ADMIN TYPES
// ============================================

export interface AdminSubmissionListResponse extends SubmissionListResponse {
  // Same as user list response
}

export interface RetryAnalysisResponse {
  message: string;
  data: {
    id: string; // Submission ID
    challenge_id: string;
  };
}

export interface SystemMetrics {
  submissions_last_24h: number;
  enrichment_success_rate: string; // e.g., "95%"
  analysis_success_rate: string; // e.g., "88%"
  avg_analysis_time_seconds: number;
  total_cost_last_24h_usd: number;
  total_tokens_last_24h: number;
  llm_requests_last_24h: number;
  errors_last_24h: string[];
  last_updated: string; // ISO 8601
}

export interface MacroIndicatorSnapshot {
  selic: {
    value: number;
    date: string; // ISO 8601
    source: string;
  };
  ipca: {
    value: number;
    date: string; // ISO 8601
    source: string;
  };
  usd_brl: {
    value: number;
    date: string; // ISO 8601
    source: string;
  };
}

export interface RefreshIndicatorsResponse {
  message: string;
  updated: string[]; // Array of indicator codes
}

export interface IndicatorHistoryResponse {
  code: string;
  data: Array<{
    value: number;
    date: string; // ISO 8601
  }>;
  range: {
    from: string; // ISO 8601
    to: string; // ISO 8601
  };
}

export interface AdminCompanyListResponse {
  companies: CompanyResponse[];
  total: number;
  limit: number;
  offset: number;
}
```

---

## Common Types

```typescript
// ============================================
// COMMON/SHARED TYPES
// ============================================

export interface MessageResponse {
  message: string;
}

export interface ErrorResponse {
  error: string;
  message: string;
}

export interface PaginationParams {
  page?: number;
  pageSize?: number;
  limit?: number;
}

export interface HealthResponse {
  status: "ok" | "error";
  services?: {
    database?: "connected" | "error";
    redis?: "connected" | "error";
    llm?: "available" | "error";
  };
}

// ============================================
// UTILITY TYPES
// ============================================

/**
 * Converts ISO 8601 string to Date object
 */
export type ParsedDate<T> = {
  [K in keyof T]: T[K] extends string
    ? T[K] extends `${number}-${number}-${number}T${number}:${number}:${number}`
      ? Date
      : T[K]
    : T[K];
};

/**
 * Makes specific fields required
 */
export type RequireFields<T, K extends keyof T> = T & Required<Pick<T, K>>;

/**
 * Makes specific fields optional
 */
export type OptionalFields<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;
```

---

## Usage Examples

```typescript
// ============================================
// USAGE EXAMPLES
// ============================================

// Example 1: Create submission
const submissionRequest: CreateSubmissionRequest = {
  companyName: "Acme Corp",
  challengeCategory: "growth",
  challengeType: "growth_organic",
  businessChallenge: "Need to scale customer acquisition",
  cnpj: "12.345.678/0001-90",
  website: "https://acme.com",
  additionalInfo: JSON.stringify({
    contactName: "John Doe",
    contactEmail: "john@acme.com",
    contactPhone: "+55 11 99999-9999",
    annualRevenueMin: 500000,
    annualRevenueMax: 2000000
  } as AdditionalInfoData)
};

// Example 2: Type-safe framework results
function processSWOT(analysis: AnalysisResponse) {
  const swot = analysis.analysis.swot;
  if (!swot) return;

  // TypeScript knows the structure of SWOTAnalysis
  swot.strengths.forEach(item => {
    console.log(`${item.content} (${item.confidence})`);
  });
}

// Example 3: Admin update analysis
const updateRequest: UpdateAnalysisRequest = {
  swot: {
    strengths: [
      {
        content: "Strong technical team",
        confidence: "Alta",
        source: "fato"
      }
    ]
  }
};

// Example 4: Type guards
function isCompleted(status: AnalysisStatus): status is "completed" {
  return status === "completed";
}

// Example 5: Wizard flow
async function processWizard(analysisId: string) {
  const state: WizardState = await getWizardState(analysisId);

  if (state.status === "pending") {
    await generateStep(analysisId, {
      human_context: "Focus on Brazilian market dynamics"
    });
  } else if (state.status === "generated") {
    // Review output, then approve
    await approveStep(analysisId);
  }
}
```

---

**Schema Version:** 1.0
**Last Updated:** 2025-12-06
**Compatible with API:** v1
**Maintained By:** IMENSIAH Engineering Team
