package agileteam

import "testing"

func TestResolveViewModeHonorsScopedLeadDeepLink(t *testing.T) {
	if got := resolveViewMode("lead", true); got != "lead" {
		t.Fatalf("explicit lead view = %q, want lead", got)
	}
	if got := resolveViewMode("", true); got != "pmo" {
		t.Fatalf("default privileged view = %q, want pmo", got)
	}
	if got := resolveViewMode("pmo", false); got != "lead" {
		t.Fatalf("unprivileged view = %q, want lead", got)
	}
}
