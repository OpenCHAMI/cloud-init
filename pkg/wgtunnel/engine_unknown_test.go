package wgtunnel

import "testing"

func TestSelectEngineUnknown(t *testing.T) {
	if _, err := SelectEngine(EngineType("not-a-real-engine"), false); err == nil {
		t.Fatalf("expected error for unknown engine type, got nil")
	}
}
