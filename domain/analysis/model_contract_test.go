package analysis

import "testing"

func TestCreateNewVersionCopiesStatusAndIncrements(t *testing.T) {
	orig := &Analysis{
		ID:           "a1",
		SubmissionID: "s1",
		EnrichmentID: "e1",
		Status:       string(StatusCompleted),
		Version:      1,
		IsLatest:     true,
	}

	v2 := orig.CreateNewVersion()

	if v2.Version != 2 || v2.ParentAnalysisID == nil || *v2.ParentAnalysisID != "a1" {
		t.Fatalf("versioning fields not set correctly: %+v", v2)
	}
	if v2.Status != orig.Status {
		t.Fatalf("status should be copied from previous version, got %s", v2.Status)
	}
	if v2.IsLatest != true {
		t.Fatalf("new version should be marked latest")
	}
}
