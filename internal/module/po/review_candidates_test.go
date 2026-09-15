package po

import "testing"

func TestNormalizeReviewerAccountsTrimsAndDeduplicates(t *testing.T) {
	got := normalizeReviewerAccounts([]string{" alice ", "", "bob", "alice", "bob "})
	want := []string{"alice", "bob"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestSplitReviewerAccountsSupportsZenTaoCSV(t *testing.T) {
	got := splitReviewerAccounts("alice, bob,alice,,")
	want := []string{"alice", "bob"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}
