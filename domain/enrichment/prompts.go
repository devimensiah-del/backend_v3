package enrichment

import (
	"fmt"
	"strings"
)

// =============================================================================
// MODEL CONFIGURATION
// =============================================================================

const (
	// Stage 1: Perplexity for web search (ONLY model with real-time data)
	ModelSearch         = "perplexity/sonar-pro"
	ModelSearchFallback = "perplexity/sonar"

	// Stage 2: Gemini 3 Flash for JSON formatting (replaces Claude)
	// Fast, cheap, reliable JSON output
	ModelFormat         = "google/gemini-3-flash-preview"
	ModelFormatFallback = "google/gemini-2.0-flash-001"
)

// =============================================================================
// STEP 1: BASIC INFO PROMPTS
// Fields: CNPJ, razão social, fundação, sede, funcionários, website, redes sociais, executivos
// =============================================================================

// Step1SearchSystemPrompt is the system prompt for Perplexity Step 1 search
const Step1SearchSystemPrompt = `Você é um pesquisador de inteligência de mercado.
Foco: Encontrar dados básicos e públicos sobre empresas brasileiras.

INSTRUÇÕES:
1. Busque informações públicas verificáveis
2. Inclua dados de: site oficial, LinkedIn, Receita Federal, notícias
3. Retorne dados de forma organizada
4. SEMPRE cite as fontes com URLs

IMPORTANTE: Busque apenas dados básicos, não análise de mercado ou concorrentes.`

// BuildStep1SearchPrompt creates the user prompt for Step 1 Perplexity search
func BuildStep1SearchPrompt(company *CompanyInput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pesquise DADOS BÁSICOS sobre a empresa brasileira: %s", company.Name))

	if company.CNPJ != nil && *company.CNPJ != "" {
		sb.WriteString(fmt.Sprintf("\nCNPJ: %s", *company.CNPJ))
	}
	if company.Website != nil && *company.Website != "" {
		sb.WriteString(fmt.Sprintf("\nWebsite: %s", *company.Website))
	}

	sb.WriteString(`

ENCONTRE APENAS:
1. **Identificação**: CNPJ completo, razão social, nome fantasia
2. **Fundação**: Ano de fundação, breve história
3. **Sede**: Localização da sede principal (cidade, estado)
4. **Tamanho**: Número de funcionários (faixa estimada)
5. **Website**: URL oficial
6. **Redes Sociais**: LinkedIn, Twitter/X, Instagram, Facebook
7. **Executivos**: CEO, fundadores, principais líderes (nome e cargo)

NÃO busque: modelo de negócio, produtos, concorrentes, análise de mercado.
Seja específico e factual. Cite as fontes.`)

	return sb.String()
}

// BuildStep1SearchPromptWithCNPJ creates the user prompt for Step 1 with CNPJ registry content
// The CNPJ content is included as additional context for Perplexity
func BuildStep1SearchPromptWithCNPJ(company *CompanyInput, cnpjContent string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pesquise DADOS BÁSICOS sobre a empresa brasileira: %s", company.Name))

	if company.CNPJ != nil && *company.CNPJ != "" {
		sb.WriteString(fmt.Sprintf("\nCNPJ: %s", *company.CNPJ))
	}
	if company.Website != nil && *company.Website != "" {
		sb.WriteString(fmt.Sprintf("\nWebsite: %s", *company.Website))
	}

	sb.WriteString(`

## DADOS OFICIAIS DO CNPJ (casadosdados.com.br)
Os dados abaixo são oficiais da Receita Federal. Use como referência principal para:
- Razão Social, Nome Fantasia
- Data de Abertura (ano de fundação)
- Endereço da Sede
- CNAE (atividades econômicas)
- Capital Social
- Sócios/Administradores
- Telefone, Email

`)
	sb.WriteString(cnpjContent)

	sb.WriteString(`

## BUSCAR ADICIONALMENTE:
1. **Website**: URL oficial (se não constar acima)
2. **Redes Sociais**: LinkedIn, Twitter/X, Instagram, Facebook
3. **Executivos**: CEO, fundadores, principais líderes (além dos sócios acima)
4. **Informações complementares**: Tamanho atual, notícias recentes

NÃO busque: modelo de negócio, produtos, concorrentes, análise de mercado.
Seja específico e factual. Cite as fontes.`)

	return sb.String()
}

// Step1FormatSystemPrompt is the system prompt for Gemini Step 1 JSON formatting
const Step1FormatSystemPrompt = `Você é um extrator de dados JSON.
Extraia APENAS os dados presentes na pesquisa.
NUNCA invente dados. Use null para campos não encontrados.

PRIORIDADE MÁXIMA: Se houver "Dados Oficiais do CNPJ (casadosdados.com.br)", use esses dados como fonte principal para:
- legal_name (Razão Social)
- trade_name (Nome Fantasia)
- foundation_year (extrair ano da Data de Abertura)
- headquarters (Município + Estado)
- phone
- email (SEMPRE em minúsculas, ex: contato@empresa.com.br)
- cnae_primary (CNAE Principal com código e descrição)
- cnae_codes (lista de todos os CNAEs)
- capital_social
- partners (lista de sócios com qualificação e data)

Para sócios (partners), extraia APENAS as linhas com nomes de pessoas no formato:
"Nome Completo - Qualificação - Data"
- Converta nomes para Proper Case (ex: "JOAO SILVA" → "João Silva")
- NÃO inclua cabeçalhos como "Sócios:" ou "Nome do sócio - Qualificação - Data de entrada".

Retorne APENAS JSON válido, sem texto antes ou depois.`

// Step1JSONTemplate is the expected JSON structure for Step 1
const Step1JSONTemplate = `{
  "cnpj": "XX.XXX.XXX/XXXX-XX ou null",
  "website": "https://... ou null",
  "legal_name": "Razão Social ou null",
  "trade_name": "Nome Fantasia ou null",
  "foundation_year": "YYYY ou null",
  "headquarters": "Cidade, Estado ou null",
  "employees_range": "50-100 ou Micro Empresa ou null",
  "phone": "telefone ou null",
  "email": "email@empresa.com ou null",
  "cnae_primary": "7990200 - Descrição da atividade ou null",
  "cnae_codes": ["7990200 - Descrição", "6202300 - Outra atividade"],
  "capital_social": "R$ 1.000,00 ou null",
  "partners": ["João da Silva - Sócio-Administrador - 01/01/2023", "Maria Santos - Sócio - 01/01/2023"],
  "linkedin_url": "https://linkedin.com/company/... ou null",
  "twitter_handle": "@empresa ou null",
  "instagram_url": "https://instagram.com/... ou null",
  "facebook_url": "https://facebook.com/... ou null",
  "key_executives": ["Nome - Cargo", "Nome - Cargo"],
  "confidence_score": 75,
  "sources": ["fonte1.com", "casadosdados.com.br"]
}`

// =============================================================================
// STEP 2: BUSINESS MODEL PROMPTS
// Fields: modelo de negócio, produtos/serviços, público alvo, proposta de valor, região geográfica
// =============================================================================

// Step2SearchSystemPrompt is the system prompt for Perplexity Step 2 search
const Step2SearchSystemPrompt = `Você é um analista de modelos de negócio.
Foco: Entender como a empresa gera valor e para quem.

INSTRUÇÕES:
1. Use o contexto fornecido sobre a empresa
2. Busque informações sobre produtos, serviços, clientes
3. Identifique proposta de valor e diferenciação
4. Mapeie regiões de atuação geográfica

IMPORTANTE: Não busque concorrentes ou análise SWOT neste momento.`

// BuildStep2SearchPrompt creates the user prompt for Step 2 Perplexity search
// websiteContent is optional - if provided, it will be included as primary source
func BuildStep2SearchPrompt(company *CompanyInput, step1Data *Step1BasicInfo, websiteContent string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Analise o MODELO DE NEGÓCIO de: %s", company.Name))

	// Include Step 1 context for richer search
	if step1Data != nil {
		if step1Data.LegalName != nil && *step1Data.LegalName != "" {
			sb.WriteString(fmt.Sprintf("\nRazão Social: %s", *step1Data.LegalName))
		}
		if step1Data.Headquarters != nil && *step1Data.Headquarters != "" {
			sb.WriteString(fmt.Sprintf("\nSede: %s", *step1Data.Headquarters))
		}
		if step1Data.EmployeesRange != nil && *step1Data.EmployeesRange != "" {
			sb.WriteString(fmt.Sprintf("\nFuncionários: %s", *step1Data.EmployeesRange))
		}
		if step1Data.Website != nil && *step1Data.Website != "" {
			sb.WriteString(fmt.Sprintf("\nWebsite: %s", *step1Data.Website))
		}
		if step1Data.LinkedInURL != nil && *step1Data.LinkedInURL != "" {
			sb.WriteString(fmt.Sprintf("\nLinkedIn: %s", *step1Data.LinkedInURL))
		}
	}

	if company.Industry != nil && *company.Industry != "" {
		sb.WriteString(fmt.Sprintf("\nSetor informado: %s", *company.Industry))
	}

	// Include website content if available (crawled via Jina Reader)
	if websiteContent != "" {
		sb.WriteString(`

=== CONTEÚDO DO SITE DA EMPRESA ===
`)
		sb.WriteString(websiteContent)
		sb.WriteString(`
=== FIM DO CONTEÚDO DO SITE ===

IMPORTANTE: Use o conteúdo do site acima como FONTE PRIMÁRIA de informações.
Busque informações ADICIONAIS na web para complementar (LinkedIn, notícias, app stores, etc).
`)
	}

	sb.WriteString(`

ENCONTRE:
1. **Modelo de Negócio**: SaaS, Marketplace, Serviços, Indústria, Varejo, etc.
2. **Setor/Subsetor**: Classificação do setor de atuação
3. **Produtos/Serviços**: Principais ofertas da empresa
4. **Modelo de Preço**: Assinatura, uso, freemium, licença, etc.
5. **Público-Alvo**: B2B, B2C, B2B2C, segmentos específicos
6. **Proposta de Valor**: O que diferencia a empresa
7. **Diferenciais**: USPs (Unique Selling Points)
8. **Região Geográfica**: Onde a empresa opera (Brasil, estados, LATAM, global)

NÃO busque: concorrentes, análise SWOT, notícias, reputação.
Seja específico. Cite as fontes.`)

	return sb.String()
}

// Step2FormatSystemPrompt is the system prompt for Gemini Step 2 JSON formatting
const Step2FormatSystemPrompt = `Você é um extrator de dados JSON para modelos de negócio.
Extraia APENAS dados da pesquisa fornecida.
Retorne APENAS JSON válido, sem texto antes ou depois.`

// Step2JSONTemplate is the expected JSON structure for Step 2
const Step2JSONTemplate = `{
  "business_model": "SaaS/Marketplace/Serviços/etc ou null",
  "industry": "Setor ou null",
  "sector": "Subsetor ou null",
  "main_products": ["Produto 1", "Produto 2"],
  "pricing_model": "Assinatura/Uso/Freemium ou null",
  "target_market": "B2B/B2C/B2B2C ou null",
  "target_audience": "Descrição do público ou null",
  "customer_segments": ["Segmento 1", "Segmento 2"],
  "value_proposition": "Proposta de valor ou null",
  "unique_selling_points": ["USP 1", "USP 2"],
  "geographic_regions": ["Brasil", "LATAM"],
  "service_areas": ["SP", "RJ", "MG"],
  "confidence_score": 75,
  "sources": ["fonte1.com"]
}`

// =============================================================================
// STEP 3: COMPETITIVE INTELLIGENCE PROMPTS
// Fields: concorrentes, informações do setor, reputação, notícias recentes
// =============================================================================

// Step3SearchSystemPrompt is the system prompt for Perplexity Step 3 search
const Step3SearchSystemPrompt = `Você é um analista de inteligência competitiva.
Foco: Concorrentes, informações do setor, reputação e notícias.

INSTRUÇÕES:
1. Use todo o contexto fornecido sobre a empresa
2. Busque concorrentes diretos e indiretos
3. Identifique informações do setor (crescimento, tendências)
4. Busque reputação (Glassdoor, Reclame Aqui)
5. Busque notícias recentes (últimos 12 meses)

IMPORTANTE: NÃO faça análise SWOT ou estimativa de TAM/SAM/SOM.`

// BuildStep3SearchPrompt creates the user prompt for Step 3 Perplexity search
func BuildStep3SearchPrompt(company *CompanyInput, step1Data *Step1BasicInfo, step2Data *Step2BusinessModel) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Análise de INTELIGÊNCIA COMPETITIVA para: %s", company.Name))

	// Include Step 1 context
	if step1Data != nil {
		if step1Data.Headquarters != nil && *step1Data.Headquarters != "" {
			sb.WriteString(fmt.Sprintf("\nSede: %s", *step1Data.Headquarters))
		}
		if step1Data.EmployeesRange != nil && *step1Data.EmployeesRange != "" {
			sb.WriteString(fmt.Sprintf("\nFuncionários: %s", *step1Data.EmployeesRange))
		}
	}

	// Include Step 2 context (rich business model data)
	if step2Data != nil {
		if step2Data.BusinessModel != nil && *step2Data.BusinessModel != "" {
			sb.WriteString(fmt.Sprintf("\nModelo: %s", *step2Data.BusinessModel))
		}
		if step2Data.Industry != nil && *step2Data.Industry != "" {
			sb.WriteString(fmt.Sprintf("\nSetor: %s", *step2Data.Industry))
		}
		if len(step2Data.MainProducts) > 0 {
			sb.WriteString(fmt.Sprintf("\nProdutos: %s", strings.Join(step2Data.MainProducts, ", ")))
		}
		if step2Data.TargetAudience != nil && *step2Data.TargetAudience != "" {
			sb.WriteString(fmt.Sprintf("\nPúblico: %s", *step2Data.TargetAudience))
		}
		if len(step2Data.GeographicRegions) > 0 {
			sb.WriteString(fmt.Sprintf("\nRegiões: %s", strings.Join(step2Data.GeographicRegions, ", ")))
		}
	}

	sb.WriteString(`

ENCONTRE:
1. **Concorrentes**: Liste TODOS os concorrentes diretos e indiretos
2. **Detalhes dos Concorrentes**: Breve descrição de cada um
3. **Crescimento do Setor**: Taxa de crescimento anual (CAGR)
4. **Tendências do Setor**: Principais tendências de mercado
5. **Concentração de Mercado**: Fragmentado ou concentrado
6. **Contexto Regulatório**: Marco regulatório relevante
7. **Posição no Mercado**: Líder, desafiador, nicho
8. **Reputação Glassdoor**: Nota e percepção dos funcionários
9. **Reputação Reclame Aqui**: Nota e percepção dos clientes
10. **Notícias Recentes**: Eventos dos últimos 12 meses (lançamentos, funding, expansões, crises)

NÃO faça: análise SWOT, estimativa de TAM/SAM/SOM, recomendações estratégicas.
Seja específico e factual. Cite as fontes.`)

	return sb.String()
}

// Step3FormatSystemPrompt is the system prompt for Gemini Step 3 JSON formatting
const Step3FormatSystemPrompt = `Você é um extrator de dados JSON para análise competitiva.
Extraia APENAS dados da pesquisa fornecida.
Preencha arrays com pelo menos 3 itens quando dados estiverem disponíveis.
Retorne APENAS JSON válido, sem texto antes ou depois.`

// Step3JSONTemplate is the expected JSON structure for Step 3
const Step3JSONTemplate = `{
  "competitors": ["Concorrente 1", "Concorrente 2", "Concorrente 3"],
  "competitor_details": ["Conc 1: descrição", "Conc 2: descrição"],
  "industry_growth_rate": "+X% CAGR ou null",
  "industry_trends": ["Tendência 1", "Tendência 2"],
  "market_concentration": "Fragmentado/Concentrado ou null",
  "regulatory_context": "Marco regulatório ou null",
  "market_position": "Líder/Desafiador/Nicho ou null",
  "glassdoor_rating": "4.2/5 ou null",
  "reclame_aqui_rating": "8.5/10 ou null",
  "reputation_summary": "Resumo da reputação ou null",
  "recent_news": ["Dez/2024: Notícia 1", "Nov/2024: Notícia 2"],
  "confidence_score": 75,
  "sources": ["fonte1.com"]
}`

// =============================================================================
// SHARED HELPERS
// =============================================================================

// BuildFormatPrompt creates the JSON formatting prompt for Gemini
func BuildFormatPrompt(stepName, rawData, jsonTemplate string) string {
	return fmt.Sprintf(`# TAREFA: Extrair %s para JSON

## DADOS DA PESQUISA
%s

## INSTRUÇÕES
Extraia os dados acima para o formato JSON abaixo.
Use null para campos não encontrados.
Retorne APENAS o JSON, sem texto adicional.

## FORMATO ESPERADO
%s`, stepName, rawData, jsonTemplate)
}
