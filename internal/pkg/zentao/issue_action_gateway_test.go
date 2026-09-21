package zentao

import (
	"context"
	"errors"
	"testing"
)

func TestUnavailableIssueActionGatewayFailsClosed(t *testing.T) {
	gw := NewUnavailableIssueActionGateway("API v1 has no issue action endpoint")
	err := gw.ExecuteIssueAction(context.Background(), IssueActionRequest{IssueID: 42, Action: IssueActionResolve, SessionID: "session"})
	if !errors.Is(err, ErrIssueActionUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestValidIssueAction(t *testing.T) {
	for _, raw := range []string{"resolve", "close", "activate"} {
		if _, ok := ValidIssueAction(raw); !ok {
			t.Fatalf("%q should be a valid issue action", raw)
		}
	}
	if _, ok := ValidIssueAction("delete"); ok {
		t.Fatal("delete must not be accepted as an issue action")
	}
}
