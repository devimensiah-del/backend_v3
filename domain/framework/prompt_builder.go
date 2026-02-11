package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// PromptMode indicates whether this is an initial generation or an update
type PromptMode string

const (
	PromptModeInitial PromptMode = "initial"
	PromptModeUpdate  PromptMode = "update"
)

// PromptBuilder constructs prompts with dependency context and current result injection
type PromptBuilder struct {
	repo       Repository
	depService *DependencyService
	logger     zerolog.Logger
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder(repo Repository, depService *DependencyService, logger zerolog.Logger) *PromptBuilder {
	return &PromptBuilder{
		repo:       repo,
		depService: depService,
		logger:     logger.With().Str("service", "prompt_builder").Logger(),
	}
}

// PromptContext holds all the context needed to build a prompt
type PromptContext struct {
	CompanyID      uuid.UUID
	FrameworkCode  string
	ChallengeID    *uuid.UUID
	Mode           PromptMode
	CompanyData    json.RawMessage // Enriched company data (EMPRESA)
	ChallengeData  json.RawMessage // Business challenge data (NEGOCIO/challenge)
	CurrentResult  json.RawMessage // Existing result (for update mode)
	UserRefinement *string         // Optional user feedback/refinement request
}

// BuildPromptResult contains the constructed prompts
type BuildPromptResult struct {
	SystemPrompt string          `json:"system_prompt"`
	UserPrompt   string          `json:"user_prompt"`
	Mode         PromptMode      `json:"mode"`
	Context      json.RawMessage `json:"context,omitempty"` // JSON of all injected context
}

// BuildPrompt constructs the complete prompt for a framework execution
func (b *PromptBuilder) BuildPrompt(ctx context.Context, pc *PromptContext) (*BuildPromptResult, error) {
	// Get framework definition
	fw, err := b.repo.GetByCode(ctx, pc.FrameworkCode)
	if err != nil || fw == nil {
		return nil, fmt.Errorf("framework não encontrado: %s", pc.FrameworkCode)
	}

	// Get dependency context (all upstream framework results)
	depContext, err := b.depService.GetDependencyContext(ctx, pc.CompanyID, pc.FrameworkCode, pc.ChallengeID)
	if err != nil {
		b.logger.Warn().Err(err).Msg("Falha ao obter contexto de dependências")
		depContext = make(map[string]json.RawMessage)
	}

	// Build the context object
	contextData := b.buildContextData(pc, depContext)

	// Determine which prompt template to use
	var userPrompt string
	if pc.Mode == PromptModeUpdate && fw.PromptUpdate != nil && *fw.PromptUpdate != "" {
		userPrompt = *fw.PromptUpdate
	} else {
		userPrompt = fw.PromptUser
	}

	// Build system prompt
	systemPrompt := b.buildSystemPrompt(fw, pc.Mode)

	// Replace placeholders in user prompt
	userPrompt = b.replacePlaceholders(userPrompt, pc, depContext)

	// If update mode with current result, inject it
	if pc.Mode == PromptModeUpdate && pc.CurrentResult != nil {
		userPrompt = b.injectCurrentResult(userPrompt, pc.CurrentResult, pc.UserRefinement)
	}

	return &BuildPromptResult{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Mode:         pc.Mode,
		Context:      contextData,
	}, nil
}

// buildSystemPrompt constructs the system prompt based on mode
func (b *PromptBuilder) buildSystemPrompt(fw *Framework, mode PromptMode) string {
	var sb strings.Builder

	// Base system prompt from framework
	if fw.PromptSystem != nil && *fw.PromptSystem != "" {
		sb.WriteString(*fw.PromptSystem)
	} else {
		sb.WriteString("Você é um consultor de estratégia empresarial especializado em análises profundas e acionáveis.")
	}

	sb.WriteString("\n\n")

	// Mode-specific instructions
	if mode == PromptModeUpdate {
		sb.WriteString(`## MODO DE ATUALIZAÇÃO

Você está ATUALIZANDO uma análise existente, não criando uma nova do zero.

REGRAS IMPORTANTES:
1. PRESERVE informações válidas e insights úteis da análise anterior
2. ATUALIZE apenas o que mudou com base nos novos dados de contexto
3. ADICIONE novos insights se os dados de entrada justificarem
4. REMOVA apenas informações que se tornaram incorretas ou obsoletas
5. MANTENHA o nível de detalhe e qualidade da análise anterior
6. Se o usuário forneceu feedback específico, priorize essas mudanças

O objetivo é REFINAR e MELHORAR, não recomeçar do zero.`)
	} else {
		sb.WriteString(`## MODO INICIAL

Você está gerando uma análise NOVA e COMPLETA.

REQUISITOS:
1. Use TODOS os dados de contexto fornecidos
2. Seja específico e acionável - evite generalizações
3. Baseie cada insight em evidências dos dados
4. Mantenha consistência com análises anteriores da cadeia`)
	}

	sb.WriteString("\n\n")

	// JSON output requirements
	sb.WriteString(`## FORMATO DE SAÍDA

Responda APENAS com um objeto JSON válido, sem texto adicional.
O JSON deve seguir EXATAMENTE o schema fornecido.
Todos os campos devem ser preenchidos com conteúdo relevante e específico.
Use português brasileiro em todo o conteúdo.`)

	return sb.String()
}

// buildContextData creates the context JSON for logging/debugging
func (b *PromptBuilder) buildContextData(pc *PromptContext, depContext map[string]json.RawMessage) json.RawMessage {
	contextMap := make(map[string]interface{})

	contextMap["mode"] = pc.Mode
	contextMap["framework"] = pc.FrameworkCode

	if pc.CompanyData != nil {
		contextMap["company_data"] = json.RawMessage(pc.CompanyData)
	}

	if pc.ChallengeData != nil {
		contextMap["challenge_data"] = json.RawMessage(pc.ChallengeData)
	}

	if len(depContext) > 0 {
		deps := make(map[string]json.RawMessage)
		for k, v := range depContext {
			deps[k] = v
		}
		contextMap["dependencies"] = deps
	}

	if pc.CurrentResult != nil {
		contextMap["current_result"] = json.RawMessage(pc.CurrentResult)
	}

	data, _ := json.Marshal(contextMap)
	return data
}

// replacePlaceholders replaces template placeholders with actual data
func (b *PromptBuilder) replacePlaceholders(prompt string, pc *PromptContext, depContext map[string]json.RawMessage) string {
	result := prompt

	// Replace {{EMPRESA}} with company data
	if pc.CompanyData != nil {
		result = strings.ReplaceAll(result, "{{EMPRESA}}", string(pc.CompanyData))
		result = strings.ReplaceAll(result, "{{empresa}}", string(pc.CompanyData))
		result = strings.ReplaceAll(result, "{{COMPANY}}", string(pc.CompanyData))
		result = strings.ReplaceAll(result, "{{company_data}}", string(pc.CompanyData))
	}

	// Replace {{NEGOCIO}} / {{DESAFIO}} with challenge data
	if pc.ChallengeData != nil {
		result = strings.ReplaceAll(result, "{{NEGOCIO}}", string(pc.ChallengeData))
		result = strings.ReplaceAll(result, "{{negocio}}", string(pc.ChallengeData))
		result = strings.ReplaceAll(result, "{{DESAFIO}}", string(pc.ChallengeData))
		result = strings.ReplaceAll(result, "{{desafio}}", string(pc.ChallengeData))
		result = strings.ReplaceAll(result, "{{CHALLENGE}}", string(pc.ChallengeData))
		result = strings.ReplaceAll(result, "{{challenge_data}}", string(pc.ChallengeData))
	}

	// Replace dependency placeholders {{FRAMEWORK_CODE}}
	for code, data := range depContext {
		placeholder := fmt.Sprintf("{{%s}}", code)
		result = strings.ReplaceAll(result, placeholder, string(data))
		// Also try lowercase
		placeholderLower := fmt.Sprintf("{{%s}}", strings.ToLower(code))
		result = strings.ReplaceAll(result, placeholderLower, string(data))
	}

	return result
}

// injectCurrentResult adds the current result and refinement request to the prompt
func (b *PromptBuilder) injectCurrentResult(prompt string, currentResult json.RawMessage, refinement *string) string {
	var sb strings.Builder

	sb.WriteString(prompt)
	sb.WriteString("\n\n")
	sb.WriteString("## ANÁLISE ATUAL (para atualização)\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(string(currentResult))
	sb.WriteString("\n```\n\n")

	if refinement != nil && *refinement != "" {
		sb.WriteString("## FEEDBACK DO USUÁRIO\n\n")
		sb.WriteString("O usuário solicitou as seguintes alterações ou refinamentos:\n\n")
		sb.WriteString(*refinement)
		sb.WriteString("\n\n")
		sb.WriteString("Por favor, incorpore este feedback na atualização da análise.\n")
	} else {
		sb.WriteString("INSTRUÇÃO: Atualize esta análise considerando os dados de contexto mais recentes.\n")
		sb.WriteString("Preserve os insights válidos e atualize apenas o que mudou.\n")
	}

	return sb.String()
}

// BuildInitialPrompt is a convenience method for initial mode
func (b *PromptBuilder) BuildInitialPrompt(ctx context.Context, companyID uuid.UUID, frameworkCode string, companyData, challengeData json.RawMessage, challengeID *uuid.UUID) (*BuildPromptResult, error) {
	return b.BuildPrompt(ctx, &PromptContext{
		CompanyID:     companyID,
		FrameworkCode: frameworkCode,
		ChallengeID:   challengeID,
		Mode:          PromptModeInitial,
		CompanyData:   companyData,
		ChallengeData: challengeData,
	})
}

// BuildUpdatePrompt is a convenience method for update mode
func (b *PromptBuilder) BuildUpdatePrompt(ctx context.Context, companyID uuid.UUID, frameworkCode string, companyData, challengeData, currentResult json.RawMessage, refinement *string, challengeID *uuid.UUID) (*BuildPromptResult, error) {
	return b.BuildPrompt(ctx, &PromptContext{
		CompanyID:      companyID,
		FrameworkCode:  frameworkCode,
		ChallengeID:    challengeID,
		Mode:           PromptModeUpdate,
		CompanyData:    companyData,
		ChallengeData:  challengeData,
		CurrentResult:  currentResult,
		UserRefinement: refinement,
	})
}

// =============================================================================
// Framework Execution Service
// =============================================================================

// ExecutionService handles framework execution with dependency management
type ExecutionService struct {
	repo          Repository
	depService    *DependencyService
	promptBuilder *PromptBuilder
	logger        zerolog.Logger
}

// NewExecutionService creates a new execution service
func NewExecutionService(repo Repository, depService *DependencyService, promptBuilder *PromptBuilder, logger zerolog.Logger) *ExecutionService {
	return &ExecutionService{
		repo:          repo,
		depService:    depService,
		promptBuilder: promptBuilder,
		logger:        logger.With().Str("service", "execution").Logger(),
	}
}

// ExecuteRequest holds the request for framework execution
type ExecuteRequest struct {
	CompanyID     uuid.UUID       `json:"company_id"`
	FrameworkCode string          `json:"framework_code"`
	ChallengeID   *uuid.UUID      `json:"challenge_id,omitempty"`
	CompanyData   json.RawMessage `json:"company_data"`
	ChallengeData json.RawMessage `json:"challenge_data,omitempty"`
	ForceRerun    bool            `json:"force_rerun"` // Ignore stale warnings
	Refinement    *string         `json:"refinement,omitempty"`
}

// ExecuteResult holds the execution result
type ExecuteResult struct {
	ResultID        uuid.UUID       `json:"result_id"`
	FrameworkCode   string          `json:"framework_code"`
	Mode            PromptMode      `json:"mode"`
	Status          string          `json:"status"`
	Result          json.RawMessage `json:"result,omitempty"`
	PreviousVersion *int            `json:"previous_version,omitempty"`
	NewVersion      int             `json:"new_version"`
	StaleCount      int             `json:"stale_count"` // Downstream frameworks marked stale
}

// PrepareExecution prepares a framework for execution, returning the prompt and mode
func (e *ExecutionService) PrepareExecution(ctx context.Context, req *ExecuteRequest) (*BuildPromptResult, error) {
	// Check if ready to run (all dependencies completed)
	readiness, err := e.depService.CheckReadyToRun(ctx, req.CompanyID, req.FrameworkCode, req.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("falha ao verificar dependências: %w", err)
	}

	if !readiness.Ready {
		return nil, fmt.Errorf("dependências incompletas: %v", readiness.MissingDependencies)
	}

	// Get framework
	fw, err := e.repo.GetByCode(ctx, req.FrameworkCode)
	if err != nil || fw == nil {
		return nil, fmt.Errorf("framework não encontrado: %s", req.FrameworkCode)
	}

	// Check for existing result
	existingResult, _ := e.repo.GetCurrentResult(ctx, req.CompanyID, fw.ID, req.ChallengeID)

	// Determine mode
	var mode PromptMode
	var currentResult json.RawMessage

	if existingResult != nil && existingResult.IsCompleted() && existingResult.Result != nil {
		mode = PromptModeUpdate
		currentResult = existingResult.Result
	} else {
		mode = PromptModeInitial
	}

	// Build prompt
	promptCtx := &PromptContext{
		CompanyID:      req.CompanyID,
		FrameworkCode:  req.FrameworkCode,
		ChallengeID:    req.ChallengeID,
		Mode:           mode,
		CompanyData:    req.CompanyData,
		ChallengeData:  req.ChallengeData,
		CurrentResult:  currentResult,
		UserRefinement: req.Refinement,
	}

	return e.promptBuilder.BuildPrompt(ctx, promptCtx)
}

// FinalizeExecution saves the result and triggers stale cascade
func (e *ExecutionService) FinalizeExecution(ctx context.Context, req *ExecuteRequest, result json.RawMessage) (*ExecuteResult, error) {
	fw, err := e.repo.GetByCode(ctx, req.FrameworkCode)
	if err != nil || fw == nil {
		return nil, fmt.Errorf("framework não encontrado: %s", req.FrameworkCode)
	}

	// Get existing result for versioning
	existingResult, _ := e.repo.GetCurrentResult(ctx, req.CompanyID, fw.ID, req.ChallengeID)

	var previousVersion *int
	newVersion := 1

	if existingResult != nil {
		// Save previous result for comparison
		if existingResult.Result != nil {
			if err := e.repo.SavePreviousResult(ctx, existingResult.ID, existingResult.Result, existingResult.Version); err != nil {
				e.logger.Warn().Err(err).Msg("Falha ao salvar resultado anterior")
			}
		}

		previousVersion = &existingResult.Version
		newVersion = existingResult.Version + 1

		// Invalidate old result
		if err := e.repo.InvalidateResults(ctx, req.CompanyID, fw.ID); err != nil {
			e.logger.Warn().Err(err).Msg("Falha ao invalidar resultados anteriores")
		}

		// Clear stale status if it was stale
		if existingResult.IsStale {
			if err := e.depService.ClearStaleStatus(ctx, existingResult.ID); err != nil {
				e.logger.Warn().Err(err).Msg("Falha ao limpar status stale")
			}
		}
	}

	// Compute upstream hash for change detection
	upstreamHash, _ := e.depService.ComputeUpstreamHash(ctx, req.CompanyID, req.FrameworkCode, req.ChallengeID)

	// Create new result
	newResult := &CompanyFrameworkResult{
		ID:               uuid.New(),
		CompanyID:        req.CompanyID,
		FrameworkID:      fw.ID,
		ChallengeID:      req.ChallengeID,
		Result:           result,
		Status:           StatusCompleted,
		Version:          newVersion,
		IsCurrent:        true,
		UpstreamDataHash: &upstreamHash,
	}

	if err := e.repo.CreateResult(ctx, newResult); err != nil {
		return nil, fmt.Errorf("falha ao salvar resultado: %w", err)
	}

	// Mark downstream frameworks as stale
	staleCount, err := e.depService.MarkDownstreamAsStale(ctx, req.CompanyID, req.FrameworkCode)
	if err != nil {
		e.logger.Warn().Err(err).Msg("Falha ao marcar downstream como stale")
	}

	mode := PromptModeInitial
	if previousVersion != nil {
		mode = PromptModeUpdate
	}

	return &ExecuteResult{
		ResultID:        newResult.ID,
		FrameworkCode:   req.FrameworkCode,
		Mode:            mode,
		Status:          StatusCompleted,
		Result:          result,
		PreviousVersion: previousVersion,
		NewVersion:      newVersion,
		StaleCount:      staleCount,
	}, nil
}

// =============================================================================
// Update Suggestion Service
// =============================================================================

// CompareResults compares the previous and new results to generate a summary
func (e *ExecutionService) CompareResults(previousResult, newResult json.RawMessage) (*UpdateSuggestion, error) {
	// Parse both results
	var prev, curr map[string]interface{}

	if err := json.Unmarshal(previousResult, &prev); err != nil {
		return nil, fmt.Errorf("falha ao parsear resultado anterior: %w", err)
	}

	if err := json.Unmarshal(newResult, &curr); err != nil {
		return nil, fmt.Errorf("falha ao parsear novo resultado: %w", err)
	}

	// Compare and find changes
	changes := e.findChanges("", prev, curr)

	summary := e.generateChangeSummary(changes)

	return &UpdateSuggestion{
		SuggestedResult: newResult,
		ChangesSummary:  summary,
		FieldChanges:    changes,
	}, nil
}

// findChanges recursively finds changes between two maps
func (e *ExecutionService) findChanges(prefix string, prev, curr map[string]interface{}) []FieldChange {
	var changes []FieldChange

	// Check for modified and new fields
	for key, currVal := range curr {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		prevVal, exists := prev[key]
		if !exists {
			changes = append(changes, FieldChange{
				FieldPath:    path,
				OldValue:     "",
				NewValue:     formatValue(currVal),
				ChangeReason: "campo adicionado",
			})
			continue
		}

		// Compare values
		if !valuesEqual(prevVal, currVal) {
			// If both are maps, recurse
			prevMap, prevIsMap := prevVal.(map[string]interface{})
			currMap, currIsMap := currVal.(map[string]interface{})
			if prevIsMap && currIsMap {
				subChanges := e.findChanges(path, prevMap, currMap)
				changes = append(changes, subChanges...)
			} else {
				changes = append(changes, FieldChange{
					FieldPath:    path,
					OldValue:     formatValue(prevVal),
					NewValue:     formatValue(currVal),
					ChangeReason: "valor alterado",
				})
			}
		}
	}

	// Check for removed fields
	for key, prevVal := range prev {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		if _, exists := curr[key]; !exists {
			changes = append(changes, FieldChange{
				FieldPath:    path,
				OldValue:     formatValue(prevVal),
				NewValue:     "",
				ChangeReason: "campo removido",
			})
		}
	}

	return changes
}

// generateChangeSummary creates a human-readable summary of changes
func (e *ExecutionService) generateChangeSummary(changes []FieldChange) string {
	if len(changes) == 0 {
		return "Nenhuma alteração detectada."
	}

	added := 0
	modified := 0
	removed := 0

	for _, c := range changes {
		switch c.ChangeReason {
		case "campo adicionado":
			added++
		case "valor alterado":
			modified++
		case "campo removido":
			removed++
		}
	}

	var parts []string
	if added > 0 {
		parts = append(parts, fmt.Sprintf("%d campos adicionados", added))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d campos modificados", modified))
	}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("%d campos removidos", removed))
	}

	return strings.Join(parts, ", ") + "."
}

// Helper functions

func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		if len(val) > 100 {
			return val[:100] + "..."
		}
		return val
	case []interface{}:
		return fmt.Sprintf("[%d itens]", len(val))
	case map[string]interface{}:
		return fmt.Sprintf("{%d campos}", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}

func valuesEqual(a, b interface{}) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)

	if err1 != nil || err2 != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}
