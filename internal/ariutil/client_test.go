package ariutil

import "testing"

func TestNewARIClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	cfg := ConfigFromEnv()
	if err := cfg.Validate(); err != nil {
		t.Skipf("ARI env vars not fully set, skipping integration test: %v", err)
	}

	cl, err := NewARIClient(cfg)
	if err != nil {
		t.Fatalf("NewARIClient failed: %v", err)
	}
	defer cl.Close()

	info, err := cl.Asterisk().Info(nil)
	if err != nil {
		t.Fatalf("failed to query Asterisk info: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil Asterisk info")
	}
}
