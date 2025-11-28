package analysis

import "testing"

// TestStatusConstantsExist verifies that all expected status constants are defined
// Simplified flow: pending → completed
func TestStatusConstantsExist(t *testing.T) {
	expected := map[Status]bool{
		StatusPending:   true,
		StatusCompleted: true,
	}

	for s := range expected {
		if s == "" {
			t.Fatalf("status constant is empty")
		}
	}
}
