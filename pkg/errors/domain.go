package errors

// Submission domain errors
var (
	ErrSubmissionInvalid = New(
		"ERR_SUBMISSION_INVALID",
		"Submission data is invalid",
		"Dados da submissao sao invalidos",
		400,
	)
)

// Enrichment domain errors
var (
	ErrEnrichmentPending = New(
		"ERR_ENRICHMENT_PENDING",
		"Enrichment is not ready yet",
		"Enriquecimento ainda nao esta pronto",
		409,
	)

	ErrEnrichmentFailed = New(
		"ERR_ENRICHMENT_FAILED",
		"Enrichment processing failed",
		"Processamento do enriquecimento falhou",
		500,
	)

	ErrEnrichmentApprovalRequired = New(
		"ERR_ENRICHMENT_APPROVAL_REQUIRED",
		"Enrichment requires approval before proceeding",
		"Enriquecimento requer aprovacao antes de prosseguir",
		409,
	)

	ErrEnrichmentAlreadyApproved = New(
		"ERR_ENRICHMENT_ALREADY_APPROVED",
		"Enrichment has already been approved",
		"Enriquecimento ja foi aprovado",
		409,
	)
)

// Analysis domain errors
var (
	ErrAnalysisProcessing = New(
		"ERR_ANALYSIS_PROCESSING",
		"Cannot modify analysis while AI is processing",
		"Nao e possivel modificar a analise enquanto a IA esta processando",
		409,
	)

	ErrAnalysisFailed = New(
		"ERR_ANALYSIS_FAILED",
		"Analysis processing failed",
		"Processamento da analise falhou",
		500,
	)

	ErrFrameworkNotFound = New(
		"ERR_FRAMEWORK_NOT_FOUND",
		"Analysis framework not found",
		"Framework de analise nao encontrado",
		404,
	)

	ErrFrameworkInvalid = New(
		"ERR_FRAMEWORK_INVALID",
		"Invalid framework data",
		"Dados do framework invalidos",
		400,
	)

	ErrAnalysisNotReady = New(
		"ERR_ANALYSIS_NOT_READY",
		"Analysis is not ready yet",
		"Analise ainda nao esta pronta",
		409,
	)
)

// AI/LLM errors
var (
	ErrAIUnavailable = New(
		"ERR_AI_UNAVAILABLE",
		"AI service is temporarily unavailable",
		"Servico de IA temporariamente indisponivel",
		503,
	)

	ErrAIRateLimited = New(
		"ERR_AI_RATE_LIMITED",
		"AI request limit exceeded, please wait",
		"Limite de requisicoes da IA excedido, aguarde",
		429,
	)

	ErrAIResponseInvalid = New(
		"ERR_AI_RESPONSE_INVALID",
		"AI response could not be processed",
		"Resposta da IA nao pode ser processada",
		500,
	)

	ErrAITimeout = New(
		"ERR_AI_TIMEOUT",
		"AI request timed out",
		"Requisicao da IA expirou",
		504,
	)
)

// Company domain errors
var (
	ErrCompanyNotVerified = New(
		"ERR_COMPANY_NOT_VERIFIED",
		"Company has not been verified",
		"Empresa nao foi verificada",
		403,
	)

	ErrCompanyAlreadyExists = New(
		"ERR_COMPANY_ALREADY_EXISTS",
		"A company with this CNPJ already exists",
		"Uma empresa com este CNPJ ja existe",
		409,
	)

	ErrCompanyInvalidCNPJ = New(
		"ERR_COMPANY_INVALID_CNPJ",
		"Invalid CNPJ format",
		"Formato de CNPJ invalido",
		400,
	)
)

// Job/Worker errors
var (
	ErrJobNotFound = New(
		"ERR_JOB_NOT_FOUND",
		"Background job not found",
		"Tarefa em segundo plano nao encontrada",
		404,
	)

	ErrJobFailed = New(
		"ERR_JOB_FAILED",
		"Background job failed",
		"Tarefa em segundo plano falhou",
		500,
	)

	ErrJobAlreadyRunning = New(
		"ERR_JOB_ALREADY_RUNNING",
		"Job is already running",
		"Tarefa ja esta em execucao",
		409,
	)
)
