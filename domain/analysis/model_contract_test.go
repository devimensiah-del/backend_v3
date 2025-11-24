package analysis

import "testing"

// TestStatusConstantsExist verifies that all expected status constants are defined
// Flow: pending → completed → approved → sent
func TestStatusConstantsExist(t *testing.T) {
	expected := map[Status]bool{
		StatusPending:   true,
		StatusCompleted: true,
		StatusApproved:  true,
		StatusSent:      true,
	}

	for s := range expected {
		if s == "" {
			t.Fatalf("status constant is empty")
		}
	}
}
