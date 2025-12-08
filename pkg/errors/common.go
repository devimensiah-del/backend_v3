package errors

// Common errors used across the application

// Validation errors (400)
var (
	ErrValidation = New(
		"ERR_VALIDATION",
		"Validation failed",
		"Falha na validacao",
		400,
	)

	ErrInvalidInput = New(
		"ERR_INVALID_INPUT",
		"Invalid input provided",
		"Entrada invalida fornecida",
		400,
	)

	ErrMissingField = New(
		"ERR_MISSING_FIELD",
		"Required field is missing",
		"Campo obrigatorio ausente",
		400,
	)

	ErrInvalidUUID = New(
		"ERR_INVALID_UUID",
		"Invalid UUID format",
		"Formato de UUID invalido",
		400,
	)
)

// Authentication errors (401)
var (
	ErrUnauthorized = New(
		"ERR_UNAUTHORIZED",
		"Unauthorized access",
		"Acesso nao autorizado",
		401,
	)

	ErrInvalidCredentials = New(
		"ERR_INVALID_CREDENTIALS",
		"Invalid email or password",
		"Email ou senha invalidos",
		401,
	)

	ErrTokenExpired = New(
		"ERR_TOKEN_EXPIRED",
		"Authentication token has expired",
		"Token de autenticacao expirado",
		401,
	)

	ErrTokenInvalid = New(
		"ERR_TOKEN_INVALID",
		"Authentication token is invalid",
		"Token de autenticacao invalido",
		401,
	)
)

// Authorization errors (403)
var (
	ErrForbidden = New(
		"ERR_FORBIDDEN",
		"Access denied",
		"Acesso negado",
		403,
	)

	ErrNotOwner = New(
		"ERR_NOT_OWNER",
		"You do not own this resource",
		"Voce nao e o proprietario deste recurso",
		403,
	)

	ErrAdminRequired = New(
		"ERR_ADMIN_REQUIRED",
		"Administrator privileges required",
		"Privilegios de administrador necessarios",
		403,
	)
)

// Not found errors (404)
var (
	ErrNotFound = New(
		"ERR_NOT_FOUND",
		"Resource not found",
		"Recurso nao encontrado",
		404,
	)

	ErrSubmissionNotFound = New(
		"ERR_SUBMISSION_NOT_FOUND",
		"Submission not found",
		"Submissao nao encontrada",
		404,
	)

	ErrEnrichmentNotFound = New(
		"ERR_ENRICHMENT_NOT_FOUND",
		"Enrichment not found",
		"Enriquecimento nao encontrado",
		404,
	)

	ErrAnalysisNotFound = New(
		"ERR_ANALYSIS_NOT_FOUND",
		"Analysis not found",
		"Analise nao encontrada",
		404,
	)

	ErrCompanyNotFound = New(
		"ERR_COMPANY_NOT_FOUND",
		"Company not found",
		"Empresa nao encontrada",
		404,
	)
)

// Conflict errors (409)
var (
	ErrConflict = New(
		"ERR_CONFLICT",
		"Resource conflict",
		"Conflito de recurso",
		409,
	)

	ErrAlreadyExists = New(
		"ERR_ALREADY_EXISTS",
		"Resource already exists",
		"Recurso ja existe",
		409,
	)
)

// Rate limiting (429)
var (
	ErrRateLimited = New(
		"ERR_RATE_LIMITED",
		"Too many requests, please try again later",
		"Muitas requisicoes, tente novamente mais tarde",
		429,
	)
)

// Server errors (500)
var (
	ErrInternal = New(
		"ERR_INTERNAL",
		"Internal server error",
		"Erro interno do servidor",
		500,
	)

	ErrDatabase = New(
		"ERR_DATABASE",
		"Database error occurred",
		"Ocorreu um erro no banco de dados",
		500,
	)
)

// Service unavailable (503)
var (
	ErrServiceUnavailable = New(
		"ERR_SERVICE_UNAVAILABLE",
		"Service temporarily unavailable",
		"Servico temporariamente indisponivel",
		503,
	)
)
