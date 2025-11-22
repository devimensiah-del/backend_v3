package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"time"

	"backend_v3/infrastructure"
)

// Data structures matching the Coimma analysis
type PESTELData struct {
	Political     []string `json:"political"`
	Economic      []string `json:"economic"`
	Social        []string `json:"social"`
	Technological []string `json:"technological"`
	Environmental []string `json:"environmental"`
	Legal         []string `json:"legal"`
	Summary       string   `json:"summary"`
}

type PorterData struct {
	CompetitiveRivalry    string `json:"competitive_rivalry"`
	SupplierPower         string `json:"supplier_power"`
	BuyerPower            string `json:"buyer_power"`
	ThreatNewEntrants     string `json:"threat_new_entrants"`
	ThreatSubstitutes     string `json:"threat_substitutes"`
	OverallAttractiveness string `json:"overall_attractiveness"`
	Summary               string `json:"summary"`
}

type TAMData struct {
	TAM         string   `json:"tam"`
	SAM         string   `json:"sam"`
	SOM         string   `json:"som"`
	Assumptions []string `json:"assumptions"`
	CAGR        string   `json:"cagr"`
	Summary     string   `json:"summary"`
}

type SWOTData struct {
	Strengths     []string `json:"strengths"`
	Weaknesses    []string `json:"weaknesses"`
	Opportunities []string `json:"opportunities"`
	Threats       []string `json:"threats"`
	Summary       string   `json:"summary"`
}

type BlueOceanData struct {
	Eliminate     []string `json:"eliminate"`
	Reduce        []string `json:"reduce"`
	Raise         []string `json:"raise"`
	Create        []string `json:"create"`
	NewValueCurve string   `json:"new_value_curve"`
	Summary       string   `json:"summary"`
}

type ScenariosData struct {
	Optimistic          string   `json:"scenario_optimistic"`
	Realist             string   `json:"scenario_realist"`
	Pessimistic         string   `json:"scenario_pessimistic"`
	EarlyWarningSignals []string `json:"early_warning_signals"`
	Summary             string   `json:"summary"`
}

type OKRObjective struct {
	Title      string   `json:"title"`
	KeyResults []string `json:"key_results"`
}

type OKRsData struct {
	Objectives []OKRObjective `json:"objectives"`
	Summary    string         `json:"summary"`
}

type BSCData struct {
	Financial         []string `json:"financial"`
	Customer          []string `json:"customer"`
	InternalProcesses []string `json:"internal_processes"`
	LearningGrowth    []string `json:"learning_growth"`
	Summary           string   `json:"summary"`
}

type SynthesisData struct {
	ExecutiveSummary     string   `json:"executive_summary"`
	KeyFindings          []string `json:"key_findings"`
	StrategicPriorities  []string `json:"strategic_priorities"`
	Roadmap              []string `json:"roadmap"`
	OverallRecommendation string   `json:"overall_recommendation"`
}

type ReportData struct {
	CompanyName string
	Industry    string
	Market      string
	Date        string
	Version     string
	PESTEL      PESTELData
	Porter      PorterData
	TAM         TAMData
	SWOT        SWOTData
	BlueOcean   BlueOceanData
	Scenarios   ScenariosData
	OKRs        OKRsData
	BSC         BSCData
	Synthesis   SynthesisData
}

func main() {
	// Report data
	data := ReportData{
		CompanyName: "Coimma",
		Industry:    "Equipamentos Agropecuários / Sistemas de Pesagem",
		Market:      "Brasil",
		Date:        time.Now().Format("02/01/2006"),
		Version:     "1.0",
		PESTEL: PESTELData{
			Political: []string{
				"Regulação governamental para importação de insumos",
				"Incentivos fiscais promovem investimentos no agronegócio",
				"Políticas de inovação tecnológica impactam o setor",
			},
			Economic: []string{
				"Oscilação cambial eleva custos de insumos importados",
				"Crescimento do agronegócio impulsiona demanda",
				"Inflação e juros afetam financiamentos no setor",
			},
			Social: []string{
				"Alta demanda por soluções que aumentem produtividade pecuária",
				"Resistência cultural à automação tradicional rural",
				"Crescente atenção ao bem-estar animal e técnicas modernas",
			},
			Technological: []string{
				"Avanços em IoT e automação revolucionam sistemas de pesagem",
				"Integração digital aprimora calibração e manutenção",
				"Inovação em sensores e tecnologia compete com grandes marcas",
			},
			Environmental: []string{
				"Pressão por práticas sustentáveis na agropecuária",
				"Impactos climáticos afetam produção e logística",
				"Demanda por redução de resíduos e emissões industriais",
			},
			Legal: []string{
				"Normas técnicas e certificações regulam calibração de equipamentos",
				"Legislação ambiental rigorosa influencia processos industriais",
				"Compliance no setor agropecuário impõe desafios regulatórios",
			},
			Summary: "A análise PESTEL destaca oportunidades e ameaças para a Coimma, com foco em regulação, avanços tecnológicos e tendências sustentáveis que influenciam o agronegócio e a inovação no setor.",
		},
		Porter: PorterData{
			CompetitiveRivalry:    "Alta rivalidade com competidores estabelecidos como Toledo e Balanças Jundiaí. Mercado maduro exige diferenciação via especialização em soluções pecuárias e sistemas integrados de pesagem, além de serviços de manutenção e calibração.",
			SupplierPower:         "Poder moderado dos fornecedores devido à dependência de componentes eletrônicos e aço. Necessidade de manter relacionamentos estratégicos com múltiplos fornecedores para garantir estabilidade de preços e suprimentos.",
			BuyerPower:            "Alto poder dos compradores devido à natureza B2B do negócio, sensibilidade a preços e disponibilidade de alternativas. Clientes pecuaristas e cerealistas demandam qualidade e suporte técnico consistente.",
			ThreatNewEntrants:     "Barreiras de entrada moderadas devido aos requisitos técnicos, certificações e rede de distribuição/suporte necessária. Investimento inicial significativo em P&D e infraestrutura reduz ameaças de novos entrantes.",
			ThreatSubstitutes:     "Baixa ameaça de substitutos diretos para sistemas de pesagem, porém crescente pressão de soluções digitais integradas e IoT que podem redefinir o mercado de monitoramento e controle pecuário.",
			OverallAttractiveness: "Média",
			Summary:               "Setor com atratividade média, caracterizado por alta rivalidade e poder de compradores, compensados por barreiras de entrada significativas e baixa ameaça de substitutos. Sucesso depende de diferenciação técnica e serviços.",
		},
		TAM: TAMData{
			TAM:         "R$ 50B",
			SAM:         "R$ 10B",
			SOM:         "R$ 100M",
			Assumptions: []string{
				"Crescimento da pecuária e grãos impulsiona demanda",
				"Digitalização aprimora eficiência dos sistemas de pesagem",
				"Manutenção e calibração definem fidelização do cliente",
			},
			CAGR:    "+15% a.a.",
			Summary: "A Coimma possui potencial para expandir no mercado agro, com inovação e foco em qualidade nos sistemas de pesagem.",
		},
		SWOT: SWOTData{
			Strengths: []string{
				"Expertise consolidada em sistemas de pesagem e manejo de gado com soluções integradas",
				"Portfolio diversificado atendendo pecuária e grãos com produtos complementares",
				"Capacidade de oferecer serviços de calibração e manutenção especializada",
				"Know-how técnico em automação e eletrônica aplicada ao agronegócio",
			},
			Weaknesses: []string{
				"Presença digital limitada e baixa maturidade em marketing digital",
				"Dependência do mercado pecuário como principal fonte de receita",
				"Ausência de diferenciação clara na proposta de valor frente a concorrentes",
				"Limitada visibilidade da marca em segmentos adjacentes",
			},
			Opportunities: []string{
				"Crescente demanda por soluções de automação e IoT no agronegócio",
				"Expansão para novos segmentos aproveitando a base tecnológica existente",
				"Desenvolvimento de serviços digitais e monitoramento remoto",
				"Parcerias estratégicas para ampliar cobertura de mercado",
			},
			Threats: []string{
				"Entrada de concorrentes com soluções tecnologicamente mais avançadas",
				"Pressão de preços devido ao alto poder de negociação dos compradores",
				"Mudanças regulatórias em padrões de pesagem e rastreabilidade",
				"Ciclos econômicos do agronegócio afetando investimentos dos clientes",
			},
			Summary: "Empresa com forte base técnica e portfolio estabelecido, mas necessitando modernização digital e diversificação. Oportunidades em inovação tecnológica podem compensar ameaças competitivas e regulatórias.",
		},
		BlueOcean: BlueOceanData{
			Eliminate: []string{
				"Vendas sem diagnóstico técnico prévio",
				"Dependência de distribuição tradicional",
				"Produtos padronizados sem personalização",
			},
			Reduce: []string{
				"Foco em produtos commodity de baixa margem",
				"Custos com estoque físico excessivo",
				"Intermediários na cadeia de vendas",
			},
			Raise: []string{
				"Integração IoT em sistemas de pesagem",
				"Consultoria especializada em otimização de manejo",
				"Análise preditiva de dados do rebanho",
			},
			Create: []string{
				"Plataforma digital de gestão pecuária integrada",
				"Marketplace de soluções para pecuaristas",
				"Sistema de rastreabilidade blockchain",
			},
			NewValueCurve: "Transformação de fabricante de equipamentos em provedor de soluções digitais integradas para gestão pecuária e rastreabilidade",
			Summary:       "Evolução do modelo tradicional de fabricante para ecossistema digital de gestão pecuária, combinando hardware IoT, software e serviços",
		},
		Scenarios: ScenariosData{
			Optimistic:  "Crescente demanda por automação no agronegócio impulsiona adoção de sistemas digitais de pesagem. Regulações favoráveis e incentivos à modernização rural aceleram investimentos. Coimma expande market share com soluções IoT integradas e certificações sustentáveis, consolidando liderança em sistemas de precisão para pecuária e grãos.",
			Realist:     "Mercado mantém crescimento moderado, com adoção gradual de tecnologias digitais no campo. Coimma preserva posição competitiva através de atualizações incrementais em produtos tradicionais, enquanto desenvolve novas soluções digitais. Pressões por sustentabilidade demandam adaptações graduais nos equipamentos.",
			Pessimistic: "Instabilidade econômica reduz investimentos em modernização rural. Concorrentes internacionais dominam segmento de alta tecnologia com soluções mais avançadas. Regulações ambientais rigorosas e custos de adequação tecnológica pressionam margens, enquanto demanda por sistemas tradicionais diminui.",
			EarlyWarningSignals: []string{
				"Queda nas vendas de equipamentos tradicionais",
				"Entrada de players internacionais com soluções IoT",
				"Novas regulamentações ambientais para o setor pecuário",
			},
			Summary: "Principais riscos incluem defasagem tecnológica frente à digitalização do agro, pressão competitiva de empresas com soluções mais avançadas e necessidade de adequação a novas exigências ambientais. Capacidade de inovação e adaptação serão cruciais.",
		},
		OKRs: OKRsData{
			Objectives: []OKRObjective{
				{
					Title: "Modernização Digital",
					KeyResults: []string{
						"Integrar IoT em 35% dos produtos atuais",
						"Reduzir custos operacionais em 15% com automação",
						"Implementar 2 novas funcionalidades digitais",
					},
				},
				{
					Title: "Expansão de Mercado",
					KeyResults: []string{
						"Aumentar base digital em 25% até Q4",
						"Firmar 3 parcerias estratégicas no setor",
						"Lançar 2 serviços integrados ao ecossistema",
					},
				},
			},
			Summary: "Roteiro de implementação focado em modernização tecnológica e expansão estratégica",
		},
		BSC: BSCData{
			Financial: []string{
				"Aumentar receita recorrente de serviços digitais em 30% ao ano",
				"Expandir margem operacional através da monetização de dados em 20%",
			},
			Customer: []string{
				"Alcançar 80% de adoção das soluções digitais pelos clientes atuais",
				"Atingir NPS >70 na plataforma digital integrada",
			},
			InternalProcesses: []string{
				"Integrar 100% dos produtos ao ecossistema IoT até 2024",
				"Implementar analytics preditivo em 80% da base instalada",
			},
			LearningGrowth: []string{
				"Capacitar 90% da equipe em tecnologias digitais e análise de dados",
				"Estabelecer 3 parcerias estratégicas com startups/techs",
			},
			Summary: "BSC focado na transformação digital do negócio, equilibrando expansão de receitas recorrentes com desenvolvimento de capacidades IoT",
		},
		Synthesis: SynthesisData{
			ExecutiveSummary: "A Coimma enfrenta um momento decisivo de transformação, necessitando evoluir de fabricante tradicional para provedor de soluções digitais integradas no agronegócio. A jornada de modernização tecnológica, combinada com sua expertise técnica estabelecida, permitirá criar um ecossistema digital competitivo e sustentável.",
			KeyFindings: []string{
				"Mercado pecuário em digitalização acelerada demanda evolução do modelo de negócios tradicional para soluções tecnológicas integradas",
				"Base técnica sólida e reputação estabelecida fornecem fundamento para transformação digital e expansão do portfólio",
				"Pressões competitivas e regulatórias exigem modernização urgente e desenvolvimento de novos recursos tecnológicos",
			},
			StrategicPriorities: []string{
				"Desenvolver plataforma digital integrando hardware IoT, software e serviços para gestão pecuária",
				"Estabelecer programa de capacitação técnica e marketing digital focado em autoridade setorial",
				"Implementar sistema de inovação contínua alinhado às tendências do agronegócio digital",
			},
			Roadmap: []string{
				"Fase 1: Modernização digital da linha atual e desenvolvimento de MVP da plataforma integrada (12 meses)",
				"Fase 2: Expansão do ecossistema digital e estabelecimento de parcerias estratégicas (24 meses)",
				"Fase 3: Consolidação como líder em soluções digitais para pecuária e diversificação do portfólio (36+ meses)",
			},
			OverallRecommendation: "Priorizar transformação digital mantendo excelência técnica, focando em construção de ecossistema integrado de soluções para o agronegócio.",
		},
	}

	// Generate HTML pages
	pages := []string{
		generateCoverPage(data),
		generatePESTELPage(data),
		generatePorterPage(data),
		generateTAMPage(data),
		generateSWOTPage(data),
		generateBlueOceanPage(data),
		generateScenariosPage(data),
		generateOKRsPage(data),
		generateBSCPage(data),
		generateSynthesisPage(data),
	}

	// Generate PDF
	gotenbergClient := infrastructure.NewGotenbergClient("http://localhost:3000")
	pdfBytes, err := gotenbergClient.Convert(context.Background(), pages)
	if err != nil {
		fmt.Printf("Error generating PDF: %v\n", err)
		os.Exit(1)
	}

	// Save PDF
	outputPath := "integration_tests/coimma_strategic_analysis.pdf"
	if err := os.WriteFile(outputPath, pdfBytes, 0644); err != nil {
		fmt.Printf("Error saving PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ PDF generated successfully: %s\n", outputPath)
	fmt.Printf("📄 Pages: %d\n", len(pages))
	fmt.Printf("📊 Size: %.2f KB\n", float64(len(pdfBytes))/1024)
}

// Template generators (landscape A4: 842px x 595px)

func generateCoverPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 50px; display: flex; position: relative; }
.accent-bar { position: absolute; left: 0; top: 140px; width: 6px; height: 200px; background: linear-gradient(180deg, #B89E68 0%, #A18852 100%); }
.left-section { flex: 1; padding-right: 40px; }
.right-section { flex: 1; display: flex; flex-direction: column; justify-content: center; }
.logo { width: 80px; height: 80px; margin-bottom: 40px; }
.report-label { font-size: 9px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.2em; color: #B89E68; margin-bottom: 20px; }
.company-name { font-size: 52px; font-weight: 300; letter-spacing: -0.02em; color: #0A101D; margin-bottom: 16px; line-height: 1.1; }
.report-title { font-size: 18px; font-weight: 400; color: #52525B; margin-bottom: 40px; }
.meta-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
.meta-item { border-left: 2px solid #E5E0D6; padding-left: 14px; }
.meta-label { font-size: 8px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #71717A; margin-bottom: 4px; }
.meta-value { font-size: 13px; font-weight: 500; color: #0A101D; }
.footer { position: absolute; bottom: 40px; left: 50px; right: 50px; padding-top: 20px; border-top: 1px solid #E5E0D6; font-size: 8px; color: #71717A; text-align: center; }
.footer-brand { font-weight: 600; color: #B89E68; }
</style>
</head><body><div class="page">
<div class="accent-bar"></div>
<div class="left-section">
<svg class="logo" viewBox="0 0 120 120"><circle cx="60" cy="60" r="58" fill="none" stroke="#0A101D" stroke-width="2"/><text x="60" y="75" font-family="Arial" font-size="32" font-weight="700" fill="#0A101D" text-anchor="middle">IM</text></svg>
<div class="report-label">Relatório de Análise Estratégica</div>
<h1 class="company-name">{{.CompanyName}}</h1>
<div class="report-title">Inteligência de Negócios Completa</div>
</div>
<div class="right-section">
<div class="meta-grid">
<div class="meta-item"><div class="meta-label">Setor</div><div class="meta-value">{{.Industry}}</div></div>
<div class="meta-item"><div class="meta-label">Mercado</div><div class="meta-value">{{.Market}}</div></div>
<div class="meta-item"><div class="meta-label">Data do Relatório</div><div class="meta-value">{{.Date}}</div></div>
<div class="meta-item"><div class="meta-label">Versão</div><div class="meta-value">{{.Version}}</div></div>
</div>
</div>
<div class="footer"><span class="footer-brand">IMENSIAH</span> — Inteligência Artificial + Inteligência Humana</div>
</div></body></html>`

	t := template.Must(template.New("cover").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generatePESTELPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.company-name-header { font-size: 10px; color: #71717A; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.factors-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 14px; }
.factor-box { border: 1px solid #E5E0D6; padding: 14px; height: 145px; display: flex; flex-direction: column; }
.factor-title { font-size: 11px; font-weight: 700; color: #0A101D; margin-bottom: 10px; padding-bottom: 6px; border-bottom: 2px solid #B89E68; }
.factor-list { list-style: none; flex: 1; overflow: hidden; }
.factor-item { font-size: 9px; line-height: 1.4; color: #111827; margin-bottom: 5px; padding-left: 12px; position: relative; }
.factor-item::before { content: '•'; position: absolute; left: 0; color: #B89E68; font-weight: 700; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">01</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">Análise PESTEL</h1>
<div class="page-subtitle">Fatores Ambientais</div>
<div class="factors-grid">
<div class="factor-box"><div class="factor-title">Político</div><ul class="factor-list">{{range .PESTEL.Political}}<li class="factor-item">{{.}}</li>{{end}}</ul></div>
<div class="factor-box"><div class="factor-title">Econômico</div><ul class="factor-list">{{range .PESTEL.Economic}}<li class="factor-item">{{.}}</li>{{end}}</ul></div>
<div class="factor-box"><div class="factor-title">Social</div><ul class="factor-list">{{range .PESTEL.Social}}<li class="factor-item">{{.}}</li>{{end}}</ul></div>
<div class="factor-box"><div class="factor-title">Tecnológico</div><ul class="factor-list">{{range .PESTEL.Technological}}<li class="factor-item">{{.}}</li>{{end}}</ul></div>
<div class="factor-box"><div class="factor-title">Ambiental</div><ul class="factor-list">{{range .PESTEL.Environmental}}<li class="factor-item">{{.}}</li>{{end}}</ul></div>
<div class="factor-box"><div class="factor-title">Legal</div><ul class="factor-list">{{range .PESTEL.Legal}}<li class="factor-item">{{.}}</li>{{end}}</ul></div>
</div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("pestel").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generatePorterPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; position: relative; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.forces-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
.force-box { border: 1px solid #E5E0D6; padding: 16px; height: 110px; }
.force-title { font-size: 11px; font-weight: 700; color: #B89E68; margin-bottom: 8px; }
.force-text { font-size: 9.5px; line-height: 1.5; color: #111827; }
.summary-box { border: 2px solid #B89E68; padding: 20px; background: #FAFAF9; }
.summary-title { font-size: 11px; font-weight: 700; color: #0A101D; margin-bottom: 10px; }
.summary-text { font-size: 10px; line-height: 1.6; color: #111827; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">02</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">5 Forças de Porter</h1>
<div class="page-subtitle">Análise Competitiva</div>
<div class="forces-grid">
<div class="force-box"><div class="force-title">Rivalidade Competitiva</div><div class="force-text">{{.Porter.CompetitiveRivalry}}</div></div>
<div class="force-box"><div class="force-title">Poder dos Fornecedores</div><div class="force-text">{{.Porter.SupplierPower}}</div></div>
<div class="force-box"><div class="force-title">Poder dos Compradores</div><div class="force-text">{{.Porter.BuyerPower}}</div></div>
<div class="force-box"><div class="force-title">Ameaça de Novos Entrantes</div><div class="force-text">{{.Porter.ThreatNewEntrants}}</div></div>
<div class="force-box"><div class="force-title">Ameaça de Substitutos</div><div class="force-text">{{.Porter.ThreatSubstitutes}}</div></div>
<div class="force-box"><div class="force-title">Atratividade Geral</div><div class="force-text"><strong>{{.Porter.OverallAttractiveness}}</strong></div></div>
</div>
<div class="summary-box"><div class="summary-title">Síntese Estratégica</div><div class="summary-text">{{.Porter.Summary}}</div></div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("porter").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generateTAMPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; position: relative; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.market-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 20px; margin-bottom: 20px; }
.market-card { border: 2px solid #E5E0D6; padding: 24px; text-align: center; }
.market-label { font-size: 10px; font-weight: 600; text-transform: uppercase; color: #71717A; margin-bottom: 8px; }
.market-value { font-size: 32px; font-weight: 300; color: #B89E68; }
.market-desc { font-size: 8px; color: #71717A; margin-top: 8px; }
.info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
.info-box { border: 1px solid #E5E0D6; padding: 16px; }
.info-title { font-size: 11px; font-weight: 700; color: #0A101D; margin-bottom: 10px; }
.info-list { list-style: none; }
.info-item { font-size: 9.5px; line-height: 1.5; color: #111827; margin-bottom: 6px; padding-left: 12px; position: relative; }
.info-item::before { content: '•'; position: absolute; left: 0; color: #B89E68; font-weight: 700; }
.cagr-badge { display: inline-block; background: #B89E68; color: white; padding: 4px 12px; border-radius: 4px; font-size: 10px; font-weight: 600; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">03</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">TAM / SAM / SOM</h1>
<div class="page-subtitle">Tamanho de Mercado</div>
<div class="market-grid">
<div class="market-card"><div class="market-label">TAM</div><div class="market-value">{{.TAM.TAM}}</div><div class="market-desc">Total Addressable Market</div></div>
<div class="market-card"><div class="market-label">SAM</div><div class="market-value">{{.TAM.SAM}}</div><div class="market-desc">Serviceable Available Market</div></div>
<div class="market-card"><div class="market-label">SOM</div><div class="market-value">{{.TAM.SOM}}</div><div class="market-desc">Serviceable Obtainable Market</div></div>
</div>
<div class="info-grid">
<div class="info-box"><div class="info-title">Premissas de Mercado</div><ul class="info-list">{{range .TAM.Assumptions}}<li class="info-item">{{.}}</li>{{end}}</ul></div>
<div class="info-box"><div class="info-title">Crescimento Projetado</div><p style="font-size: 10px; line-height: 1.6; color: #111827;">{{.TAM.Summary}}</p><div style="margin-top: 12px;"><span class="cagr-badge">CAGR: {{.TAM.CAGR}}</span></div></div>
</div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("tam").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generateSWOTPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; position: relative; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.swot-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
.swot-box { border: 2px solid #E5E0D6; padding: 16px; height: 140px; }
.swot-title { font-size: 11px; font-weight: 700; color: #B89E68; margin-bottom: 10px; }
.swot-list { list-style: none; }
.swot-item { font-size: 9px; line-height: 1.4; color: #111827; margin-bottom: 5px; padding-left: 12px; position: relative; }
.swot-item::before { content: '•'; position: absolute; left: 0; color: #B89E68; font-weight: 700; }
.summary-box { border: 2px solid #B89E68; padding: 16px; background: #FAFAF9; }
.summary-text { font-size: 9.5px; line-height: 1.6; color: #111827; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">04</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">Análise SWOT</h1>
<div class="page-subtitle">Fatores Internos e Externos</div>
<div class="swot-grid">
<div class="swot-box"><div class="swot-title">Forças (Strengths)</div><ul class="swot-list">{{range .SWOT.Strengths}}<li class="swot-item">{{.}}</li>{{end}}</ul></div>
<div class="swot-box"><div class="swot-title">Fraquezas (Weaknesses)</div><ul class="swot-list">{{range .SWOT.Weaknesses}}<li class="swot-item">{{.}}</li>{{end}}</ul></div>
<div class="swot-box"><div class="swot-title">Oportunidades (Opportunities)</div><ul class="swot-list">{{range .SWOT.Opportunities}}<li class="swot-item">{{.}}</li>{{end}}</ul></div>
<div class="swot-box"><div class="swot-title">Ameaças (Threats)</div><ul class="swot-list">{{range .SWOT.Threats}}<li class="swot-item">{{.}}</li>{{end}}</ul></div>
</div>
<div class="summary-box"><div class="summary-text">{{.SWOT.Summary}}</div></div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("swot").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generateBlueOceanPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; position: relative; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.errc-grid { display: grid; grid-template-columns: 1fr 1fr 1fr 1fr; gap: 12px; margin-bottom: 16px; }
.errc-box { border: 2px solid #E5E0D6; padding: 14px; height: 140px; }
.errc-title { font-size: 10px; font-weight: 700; color: #B89E68; margin-bottom: 8px; }
.errc-list { list-style: none; }
.errc-item { font-size: 8.5px; line-height: 1.4; color: #111827; margin-bottom: 4px; padding-left: 10px; position: relative; }
.errc-item::before { content: '•'; position: absolute; left: 0; color: #B89E68; font-weight: 700; }
.value-box { border: 2px solid #B89E68; padding: 16px; background: #FAFAF9; margin-bottom: 10px; }
.value-title { font-size: 10px; font-weight: 700; color: #0A101D; margin-bottom: 8px; }
.value-text { font-size: 9.5px; line-height: 1.5; color: #111827; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">05</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">Blue Ocean Strategy</h1>
<div class="page-subtitle">Framework ERRC</div>
<div class="errc-grid">
<div class="errc-box"><div class="errc-title">Eliminar</div><ul class="errc-list">{{range .BlueOcean.Eliminate}}<li class="errc-item">{{.}}</li>{{end}}</ul></div>
<div class="errc-box"><div class="errc-title">Reduzir</div><ul class="errc-list">{{range .BlueOcean.Reduce}}<li class="errc-item">{{.}}</li>{{end}}</ul></div>
<div class="errc-box"><div class="errc-title">Elevar</div><ul class="errc-list">{{range .BlueOcean.Raise}}<li class="errc-item">{{.}}</li>{{end}}</ul></div>
<div class="errc-box"><div class="errc-title">Criar</div><ul class="errc-list">{{range .BlueOcean.Create}}<li class="errc-item">{{.}}</li>{{end}}</ul></div>
</div>
<div class="value-box"><div class="value-title">Nova Curva de Valor</div><div class="value-text">{{.BlueOcean.NewValueCurve}}</div></div>
<div class="value-box"><div class="value-title">Síntese Estratégica</div><div class="value-text">{{.BlueOcean.Summary}}</div></div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("blue_ocean").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generateScenariosPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; position: relative; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.scenarios-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 14px; margin-bottom: 14px; }
.scenario-box { border: 2px solid #E5E0D6; padding: 14px; height: 130px; }
.scenario-title { font-size: 11px; font-weight: 700; color: #B89E68; margin-bottom: 8px; }
.scenario-text { font-size: 9px; line-height: 1.5; color: #111827; }
.warning-box { border: 2px solid #B89E68; padding: 16px; background: #FAFAF9; }
.warning-title { font-size: 10px; font-weight: 700; color: #0A101D; margin-bottom: 10px; }
.warning-list { list-style: none; }
.warning-item { font-size: 9px; line-height: 1.4; color: #111827; margin-bottom: 4px; padding-left: 12px; position: relative; }
.warning-item::before { content: '⚠'; position: absolute; left: 0; color: #B89E68; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">06</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">Planejamento de Cenários</h1>
<div class="page-subtitle">Futuros Alternativos</div>
<div class="scenarios-grid">
<div class="scenario-box"><div class="scenario-title">Cenário Otimista</div><div class="scenario-text">{{.Scenarios.Optimistic}}</div></div>
<div class="scenario-box"><div class="scenario-title">Cenário Realista</div><div class="scenario-text">{{.Scenarios.Realist}}</div></div>
<div class="scenario-box"><div class="scenario-title">Cenário Pessimista</div><div class="scenario-text">{{.Scenarios.Pessimistic}}</div></div>
</div>
<div class="warning-box"><div class="warning-title">Sinais de Alerta Precoce</div><ul class="warning-list">{{range .Scenarios.EarlyWarningSignals}}<li class="warning-item">{{.}}</li>{{end}}</ul><div style="margin-top: 12px; padding-top: 12px; border-top: 1px solid #E5E0D6; font-size: 9px; color: #111827;">{{.Scenarios.Summary}}</div></div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("scenarios").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generateOKRsPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; position: relative; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.okrs-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
.okr-box { border: 2px solid #E5E0D6; padding: 20px; }
.okr-title { font-size: 14px; font-weight: 700; color: #B89E68; margin-bottom: 14px; }
.kr-list { list-style: none; }
.kr-item { font-size: 10px; line-height: 1.6; color: #111827; margin-bottom: 10px; padding-left: 16px; position: relative; }
.kr-item::before { content: '→'; position: absolute; left: 0; color: #B89E68; font-weight: 700; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">07</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">OKRs</h1>
<div class="page-subtitle">Objectives & Key Results</div>
<div class="okrs-grid">
{{range .OKRs.Objectives}}
<div class="okr-box">
<div class="okr-title">{{.Title}}</div>
<ul class="kr-list">
{{range .KeyResults}}<li class="kr-item">{{.}}</li>{{end}}
</ul>
</div>
{{end}}
</div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("okrs").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generateBSCPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; position: relative; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.bsc-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 14px; }
.bsc-box { border: 2px solid #E5E0D6; padding: 16px; height: 125px; }
.bsc-title { font-size: 11px; font-weight: 700; color: #B89E68; margin-bottom: 10px; }
.bsc-list { list-style: none; }
.bsc-item { font-size: 9px; line-height: 1.5; color: #111827; margin-bottom: 6px; padding-left: 12px; position: relative; }
.bsc-item::before { content: '•'; position: absolute; left: 0; color: #B89E68; font-weight: 700; }
.summary-box { border: 2px solid #B89E68; padding: 14px; background: #FAFAF9; }
.summary-text { font-size: 9.5px; line-height: 1.5; color: #111827; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">08</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">Balanced Scorecard</h1>
<div class="page-subtitle">Perspectivas Estratégicas</div>
<div class="bsc-grid">
<div class="bsc-box"><div class="bsc-title">Perspectiva Financeira</div><ul class="bsc-list">{{range .BSC.Financial}}<li class="bsc-item">{{.}}</li>{{end}}</ul></div>
<div class="bsc-box"><div class="bsc-title">Perspectiva do Cliente</div><ul class="bsc-list">{{range .BSC.Customer}}<li class="bsc-item">{{.}}</li>{{end}}</ul></div>
<div class="bsc-box"><div class="bsc-title">Processos Internos</div><ul class="bsc-list">{{range .BSC.InternalProcesses}}<li class="bsc-item">{{.}}</li>{{end}}</ul></div>
<div class="bsc-box"><div class="bsc-title">Aprendizado e Crescimento</div><ul class="bsc-list">{{range .BSC.LearningGrowth}}<li class="bsc-item">{{.}}</li>{{end}}</ul></div>
</div>
<div class="summary-box"><div class="summary-text">{{.BSC.Summary}}</div></div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("bsc").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

func generateSynthesisPage(data ReportData) string {
	tmpl := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
@page { size: A4 landscape; margin: 0; }
body { width: 842px; height: 595px; font-family: -apple-system, sans-serif; background: #FFFFFF; }
.page { width: 842px; height: 595px; padding: 40px; position: relative; }
.header { display: flex; justify-content: space-between; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 1px solid #E5E0D6; }
.page-number { font-size: 10px; font-weight: 600; color: #B89E68; }
.page-title { font-size: 22px; font-weight: 300; color: #0A101D; margin-bottom: 4px; }
.page-subtitle { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.15em; color: #B89E68; margin-bottom: 20px; }
.summary-section { margin-bottom: 16px; }
.section-title { font-size: 11px; font-weight: 700; color: #B89E68; margin-bottom: 8px; }
.section-text { font-size: 9.5px; line-height: 1.6; color: #111827; margin-bottom: 10px; }
.section-list { list-style: none; }
.section-item { font-size: 9px; line-height: 1.5; color: #111827; margin-bottom: 5px; padding-left: 12px; position: relative; }
.section-item::before { content: '•'; position: absolute; left: 0; color: #B89E68; font-weight: 700; }
.recommendation-box { border: 2px solid #B89E68; padding: 16px; background: #FAFAF9; }
.recommendation-text { font-size: 10px; font-weight: 600; line-height: 1.6; color: #0A101D; }
.footer { position: absolute; bottom: 40px; left: 40px; right: 40px; padding-top: 10px; border-top: 1px solid #E5E0D6; display: flex; justify-content: space-between; font-size: 8px; color: #71717A; }
</style>
</head><body><div class="page">
<div class="header"><div class="page-number">09</div><div class="company-name-header">{{.CompanyName}}</div></div>
<h1 class="page-title">Síntese Executiva</h1>
<div class="page-subtitle">Conclusões e Recomendações</div>
<div class="summary-section">
<div class="section-text">{{.Synthesis.ExecutiveSummary}}</div>
</div>
<div class="summary-section">
<div class="section-title">Principais Descobertas</div>
<ul class="section-list">{{range .Synthesis.KeyFindings}}<li class="section-item">{{.}}</li>{{end}}</ul>
</div>
<div class="summary-section">
<div class="section-title">Prioridades Estratégicas</div>
<ul class="section-list">{{range .Synthesis.StrategicPriorities}}<li class="section-item">{{.}}</li>{{end}}</ul>
</div>
<div class="summary-section">
<div class="section-title">Roadmap de Implementação</div>
<ul class="section-list">{{range .Synthesis.Roadmap}}<li class="section-item">{{.}}</li>{{end}}</ul>
</div>
<div class="recommendation-box">
<div class="recommendation-text">{{.Synthesis.OverallRecommendation}}</div>
</div>
<div class="footer"><div>IMENSIAH — Relatório de Inteligência Estratégica</div><div>{{.Date}}</div></div>
</div></body></html>`

	t := template.Must(template.New("synthesis").Parse(tmpl))
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}
