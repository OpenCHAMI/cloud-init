package wgtunnel

import "testing"

func TestSelectEngineExplicit(t *testing.T) {
	if _, err := SelectEngine(EngineKernel, false); err != nil {
		t.Fatalf("kernel engine selection failed: %v", err)
	}
	if _, err := SelectEngine(EngineUserspace, false); err != nil {
		t.Fatalf("userspace engine selection failed: %v", err)
	}
}

func TestSelectEngineAuto(t *testing.T) {
	eng, err := SelectEngine(EngineAuto, false)
	if err != nil {
		t.Fatalf("auto engine selection failed: %v", err)
	}
	if eng == nil {
		t.Fatalf("auto engine returned nil engine")
	}

	eng2, err := SelectEngine(EngineAuto, true)
	if err != nil {
		t.Fatalf("auto+fips engine selection failed: %v", err)
	}
	if eng2 == nil {
		t.Fatalf("auto+fips returned nil engine")
	}
}
