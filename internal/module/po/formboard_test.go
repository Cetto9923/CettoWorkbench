package po

import (
	"testing"
	"time"
)

func TestBoardTaskReqValidatePreservesTeamgroup(t *testing.T) {
	req := BoardTaskReq{TeamgroupID: 17, StatusFilter: "doing", Focus: "overdue"}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("Validate() errors = %v", errs)
	}
	if req.TeamgroupID != 17 {
		t.Fatalf("TeamgroupID = %d, want 17", req.TeamgroupID)
	}
}

func TestBoardTaskReqValidateRejectsUnsupportedStatus(t *testing.T) {
	req := BoardTaskReq{StatusFilter: "closed"}
	if errs := req.Validate(); len(errs) == 0 {
		t.Fatal("Validate() accepted a status outside the three-column board")
	}
}

func TestTaskColumnKeyAndOverdueRule(t *testing.T) {
	if got := taskColumnKey("pause"); got != "doing" {
		t.Fatalf("pause column = %q, want doing", got)
	}
	deadline := mustBoardDate(t, "2026-09-03")
	today := mustBoardDate(t, "2026-09-04")
	if !taskIsOverdue("doing", &deadline, today) {
		t.Fatal("unfinished task before its deadline should be overdue")
	}
	if taskIsOverdue("done", &deadline, today) {
		t.Fatal("completed task must not be overdue")
	}
}

func mustBoardDate(t *testing.T, value string) (out time.Time) {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	return parsed
}
