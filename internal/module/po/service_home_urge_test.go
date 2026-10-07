package po

import (
	"strings"
	"testing"
)

func TestHomeAcceptanceRecipients(t *testing.T) {
	got := homeAcceptanceRecipients(&homeActionDemandRow{RD: "alice", Accepter: "bob", VeriFier: "carol"})
	want := []string{"alice"}
	if len(got) != len(want) {
		t.Fatalf("recipients = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("recipients = %#v, want %#v", got, want)
		}
	}
}

func TestUrgeHomeDemandReqDefaultsToAcceptanceInApp(t *testing.T) {
	req := UrgeHomeDemandReq{UrgeType: "remind_accept"}
	req.normalize()
	if req.UrgeType != "accept" || len(req.Channels) != 1 || req.Channels[0] != "inapp" {
		t.Fatalf("normalized req = %#v", req)
	}
}

func TestCanUrgeHomeAcceptance(t *testing.T) {
	repo := NewRepo(nil, nil)
	row := &homeActionDemandRow{Status: "waitacceptance", AssignedTo: "other", RD: "acceptor", AcceptanceInitiator: "owner", VeriFier: "owner"}
	if !repo.canUrgeHomeAcceptance(row, "owner") {
		t.Fatal("latest initiator should be allowed even without another participant role")
	}
	if repo.canUrgeHomeAcceptance(row, "acceptor") || repo.canUrgeHomeAcceptance(row, "verifier") {
		t.Fatal("acceptance recipients must not remind themselves")
	}
	if repo.canUrgeHomeAcceptance(row, "other") {
		t.Fatal("assigned participant without startacceptance must not urge")
	}
	row.RD = "owner"
	if repo.canUrgeHomeAcceptance(row, "owner") {
		t.Fatal("initiator who is also acceptance owner must not urge himself")
	}
	row.RD = "acceptor"
	row.Status = "active"
	if repo.canUrgeHomeAcceptance(row, "owner") {
		t.Fatal("non-acceptance stage must reject urge")
	}
}

func TestAcceptanceUrgeIsDuplicate(t *testing.T) {
	recipients := []string{"acceptor", "verifier"}
	if acceptanceUrgeIsDuplicate(1, recipients) {
		t.Fatal("one recipient notification is not a complete duplicate")
	}
	if !acceptanceUrgeIsDuplicate(2, recipients) {
		t.Fatal("all recipients notified within the window must be duplicate")
	}

}

func TestBuildAcceptanceUrgeMessage(t *testing.T) {
	msg := buildAcceptanceUrgeMessage(&homeActionDemandRow{ID: 63411, Name: "测试0819", Status: "waitacceptance"}, []string{"张三(001)"}, "优先处理")
	for _, part := range []string{"US63411", "测试0819", "待验收", "张三(001)", "优先处理", "催办验收"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("message missing %q: %q", part, msg)
		}
	}
}

func TestHomeAcceptanceRecipientsFallback(t *testing.T) {
	got := homeAcceptanceRecipients(&homeActionDemandRow{AssignedTo: "assigned", Originator: "ori", CreatedBy: "creator"})
	if len(got) != 1 || got[0] != "assigned" {
		t.Fatalf("fallback = %#v", got)
	}
	if got := homeAcceptanceRecipients(&homeActionDemandRow{Originator: "ori", CreatedBy: "creator"}); len(got) != 0 {
		t.Fatalf("unexpected fallback = %#v", got)
	}
}
