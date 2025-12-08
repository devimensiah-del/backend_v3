package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestDebugSubmissions lists all submissions with full details
func TestDebugSubmissions(t *testing.T) {
	client := NewTestClient(t)
	RequireAdminAuth(t, client)

	resp, err := client.GET("/api/v1/admin/submissions?pageSize=20")
	if err != nil {
		t.Fatalf("Failed to list submissions: %v", err)
	}

	var listResp map[string]interface{}
	if err := resp.JSON(&listResp); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	submissions := listResp["data"].([]interface{})
	t.Logf("Found %d submissions:\n", len(submissions))

	for i, sub := range submissions {
		subMap := sub.(map[string]interface{})
		prettyJSON, _ := json.MarshalIndent(subMap, "", "  ")
		t.Logf("\n=== Submission %d ===\n%s\n", i+1, string(prettyJSON))
	}
}

// TestDebugCompanies lists all companies
func TestDebugCompanies(t *testing.T) {
	client := NewTestClient(t)
	RequireAdminAuth(t, client)

	resp, err := client.GET("/api/v1/admin/companies?limit=20")
	if err != nil {
		t.Fatalf("Failed to list companies: %v", err)
	}

	var listResp map[string]interface{}
	if err := resp.JSON(&listResp); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	companies := listResp["companies"].([]interface{})
	t.Logf("Found %d companies:\n", len(companies))

	for i, co := range companies {
		coMap := co.(map[string]interface{})
		t.Logf("\n=== Company %d ===", i+1)
		t.Logf("  ID: %v", coMap["id"])
		t.Logf("  Name: %v", coMap["name"])
		t.Logf("  Enrichment: %v", coMap["enrichmentStatus"])
		t.Logf("  Owner: %v", coMap["ownerId"])
	}
}

// TestGetSubmissionDetail gets details for a specific submission
func TestGetSubmissionDetail(t *testing.T) {
	client := NewTestClient(t)
	RequireAdminAuth(t, client)

	// First get list of submissions
	resp, err := client.GET("/api/v1/admin/submissions?pageSize=10")
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	var listResp map[string]interface{}
	resp.JSON(&listResp)
	submissions := listResp["data"].([]interface{})

	for _, sub := range submissions {
		subMap := sub.(map[string]interface{})
		subID := fmt.Sprintf("%v", subMap["id"])

		// Get full details
		detailResp, err := client.GET("/api/v1/admin/submissions/" + subID)
		if err != nil {
			t.Logf("Failed to get %s: %v", subID, err)
			continue
		}

		var detail map[string]interface{}
		detailResp.JSON(&detail)

		if submission, ok := detail["submission"].(map[string]interface{}); ok {
			t.Logf("\n=== Submission: %s ===", subID)
			t.Logf("  Company: %v", submission["companyName"])
			t.Logf("  CompanyID: %v", submission["companyId"])
			t.Logf("  ChallengeID: %v", submission["challengeId"])
			t.Logf("  AnalysisID: %v", submission["analysisId"])
			t.Logf("  Status: %v", submission["status"])
		}
	}
}
