package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestCoimmaFullFlow creates a new submission for Coimma and runs through all wizard steps
func TestCoimmaFullFlow(t *testing.T) {
	client := NewTestClient(t)
	RequireAdminAuth(t, client)

	t.Logf("\n========================================")
	t.Logf("COIMMA FULL FLOW - FROM SUBMISSION TO ANALYSIS")
	t.Logf("========================================\n")

	// Step 1: Create submission for Coimma
	t.Logf("=== STEP 1: CREATING SUBMISSION ===")

	additionalInfo := map[string]interface{}{
		"contactName":  "Daniela",
		"contactEmail": "ddaprado@coimma.com.br",
		"contactPhone": "+55 11 99999-9999",
	}
	additionalInfoJSON, _ := json.Marshal(additionalInfo)

	submissionResp, err := client.POST("/api/v1/submissions", map[string]interface{}{
		"companyName":       "Coimma",
		"challengeCategory": "compete",
		"challengeType":     "compete_differentiate",
		"businessChallenge": `Sou CEO da Coimma. Somos fabricantes de equipamentos de manejo para gado, balanças bovinas eletrônicas, balanças rodoviárias, automação de balanças rodoviárias e balanças de fluxo por batelada para grãos.

Atualmente 70% do nosso faturamento é proveniente do mercado pecuária. Os outros 30% são provenientes de segmentos diversos destacando-se grãos e armazenamento. A prestação de serviços de calibração e manutenção representa 5% do faturamento.

Nossas competências são fabricação de equipamentos, sistemas de pesagem e eletrônica.

Elabore um plano de negócios para nossa companhia em duas fases:
- Fase 1 (imediata): expandir vendas com portfólio atual e estrutura existente de vendas e assistência técnica
- Fase 2: desenvolver novos produtos para o segmento industrial com investimentos em estrutura de vendas e assistência técnica`,
		"industry":       "Agronegócio / Equipamentos Industriais",
		"companySize":    "medium",
		"additionalInfo": string(additionalInfoJSON),
	})

	if err != nil {
		t.Fatalf("Failed to create submission: %v", err)
	}

	if submissionResp.StatusCode != 201 {
		t.Fatalf("Submission failed with status %d: %s", submissionResp.StatusCode, string(submissionResp.Body))
	}

	var submissionData struct {
		Submission struct {
			ID          string `json:"id"`
			CompanyID   string `json:"companyId"`
			ChallengeID string `json:"challengeId"`
		} `json:"submission"`
	}
	if err := submissionResp.JSON(&submissionData); err != nil {
		t.Fatalf("Failed to parse submission: %v", err)
	}

	submissionID := submissionData.Submission.ID
	companyID := submissionData.Submission.CompanyID
	challengeID := submissionData.Submission.ChallengeID

	t.Logf("Submission created successfully!")
	t.Logf("  Submission ID: %s", submissionID)
	t.Logf("  Company ID: %s", companyID)
	t.Logf("  Challenge ID: %s", challengeID)

	// Wait a moment for enrichment to start
	time.Sleep(2 * time.Second)

	// Step 2: Start the wizard
	t.Logf("\n=== STEP 2: STARTING WIZARD ===")

	startResp, err := client.POST("/api/v1/wizard/start", map[string]interface{}{
		"company_id":   companyID,
		"challenge_id": challengeID,
	})
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	if startResp.StatusCode != 200 {
		t.Fatalf("Start wizard failed with status %d: %s", startResp.StatusCode, string(startResp.Body))
	}

	var startData struct {
		State struct {
			AnalysisID  string `json:"analysis_id"`
			CurrentStep int    `json:"current_step"`
			TotalSteps  int    `json:"total_steps"`
			Framework   struct {
				Code string `json:"code"`
				Name string `json:"name"`
			} `json:"framework"`
		} `json:"state"`
	}
	if err := startResp.JSON(&startData); err != nil {
		t.Fatalf("Failed to parse start response: %v", err)
	}

	analysisID := startData.State.AnalysisID
	totalSteps := startData.State.TotalSteps

	t.Logf("Wizard started successfully!")
	t.Logf("  Analysis ID: %s", analysisID)
	t.Logf("  Total Steps: %d", totalSteps)
	t.Logf("  First Framework: %s (%s)", startData.State.Framework.Name, startData.State.Framework.Code)

	// Framework names for display
	frameworkNames := map[int]string{
		0:  "Challenge Refinement",
		1:  "PESTEL",
		2:  "Porter's Forces",
		3:  "Benchmarking",
		4:  "SWOT",
		5:  "TAM-SAM-SOM",
		6:  "Blue Ocean",
		7:  "Growth Hacking",
		8:  "Scenarios",
		9:  "Decision Matrix",
		10: "OKRs",
		11: "BSC",
	}

	// Step 3: Process each wizard step
	for step := 0; step < totalSteps; step++ {
		stepName := frameworkNames[step]
		if stepName == "" {
			stepName = fmt.Sprintf("Step %d", step)
		}

		t.Logf("\n========================================")
		t.Logf("STEP %d/%d: %s", step+1, totalSteps, stepName)
		t.Logf("========================================")

		// Generate the step
		t.Logf("Generating analysis...")
		startTime := time.Now()

		genResp, err := client.POST(fmt.Sprintf("/api/v1/analyses/%s/wizard/generate", analysisID), nil)
		if err != nil {
			t.Logf("ERROR: Generate request failed: %v", err)
			continue
		}

		duration := time.Since(startTime)
		t.Logf("LLM Response: %.1fs (status: %d)", duration.Seconds(), genResp.StatusCode)

		if genResp.StatusCode != 200 {
			t.Logf("Generate error: %s", string(genResp.Body))
			continue
		}

		// Parse and display generated content
		var genData map[string]interface{}
		if err := json.Unmarshal(genResp.Body, &genData); err == nil {
			if output, ok := genData["generated_output"]; ok {
				outputJSON, _ := json.MarshalIndent(output, "", "  ")
				outputStr := string(outputJSON)
				if len(outputStr) > 1500 {
					outputStr = outputStr[:1500] + "\n  ... (truncated)"
				}
				t.Logf("\n--- GENERATED CONTENT ---\n%s\n--- END CONTENT ---", outputStr)
			}
		}

		// Approve the step
		t.Logf("\nApproving step %d...", step)
		approveResp, err := client.POST(fmt.Sprintf("/api/v1/analyses/%s/wizard/approve", analysisID), nil)
		if err != nil {
			t.Logf("ERROR: Approve request failed: %v", err)
			continue
		}

		if approveResp.StatusCode == 200 {
			t.Logf("Step %d APPROVED", step)

			// Check next step
			var approveData struct {
				State struct {
					CurrentStep int `json:"current_step"`
					Framework   struct {
						Name string `json:"name"`
					} `json:"framework"`
				} `json:"state"`
			}
			if json.Unmarshal(approveResp.Body, &approveData) == nil {
				if approveData.State.CurrentStep > step {
					t.Logf("Next: %s", approveData.State.Framework.Name)
				}
			}
		} else {
			t.Logf("Approve returned status %d: %s", approveResp.StatusCode, string(approveResp.Body))
		}

		// Brief pause between steps
		time.Sleep(500 * time.Millisecond)
	}

	// Step 4: Get final summary
	t.Logf("\n========================================")
	t.Logf("FINAL SUMMARY")
	t.Logf("========================================")

	summaryResp, err := client.GET(fmt.Sprintf("/api/v1/analyses/%s/wizard/summary", analysisID))
	if err != nil {
		t.Logf("Failed to get summary: %v", err)
	} else if summaryResp.StatusCode == 200 {
		var summary map[string]interface{}
		if json.Unmarshal(summaryResp.Body, &summary) == nil {
			prettyJSON, _ := json.MarshalIndent(summary, "", "  ")
			summaryStr := string(prettyJSON)
			if len(summaryStr) > 3000 {
				summaryStr = summaryStr[:3000] + "\n  ... (truncated)"
			}
			t.Logf("\n%s", summaryStr)
		}
	}

	t.Logf("\n========================================")
	t.Logf("COIMMA ANALYSIS COMPLETE!")
	t.Logf("========================================")
	t.Logf("Submission ID: %s", submissionID)
	t.Logf("Analysis ID: %s", analysisID)
	t.Logf("Company ID: %s", companyID)
	t.Logf("========================================")
}
