package enrichment

import "fmt"

// =============================================================================
// STAGE 1: PERPLEXITY SEARCH PROMPTS
// =============================================================================

// SearchSystemPrompt is the system prompt for Perplexity web search
const SearchSystemPrompt = `Você é um pesquisador de inteligência de mercado. Sua tarefa é encontrar informações factuais sobre empresas brasileiras.

INSTRUÇÕES:
1. Busque informações públicas e verificáveis
2. Inclua dados de: site oficial, LinkedIn, notícias recentes, Glassdoor, Reclame Aqui
3. Liste TODOS os concorrentes que encontrar
4. Inclua notícias dos últimos 12 meses
5. Retorne os dados de forma organizada (pode ser texto estruturado)
6. SEMPRE cite as fontes com URLs`

// BuildSearchPrompt creates the user prompt for Perplexity web search
func BuildSearchPrompt(company *CompanyInput) string {
	prompt := fmt.Sprintf(`Pesquise informações sobre a empresa brasileira: %s`, company.Name)

	if company.Website != nil && *company.Website != "" {
		prompt += fmt.Sprintf("\nWebsite: %s", *company.Website)
	}
	if company.Industry != nil && *company.Industry != "" {
		prompt += fmt.Sprintf("\nSetor: %s", *company.Industry)
	}
	if company.Location != nil && *company.Location != "" {
		prompt += fmt.Sprintf("\nLocalização: %s", *company.Location)
	}

	prompt += `

ENCONTRE E LISTE:
1. **Dados Básicos**: CNPJ, razão social, fundação, sede, funcionários
2. **Negócio**: Produtos/serviços, modelo de negócio, público-alvo, proposta de valor
3. **Concorrentes**: Liste TODOS os concorrentes diretos e indiretos com breve descrição
4. **Mercado**: Tamanho do mercado, crescimento, tendências, regulamentações
5. **Notícias Recentes**: Eventos dos últimos 12 meses (lançamentos, funding, expansões)
6. **Reputação**: Notas no Glassdoor, Reclame Aqui, avaliações
7. **Executivos**: CEO, fundadores, liderança
8. **Forças e Fraquezas**: O que a empresa faz bem? Onde falha?
9. **Redes Sociais**: LinkedIn, Twitter/X

Seja específico e factual. Cite as fontes.`

	return prompt
}

// =============================================================================
// STAGE 2: CLAUDE SYNTHESIS PROMPTS
// =============================================================================

// SynthesisSystemPrompt is the system prompt for Claude synthesis
const SynthesisSystemPrompt = `Você é um extrator de dados. Sua ÚNICA tarefa é extrair dados da pesquisa e retornar JSON.

REGRA CRÍTICA DE INTEGRIDADE:
- Use APENAS dados do texto de pesquisa fornecido
- NUNCA use seu conhecimento prévio para "corrigir" ou substituir valores
- Se a pesquisa diz "SELIC = 15%", retorne 15%, mesmo que pareça diferente do que você "sabe"

REGRAS DE FORMATO:
1. Retorne APENAS JSON válido, SEM texto antes ou depois
2. JSON deve ser PLANO (sem objetos aninhados)
3. Arrays devem conter APENAS strings, NUNCA objetos
4. SEMPRE preencha: competitors, strengths, weaknesses (mínimo 3 itens cada)
5. Se dado não existir na pesquisa, use "N/A" ou infira do setor

FORMATO DOS ARRAYS (OBRIGATÓRIO):
- ✅ CORRETO: "competitors": ["Empresa A", "Empresa B"]
- ❌ ERRADO: "competitors": [{"nome": "Empresa A"}]
- ✅ CORRETO: "recent_news": ["Jan/2024: Notícia X", "Fev/2024: Notícia Y"]
- ❌ ERRADO: "recent_news": [{"data": "Jan/2024", "titulo": "Notícia X"}]`

// synthesisJSONTemplate is the expected JSON structure for synthesis output
const synthesisJSONTemplate = `{
  "cnpj": "XX.XXX.XXX/XXXX-XX",
  "website": "https://...",
  "industry": "Setor",
  "company_size": "Micro/Pequena/Média/Grande",
  "location": "Cidade, Estado",
  "target_market": "B2B/B2C/B2B2C",
  "funding_stage": "Bootstrap/Seed/Series A/B/C",
  "foundation_year": "YYYY",
  "legal_name": "Razão Social",
  "headquarters": "Cidade, Estado",
  "sector": "Subsetor",
  "target_audience": "Público-alvo",
  "value_proposition": "Proposta de valor",
  "employees_range": "50-100",
  "revenue_estimate": "R$ XM - YM/ano",
  "business_model": "SaaS/Marketplace/etc",
  "market_share_status": "Líder/Desafiador/Nicho",
  "digital_maturity": 7,

  "main_products": ["Produto 1", "Produto 2"],
  "service_areas": ["Brasil", "LATAM"],
  "key_partnerships": ["Parceiro 1"],
  "recent_news": ["Dez/2024: Notícia 1", "Nov/2024: Notícia 2"],
  "key_executives": ["Nome - Cargo"],
  "company_history": "Breve história",
  "customer_segments": ["Segmento 1", "Segmento 2"],
  "pricing_model": "Assinatura/Uso/Freemium",
  "market_position": "Posicionamento",
  "unique_selling_points": ["USP 1", "USP 2"],

  "competitors": ["Concorrente 1", "Concorrente 2", "Concorrente 3"],
  "competitor_details": ["Conc 1: descrição", "Conc 2: descrição"],
  "competitive_advantage": "Vantagem principal",
  "market_share": "X%% estimado",

  "strengths": ["Força 1", "Força 2", "Força 3"],
  "weaknesses": ["Fraqueza 1", "Fraqueza 2"],
  "opportunities": ["Oportunidade 1", "Oportunidade 2"],
  "threats": ["Ameaça 1", "Ameaça 2"],
  "strategic_challenges": ["Desafio 1"],

  "industry_growth_rate": "+X%% CAGR",
  "industry_trends": ["Tendência 1", "Tendência 2"],
  "regulatory_context": "Marco regulatório",
  "market_concentration": "Fragmentado/Concentrado",

  "tam_estimate": "R$ XXB",
  "sam_estimate": "R$ XXB",
  "som_estimate": "R$ XXM",

  "linkedin_url": "https://linkedin.com/company/...",
  "twitter_handle": "@empresa",

  "confidence_score": 75,
  "sources": ["fonte1.com", "fonte2.com"]
}`

// BuildSynthesisPrompt creates the user prompt for Claude synthesis
func BuildSynthesisPrompt(company *CompanyInput, rawData string) string {
	prompt := fmt.Sprintf(`# TAREFA: Sintetizar Dados em Análise Estratégica

## EMPRESA
Nome: %s`, company.Name)

	if company.Industry != nil && *company.Industry != "" {
		prompt += fmt.Sprintf("\nSetor: %s", *company.Industry)
	}

	prompt += fmt.Sprintf(`

## DADOS BRUTOS DA PESQUISA (Perplexity)
%s

## INSTRUÇÕES

Analise os dados acima e retorne um JSON estruturado com análise estratégica.

IMPORTANTE:
- Extraia TODOS os concorrentes mencionados
- Identifique forças e fraquezas baseado nos dados
- Infira oportunidades e ameaças do contexto de mercado
- Se algum dado não existir, use seu conhecimento do setor para inferir

## FORMATO DE RESPOSTA (JSON ESTRITO)

Retorne APENAS o JSON abaixo, preenchido com dados reais:

%s`, rawData, synthesisJSONTemplate)

	return prompt
}
