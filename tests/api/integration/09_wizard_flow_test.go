package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestWizardFullFlow runs through the complete wizard workflow
// This test actually triggers AI analysis calls - use sparingly
func TestWizardFullFlow(t *testing.T) {
	client := NewTestClient(t)
	RequireAdminAuth(t, client)

	// Step 1: Get an enriched submission to find company + challenge IDs
	t.Log("=== STEP 1: Finding company and challenge to analyze ===")

	resp, err := client.GET("/api/v1/admin/submissions?pageSize=10")
	if err != nil {
		t.Fatalf("Failed to list submissions: %v", err)
	}

	var listResp map[string]interface{}
	if err := resp.JSON(&listResp); err != nil {
		t.Fatalf("Failed to parse submissions: %v", err)
	}

	submissions, ok := listResp["data"].([]interface{})
	if !ok || len(submissions) == 0 {
		t.Fatal("No submissions found to analyze")
	}

	// Find a submission with status "enriched" (ready for wizard)
	var companyID string
	var challengeID string
	var companyName string
	for _, sub := range submissions {
		subMap := sub.(map[string]interface{})
		status := fmt.Sprintf("%v", subMap["status"])
		// enriched status means company enrichment is complete and ready for wizard
		if status == "enriched" {
			companyName = fmt.Sprintf("%v", subMap["companyName"])
			if cid, ok := subMap["companyId"].(string); ok {
				companyID = cid
			}
			if chid, ok := subMap["challengeId"].(string); ok {
				challengeID = chid
			}
			break
		}
	}

	if companyID == "" || challengeID == "" {
		// Use the first submission anyway
		subMap := submissions[0].(map[string]interface{})
		companyName = fmt.Sprintf("%v", subMap["companyName"])
		if cid, ok := subMap["companyId"].(string); ok {
			companyID = cid
		}
		if chid, ok := subMap["challengeId"].(string); ok {
			challengeID = chid
		}
		t.Logf("No enriched submissions found, using first one")
	}

	t.Logf("Company: %s", companyName)
	t.Logf("Company ID: %s", companyID)
	t.Logf("Challenge ID: %s", challengeID)

	if companyID == "" || challengeID == "" {
		t.Fatal("Could not find company_id and challenge_id")
	}

	// Step 2: Start the wizard with company_id + challenge_id
	t.Log("\n=== STEP 2: Starting wizard ===")

	startResp, err := client.POST("/api/v1/wizard/start", map[string]string{
		"company_id":   companyID,
		"challenge_id": challengeID,
	})
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	t.Logf("Start wizard response status: %d", startResp.StatusCode)
	t.Logf("Response: %s", string(startResp.Body))

	if startResp.StatusCode != http.StatusOK && startResp.StatusCode != http.StatusCreated {
		t.Logf("Could not start wizard (status %d) - might already be in progress or completed", startResp.StatusCode)
		// Try to get current wizard state instead
	}

	var wizardState map[string]interface{}
	if err := startResp.JSON(&wizardState); err != nil {
		t.Fatalf("Failed to parse wizard state: %v", err)
	}

	// Check for state wrapper
	state := wizardState
	if stateObj, ok := wizardState["state"].(map[string]interface{}); ok {
		state = stateObj
	}

	analysisID, ok := state["analysis_id"].(string)
	if !ok {
		t.Fatalf("No analysis_id in response: %v", wizardState)
	}

	currentStep := int(state["current_step"].(float64))
	totalSteps := int(state["total_steps"].(float64))
	stepStatus := fmt.Sprintf("%v", state["step_status"])

	t.Logf("Analysis ID: %s", analysisID)
	t.Logf("Current step: %d/%d", currentStep, totalSteps)
	t.Logf("Step status: %s", stepStatus)

	if framework, ok := state["framework"].(map[string]interface{}); ok {
		t.Logf("Framework: %v (%v)", framework["name"], framework["code"])
	}

	// Step 3: Run through wizard steps (limit to 3 steps for testing)
	maxStepsToTest := 3
	stepsCompleted := 0

	for step := currentStep; step < totalSteps && stepsCompleted < maxStepsToTest; step++ {
		t.Logf("\n=== STEP 3.%d: Processing wizard step %d ===", stepsCompleted+1, step)

		// 3a: Generate the step output
		t.Logf("Generating step %d...", step)
		startTime := time.Now()

		generateResp, err := client.POST("/api/v1/analyses/"+analysisID+"/wizard/generate", map[string]interface{}{
			"human_context": "Please analyze thoroughly",
			"answers":       map[string]string{},
		})

		generateDuration := time.Since(startTime)

		if err != nil {
			t.Fatalf("Failed to generate step %d: %v", step, err)
		}

		t.Logf("Generate response status: %d (took %v)", generateResp.StatusCode, generateDuration)

		if generateResp.StatusCode != http.StatusOK {
			t.Logf("Generate failed: %s", string(generateResp.Body))
			break
		}

		var generateState map[string]interface{}
		if err := generateResp.JSON(&generateState); err != nil {
			t.Fatalf("Failed to parse generate response: %v", err)
		}

		// Check for state wrapper
		genState := generateState
		if stateObj, ok := generateState["state"].(map[string]interface{}); ok {
			genState = stateObj
		}

		newStatus := fmt.Sprintf("%v", genState["step_status"])
		t.Logf("New step status: %s", newStatus)

		if output, ok := genState["output"].(map[string]interface{}); ok {
			// Log a preview of the output
			outputPreview := fmt.Sprintf("%v", output)
			if len(outputPreview) > 500 {
				outputPreview = outputPreview[:500] + "..."
			}
			t.Logf("Output preview: %s", outputPreview)
		}

		// 3b: Approve the step
		t.Logf("Approving step %d...", step)

		approveResp, err := client.POST("/api/v1/analyses/"+analysisID+"/wizard/approve", nil)
		if err != nil {
			t.Fatalf("Failed to approve step %d: %v", step, err)
		}

		t.Logf("Approve response status: %d", approveResp.StatusCode)

		if approveResp.StatusCode != http.StatusOK {
			t.Logf("Approve failed: %s", string(approveResp.Body))
			break
		}

		var approveState map[string]interface{}
		if err := approveResp.JSON(&approveState); err != nil {
			t.Fatalf("Failed to parse approve response: %v", err)
		}

		// Check for state wrapper
		appState := approveState
		if stateObj, ok := approveState["state"].(map[string]interface{}); ok {
			appState = stateObj
		}

		newCurrentStep := int(appState["current_step"].(float64))
		t.Logf("Approved, now on step %d", newCurrentStep)

		if newFramework, ok := appState["framework"].(map[string]interface{}); ok {
			t.Logf("Next framework: %v (%v)", newFramework["name"], newFramework["code"])
		}

		stepsCompleted++
	}

	// Step 4: Get wizard summary
	t.Log("\n=== STEP 4: Getting wizard summary ===")

	summaryResp, err := client.GET("/api/v1/analyses/" + analysisID + "/wizard/summary")
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}

	t.Logf("Summary response status: %d", summaryResp.StatusCode)

	if summaryResp.StatusCode == http.StatusOK {
		var summary map[string]interface{}
		if err := summaryResp.JSON(&summary); err == nil {
			if results, ok := summary["framework_results"].(map[string]interface{}); ok {
				t.Logf("Completed frameworks: %d", len(results))
				for code := range results {
					t.Logf("  - %s", code)
				}
			}
		}
	}

	t.Logf("\n=== WIZARD FLOW TEST COMPLETE ===")
	t.Logf("Steps completed: %d", stepsCompleted)
}
