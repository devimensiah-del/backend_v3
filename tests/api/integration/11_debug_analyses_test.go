package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestDebugAnalyses lists all analyses to find any we can use for wizard testing
func TestDebugAnalyses(t *testing.T) {
	client := NewTestClient(t)
	RequireAdminAuth(t, client)

	// Try to get analyses via admin endpoint
	resp, err := client.GET("/api/v1/admin/submissions?pageSize=50")
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	var listResp map[string]interface{}
	resp.JSON(&listResp)
	submissions := listResp["data"].([]interface{})

	t.Logf("Checking %d submissions for analyses:\n", len(submissions))

	for _, sub := range submissions {
		subMap := sub.(map[string]interface{})
		subID := fmt.Sprintf("%v", subMap["id"])
		status := fmt.Sprintf("%v", subMap["status"])
		companyName := fmt.Sprintf("%v", subMap["companyName"])

		var analysisID string
		if aid, ok := subMap["analysisId"].(string); ok && aid != "" {
			analysisID = aid
		}

		t.Logf("  [%s] %s - %s (analysis: %v)", status, companyName, subID[:8], analysisID)

		if analysisID != "" {
			// Get analysis details
			analysisResp, err := client.GET("/api/v1/admin/analysis/" + analysisID)
			if err != nil {
				t.Logf("    Failed to get analysis: %v", err)
				continue
			}

			if analysisResp.StatusCode == 200 {
				var analysis map[string]interface{}
				analysisResp.JSON(&analysis)
				prettyJSON, _ := json.MarshalIndent(analysis, "    ", "  ")
				t.Logf("    Analysis details:\n%s", string(prettyJSON))
			}
		}
	}
}

// TestDebugWizardState checks if wizard can be started and what's happening
func TestDebugWizardState(t *testing.T) {
	client := NewTestClient(t)
	RequireAdminAuth(t, client)

	// Get first enriched submission
	resp, err := client.GET("/api/v1/admin/submissions?pageSize=10")
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	var listResp map[string]interface{}
	resp.JSON(&listResp)
	submissions := listResp["data"].([]interface{})

	var targetSubmission map[string]interface{}
	for _, sub := range submissions {
		subMap := sub.(map[string]interface{})
		status := fmt.Sprintf("%v", subMap["status"])
		if status == "enriched" {
			targetSubmission = subMap
			break
		}
	}

	if targetSubmission == nil {
		t.Fatal("No enriched submission found")
	}

	subID := fmt.Sprintf("%v", targetSubmission["id"])
	t.Logf("Target submission: %s", subID)
	t.Logf("Company Name: %v", targetSubmission["companyName"])
	t.Logf("Company ID: %v", targetSubmission["companyId"])
	t.Logf("Challenge ID: %v", targetSubmission["challengeId"])
	t.Logf("Analysis ID: %v", targetSubmission["analysisId"])
	t.Logf("Status: %v", targetSubmission["status"])

	// Log the full submission
	prettyJSON, _ := json.MarshalIndent(targetSubmission, "", "  ")
	t.Logf("Full submission data:\n%s", prettyJSON)

	// Try to start wizard
	t.Log("\n=== Attempting wizard start ===")
	// Get company_id and challenge_id from submission
	companyID := fmt.Sprintf("%v", targetSubmission["companyId"])
	challengeID := fmt.Sprintf("%v", targetSubmission["challengeId"])

	wizardResp, err := client.POST("/api/v1/wizard/start", map[string]string{
		"company_id":   companyID,
		"challenge_id": challengeID,
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	t.Logf("Wizard start response status: %d", wizardResp.StatusCode)
	t.Logf("Response body: %s", string(wizardResp.Body))
}
