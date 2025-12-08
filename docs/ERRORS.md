# Error Handling Guide

**Complete error reference for frontend developers**

This document describes all error types, codes, messages, and recommended handling strategies.

---

## Table of Contents

- [Error Response Format](#error-response-format)
- [HTTP Status Codes](#http-status-codes)
- [Error Categories](#error-categories)
- [Common Errors](#common-errors)
- [Handling Strategies](#handling-strategies)
- [User-Friendly Messages](#user-friendly-messages)

---

## Error Response Format

All API errors return JSON in this format:

```json
{
  "error": "Error Category",
  "message": "Human-readable error description"
}
```

### Example Responses

```json
// 400 Bad Request
{
  "error": "Invalid request",
  "message": "companyName is required"
}

// 401 Unauthorized
{
  "error": "Unauthorized",
  "message": "Token not provided or invalid"
}

// 403 Forbidden
{
  "error": "Forbidden",
  "message": "You don't have permission to access this resource"
}

// 404 Not Found
{
  "error": "Not found",
  "message": "Submission not found"
}

// 500 Internal Server Error
{
  "error": "Internal error",
  "message": "Failed to process request"
}
```

---

## HTTP Status Codes

### Success Codes

| Code | Name | Usage |
|------|------|-------|
| `200` | OK | Successful GET, PUT, DELETE |
| `201` | Created | Successful POST (resource created) |
| `202` | Accepted | Request accepted for processing (async) |

### Client Error Codes

| Code | Name | Usage |
|------|------|-------|
| `400` | Bad Request | Invalid request body, validation failure |
| `401` | Unauthorized | Missing or invalid authentication |
| `403` | Forbidden | Valid auth but insufficient permissions |
| `404` | Not Found | Resource doesn't exist |
| `429` | Too Many Requests | Rate limit exceeded |

### Server Error Codes

| Code | Name | Usage |
|------|------|-------|
| `500` | Internal Server Error | Unexpected server error |
| `502` | Bad Gateway | External service failure (LLM, Perplexity) |
| `503` | Service Unavailable | Temporary service outage |
| `504` | Gateway Timeout | External service timeout |

---

## Error Categories

### Validation Errors (400)

**Error:** `Invalid request`

**Common Messages:**
- "companyName is required"
- "challengeCategory must be one of: growth, transform, transition, compete, funding"
- "email format is invalid"
- "password must be at least 6 characters"
- "cnpj format is invalid"

**Frontend Handling:**
```typescript
if (error.error === "Invalid request") {
  // Show field-specific errors
  const fieldErrors = parseValidationMessage(error.message);
  displayFormErrors(fieldErrors);
}
```

---

### Authentication Errors (401)

**Error:** `Unauthorized`

**Common Messages:**
- "Token not provided"
- "Token invalid or expired"
- "Invalid credentials"
- "E-mail ou senha inválidos"

**Frontend Handling:**
```typescript
if (response.status === 401) {
  // Clear auth state
  clearAuthToken();

  // Redirect to login
  navigateTo('/login');

  // Show message
  toast.error('Sessão expirada. Por favor, faça login novamente.');
}
```

---

### Authorization Errors (403)

**Error:** `Forbidden`

**Common Messages:**
- "You don't have permission to access this resource"
- "Admin access required"
- "You don't have permission to access this submission"
- "Acesso negado - não é admin ou usuário autorizado"

**Frontend Handling:**
```typescript
if (response.status === 403) {
  // Show error message
  toast.error('Você não tem permissão para acessar este recurso');

  // Redirect to safe page
  navigateTo('/dashboard');
}
```

---

### Not Found Errors (404)

**Error:** `Not found`

**Common Messages:**
- "Submission not found"
- "Analysis not found"
- "Company not found"
- "Relatório não encontrado"
- "O código de acesso é inválido ou o relatório não está mais disponível"

**Frontend Handling:**
```typescript
if (response.status === 404) {
  if (isPublicReport) {
    // Show user-friendly message
    showError('Relatório não encontrado. Verifique o código de acesso.');
  } else {
    // Navigate to 404 page
    navigateTo('/404');
  }
}
```

---

### Rate Limit Errors (429)

**Error:** `Rate limit exceeded`

**Common Messages:**
- "Too many requests. Please wait 60 seconds before retrying."
- "Rate limit: 100 requests per minute exceeded"

**Response Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1733486460
```

**Frontend Handling:**
```typescript
if (response.status === 429) {
  const resetTime = response.headers.get('X-RateLimit-Reset');
  const waitSeconds = resetTime ? (parseInt(resetTime) - Date.now() / 1000) : 60;

  toast.error(`Muitas requisições. Aguarde ${Math.ceil(waitSeconds)} segundos.`);

  // Optional: Auto-retry after reset
  setTimeout(() => retryRequest(), waitSeconds * 1000);
}
```

---

### Server Errors (500+)

**Error:** `Internal error`, `Service unavailable`, `Gateway timeout`

**Common Messages:**
- "Failed to process request"
- "Database connection failed"
- "External service unavailable"
- "LLM API timeout"
- "Perplexity API error"

**Frontend Handling:**
```typescript
if (response.status >= 500) {
  // Log to monitoring service
  logError(error);

  // Show generic error (don't expose internals)
  toast.error('Erro no servidor. Tente novamente em alguns instantes.');

  // Optional: Retry with exponential backoff
  if (isRetryable(error)) {
    retryWithBackoff(request, 3);
  }
}
```

---

## Common Errors

### Authentication & Authorization

#### Invalid Login Credentials
```json
{
  "error": "Unauthorized",
  "message": "E-mail ou senha inválidos"
}
```
**Trigger:** POST /api/v1/auth/login with wrong credentials
**Fix:** User enters correct credentials

#### Token Expired
```json
{
  "error": "Unauthorized",
  "message": "Token expired"
}
```
**Trigger:** JWT token expired (1 hour default)
**Fix:** User logs in again

#### Missing Authorization Header
```json
{
  "error": "Unauthorized",
  "message": "Token not provided"
}
```
**Trigger:** Protected endpoint called without `Authorization` header
**Fix:** Include `Authorization: Bearer <token>` header

#### Insufficient Permissions
```json
{
  "error": "Forbidden",
  "message": "Admin access required"
}
```
**Trigger:** User tries to access admin endpoint
**Fix:** User needs admin role

---

### Submission Errors

#### Missing Required Field
```json
{
  "error": "Invalid request",
  "message": "companyName is required"
}
```
**Trigger:** POST /api/v1/submissions without `companyName`
**Fix:** Include all required fields

#### Invalid Challenge Category
```json
{
  "error": "Invalid request",
  "message": "challengeCategory must be one of: growth, transform, transition, compete, funding"
}
```
**Trigger:** Invalid `challengeCategory` value
**Fix:** Use valid category from enum

#### Invalid Challenge Type
```json
{
  "error": "Invalid request",
  "message": "challengeType 'invalid_type' is not valid for category 'growth'"
}
```
**Trigger:** Challenge type doesn't match category
**Fix:** Use valid type for selected category

#### Contact Email Required
```json
{
  "error": "Invalid request",
  "message": "contact_email is required in additionalInfo"
}
```
**Trigger:** Missing `contactEmail` in `additionalInfo` JSON
**Fix:** Include contact email

---

### Company Errors

#### Company Not Found
```json
{
  "error": "Not found",
  "message": "Company not found"
}
```
**Trigger:** GET /api/v1/companies/:id with invalid ID
**Fix:** Verify company ID exists

#### Enrichment Not Completed
```json
{
  "error": "Company required",
  "message": "Cannot run analysis without company data. Create company first."
}
```
**Trigger:** Retry analysis before enrichment completes
**Fix:** Wait for enrichment to finish

#### Enrichment Failed
```json
{
  "error": "Enrichment failed",
  "message": "Perplexity API rate limit exceeded"
}
```
**Trigger:** Perplexity API error during enrichment
**Fix:** Admin retries or contacts support

---

### Analysis Errors

#### Analysis Not Found
```json
{
  "error": "Not found",
  "message": "Analysis not found for this submission"
}
```
**Trigger:** GET /api/v1/submissions/:id/analysis before analysis created
**Fix:** Start wizard first

#### Not Visible to User
```json
{
  "error": "Not found",
  "message": "Relatório não encontrado. O código de acesso é inválido ou o relatório não está mais disponível."
}
```
**Trigger:** Access report with `is_visible_to_user: false`
**Fix:** Admin makes analysis visible

#### Authentication Required (Private Report)
```json
{
  "error": "Authentication required",
  "message": "Este relatório requer autenticação. Por favor, faça login para acessar."
}
```
**Trigger:** Anonymous user tries to access report with `is_public: false`
**Fix:** User logs in

---

### Wizard Errors

#### Wizard Not Started
```json
{
  "error": "Not found",
  "message": "Wizard not found for this analysis"
}
```
**Trigger:** GET /api/v1/analyses/:id/wizard before starting wizard
**Fix:** POST /api/v1/wizard/start with company_id and challenge_id first

#### Step Generation Failed
```json
{
  "error": "Generation failed",
  "message": "LLM API timeout on framework: pestel"
}
```
**Trigger:** OpenRouter API timeout during generation
**Fix:** Retry generation

#### Invalid Step Transition
```json
{
  "error": "Invalid operation",
  "message": "Cannot approve step: output not generated yet"
}
```
**Trigger:** Approve step before generating output
**Fix:** Generate step output first

---

### Framework Errors

#### Framework Not Found
```json
{
  "error": "Not found",
  "message": "Framework not found"
}
```
**Trigger:** GET /api/v1/frameworks/:code with invalid code
**Fix:** Use valid framework code

#### Framework Inactive
```json
{
  "error": "Framework inactive",
  "message": "Framework 'old_framework' is no longer active"
}
```
**Trigger:** Reference inactive framework
**Fix:** Use active framework

---

## Handling Strategies

### Retry Logic

```typescript
async function fetchWithRetry(
  url: string,
  options: RequestInit,
  maxRetries = 3
): Promise<Response> {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch(url, options);

      // Don't retry 4xx errors (client errors)
      if (response.status >= 400 && response.status < 500) {
        return response;
      }

      // Retry 5xx errors
      if (response.status >= 500) {
        const delay = Math.pow(2, i) * 1000; // Exponential backoff
        await sleep(delay);
        continue;
      }

      return response;
    } catch (error) {
      // Network error, retry
      if (i === maxRetries - 1) throw error;
      await sleep(Math.pow(2, i) * 1000);
    }
  }

  throw new Error('Max retries exceeded');
}
```

### Error Handling Wrapper

```typescript
async function apiCall<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  try {
    const response = await fetch(`${API_URL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${getToken()}`,
        ...options?.headers
      }
    });

    // Handle success
    if (response.ok) {
      return await response.json();
    }

    // Handle errors
    const error = await response.json();

    switch (response.status) {
      case 400:
        throw new ValidationError(error.message);
      case 401:
        handleAuthError();
        throw new AuthError(error.message);
      case 403:
        throw new PermissionError(error.message);
      case 404:
        throw new NotFoundError(error.message);
      case 429:
        handleRateLimitError(response);
        throw new RateLimitError(error.message);
      case 500:
      default:
        throw new ServerError(error.message);
    }
  } catch (error) {
    if (error instanceof TypeError) {
      // Network error
      throw new NetworkError('Erro de conexão. Verifique sua internet.');
    }
    throw error;
  }
}
```

### Global Error Handler

```typescript
// React example
function App() {
  useEffect(() => {
    window.addEventListener('unhandledrejection', (event) => {
      const error = event.reason;

      // Log to monitoring
      logError(error);

      // Show user-friendly message
      if (error instanceof NetworkError) {
        toast.error('Erro de conexão. Verifique sua internet.');
      } else if (error instanceof AuthError) {
        toast.error('Sessão expirada. Faça login novamente.');
        navigateTo('/login');
      } else if (error instanceof ServerError) {
        toast.error('Erro no servidor. Tente novamente.');
      } else {
        toast.error('Ocorreu um erro inesperado.');
      }
    });
  }, []);

  return <AppContent />;
}
```

---

## User-Friendly Messages

### Mapping Technical Errors to User Messages

```typescript
const ERROR_MESSAGES = {
  // Authentication
  'Invalid credentials': 'E-mail ou senha incorretos. Tente novamente.',
  'Token expired': 'Sua sessão expirou. Por favor, faça login novamente.',
  'Token not provided': 'Você precisa estar logado para acessar esta página.',

  // Validation
  'companyName is required': 'Nome da empresa é obrigatório.',
  'email format is invalid': 'E-mail inválido. Verifique o formato.',
  'password must be at least 6 characters': 'A senha deve ter pelo menos 6 caracteres.',

  // Not Found
  'Submission not found': 'Submissão não encontrada. Verifique o link.',
  'Analysis not found': 'Análise não encontrada.',
  'Company not found': 'Empresa não encontrada.',

  // Enrichment
  'Enrichment not completed': 'Aguardando enriquecimento de dados. Por favor, aguarde.',
  'Enrichment failed': 'Falha ao enriquecer dados. Entre em contato com suporte.',

  // Analysis
  'Cannot run analysis': 'Não é possível iniciar análise. Verifique os dados da empresa.',
  'Analysis failed': 'Falha na análise. Tente novamente ou entre em contato com suporte.',

  // Permissions
  'Forbidden': 'Você não tem permissão para acessar este recurso.',
  'Admin access required': 'Esta ação requer permissões de administrador.',

  // Rate Limit
  'Rate limit exceeded': 'Muitas requisições. Aguarde alguns segundos e tente novamente.',

  // Server
  'Internal error': 'Erro no servidor. Tente novamente em alguns instantes.',
  'Service unavailable': 'Serviço temporariamente indisponível. Tente novamente mais tarde.',
  'Gateway timeout': 'Tempo de resposta excedido. Tente novamente.'
};

function getUserMessage(technicalMessage: string): string {
  return ERROR_MESSAGES[technicalMessage] || 'Ocorreu um erro inesperado. Tente novamente.';
}
```

### Portuguese Error Messages (Built-in)

The API already returns some errors in Portuguese:

```json
// Login failure
{
  "error": "Unauthorized",
  "message": "E-mail ou senha inválidos"
}

// Signup duplicate email
{
  "error": "Bad Request",
  "message": "Este e-mail já está cadastrado"
}

// Weak password
{
  "error": "Bad Request",
  "message": "Senha muito fraca. Use pelo menos 6 caracteres"
}

// Password update success
{
  "message": "Senha atualizada com sucesso"
}

// Logout success
{
  "message": "Logout realizado com sucesso"
}

// Public report not found
{
  "error": "Not found",
  "message": "Relatório não encontrado. O código de acesso é inválido ou o relatório não está mais disponível."
}

// Public report requires auth
{
  "error": "Authentication required",
  "message": "Este relatório requer autenticação. Por favor, faça login para acessar."
}
```

---

## Testing Error Scenarios

### Manual Testing

```bash
# Test 401 Unauthorized
curl -X GET http://localhost:8080/api/v1/submissions \
  -H "Authorization: Bearer invalid_token"

# Test 400 Bad Request
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{"companyName": ""}'

# Test 404 Not Found
curl -X GET http://localhost:8080/api/v1/submissions/invalid-uuid

# Test 429 Rate Limit
for i in {1..150}; do
  curl -X GET http://localhost:8080/api/v1/frameworks
done
```

### Frontend Error Handling Tests

```typescript
// Test suite for error handling
describe('API Error Handling', () => {
  it('should redirect to login on 401', async () => {
    mockFetch(401, { error: 'Unauthorized', message: 'Token expired' });

    await apiCall('/submissions');

    expect(navigateTo).toHaveBeenCalledWith('/login');
    expect(toast.error).toHaveBeenCalledWith('Sessão expirada. Faça login novamente.');
  });

  it('should show validation errors on 400', async () => {
    mockFetch(400, { error: 'Invalid request', message: 'companyName is required' });

    try {
      await createSubmission({});
    } catch (error) {
      expect(error).toBeInstanceOf(ValidationError);
      expect(error.message).toBe('companyName is required');
    }
  });

  it('should retry on 500 error', async () => {
    mockFetch(500, { error: 'Internal error' })
      .mockFetchOnce(500, { error: 'Internal error' })
      .mockFetchOnce(200, { data: 'success' });

    const result = await fetchWithRetry('/api/endpoint');

    expect(fetch).toHaveBeenCalledTimes(3);
    expect(result).toEqual({ data: 'success' });
  });
});
```

---

## Error Monitoring

### Recommended Monitoring Setup

```typescript
// Sentry integration example
import * as Sentry from '@sentry/react';

Sentry.init({
  dsn: process.env.REACT_APP_SENTRY_DSN,
  environment: process.env.NODE_ENV,
  integrations: [
    new Sentry.BrowserTracing(),
    new Sentry.Replay()
  ],
  tracesSampleRate: 0.1,
  replaysSessionSampleRate: 0.1,
  replaysOnErrorSampleRate: 1.0
});

// Capture API errors
function logError(error: Error, context?: Record<string, any>) {
  Sentry.captureException(error, {
    contexts: {
      api: context
    }
  });
}

// Use in API wrapper
async function apiCall(endpoint: string) {
  try {
    return await fetch(endpoint);
  } catch (error) {
    logError(error, {
      endpoint,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent
    });
    throw error;
  }
}
```

---

**Error Documentation Version:** 1.0
**Last Updated:** 2025-12-06
**Compatible with API:** v1
**Maintained By:** IMENSIAH Engineering Team
