package challenge

// ChallengeTypeInfo represents a specific challenge type with display metadata
// Used by frontend to show available challenge options with labels and examples
type ChallengeTypeInfo struct {
	Code        string            `json:"code"`
	Category    ChallengeCategory `json:"category"`
	Label       string            `json:"label"`
	Description string            `json:"description"`
	Example     string            `json:"example"`
}

// ChallengeCategoryInfo represents category metadata for frontend display
type ChallengeCategoryInfo struct {
	Code        ChallengeCategory `json:"code"`
	Label       string            `json:"label"`
	Description string            `json:"description"`
}

// ChallengeCategories returns all available categories with display metadata
var ChallengeCategories = []ChallengeCategoryInfo{
	{Code: CategoryGrowth, Label: "Crescimento", Description: "Estratégias de expansão e crescimento"},
	{Code: CategoryTransform, Label: "Transformação", Description: "Mudanças estruturais e modernização"},
	{Code: CategoryCompete, Label: "Competitividade", Description: "Posicionamento e diferenciação"},
}

// ChallengeTypesByCategory returns all challenge types organized by category with display metadata
var ChallengeTypesByCategory = map[ChallengeCategory][]ChallengeTypeInfo{
	CategoryGrowth: {
		{Code: "growth_organic", Category: CategoryGrowth, Label: "Crescimento Orgânico", Description: "Crescer com recursos próprios", Example: "Aumentar faturamento 20% ao ano"},
		{Code: "growth_geographic", Category: CategoryGrowth, Label: "Expansão Geográfica", Description: "Expandir para novas regiões", Example: "Entrar no Nordeste"},
		{Code: "growth_segment", Category: CategoryGrowth, Label: "Novo Segmento", Description: "Entrar em novo segmento de mercado", Example: "Atender indústria além de agro"},
		{Code: "growth_product", Category: CategoryGrowth, Label: "Novos Produtos", Description: "Lançar novos produtos/serviços", Example: "Criar linha de serviços"},
		{Code: "growth_channel", Category: CategoryGrowth, Label: "Novos Canais", Description: "Novos canais de venda", Example: "Vender direto ao consumidor"},
	},
	CategoryTransform: {
		{Code: "transform_digital", Category: CategoryTransform, Label: "Transformação Digital", Description: "Digitalização de processos", Example: "Implementar ERP/CRM"},
		{Code: "transform_model", Category: CategoryTransform, Label: "Modelo de Negócio", Description: "Mudar modelo de negócio", Example: "De venda para assinatura"},
	},
	CategoryCompete: {
		{Code: "compete_differentiate", Category: CategoryCompete, Label: "Diferenciação", Description: "Criar diferenciação", Example: "Sair da guerra de preço"},
		{Code: "compete_defend", Category: CategoryCompete, Label: "Defender Posição", Description: "Defender posição de mercado", Example: "Novos entrantes ameaçando"},
		{Code: "compete_reposition", Category: CategoryCompete, Label: "Reposicionamento", Description: "Reposicionar marca", Example: "Subir de segmento"},
	},
}

// GetChallengeTypeInfo returns the challenge type info for a given code
func GetChallengeTypeInfo(code string) *ChallengeTypeInfo {
	for _, types := range ChallengeTypesByCategory {
		for _, ct := range types {
			if ct.Code == code {
				return &ct
			}
		}
	}
	return nil
}

// GetAllChallengeTypes returns a flat list of all challenge types with display metadata
func GetAllChallengeTypes() []ChallengeTypeInfo {
	var all []ChallengeTypeInfo
	for _, types := range ChallengeTypesByCategory {
		all = append(all, types...)
	}
	return all
}
