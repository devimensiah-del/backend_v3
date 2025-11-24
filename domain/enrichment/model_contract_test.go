package enrichment

import "testing"

func TestNewEnrichmentDefaultsPending(t *testing.T) {
	enrich := NewEnrichment(uuidNil)
	if enrich.Status != StatusPending {
		t.Fatalf("expected status pending, got %s", enrich.Status)
	}
}

var uuidNil = [16]byte{}
