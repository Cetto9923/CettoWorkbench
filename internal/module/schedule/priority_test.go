package schedule

import "testing"

func TestFormatPriorityBlankForUnset(t *testing.T) {
	label, class := formatPriority(0)
	if label != "" || class != "" {
		t.Fatalf("expected empty priority for unset value, got label=%q class=%q", label, class)
	}

	label, class = formatPriority(3)
	if label != "P3" || class != "p3" {
		t.Fatalf("expected P3/p3, got label=%q class=%q", label, class)
	}
}

func TestFormatTaskPriorityBlankForUnset(t *testing.T) {
	if got := formatTaskPriority(0); got != "" {
		t.Fatalf("expected empty task priority for unset value, got %q", got)
	}

	if got := formatTaskPriority(3); got != "P3" {
		t.Fatalf("expected P3, got %q", got)
	}
}
