package netguard

import (
	"context"
	"testing"
)

func TestResolve(t *testing.T) {
	ctx := context.Background()
	for _, h := range []string{"127.0.0.1", "10.1.2.3", "192.168.1.1", "169.254.169.254", "0.0.0.0", "::1", "100.64.1.1", "localhost"} {
		if _, err := Resolve(ctx, h, false); err == nil {
			t.Errorf("%s: expected rejection", h)
		}
	}
	if ip, err := Resolve(ctx, "146.70.41.142", false); err != nil || ip != "146.70.41.142" {
		t.Errorf("public IP rejected: %v", err)
	}
	if _, err := Resolve(ctx, "10.1.2.3", true); err != nil {
		t.Errorf("private allowed by flag but rejected: %v", err)
	}
	if _, err := Resolve(ctx, "169.254.169.254", true); err == nil {
		t.Error("metadata address must stay blocked even with allowPrivate")
	}
}
