// =============================================================================
// 文件: internal/module/po/notice_classification_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 通知中心主题解析、邮件对象归类、分类与待办兜底及对象过滤测试。
// 依赖: testing, github.com/DATA-DOG/go-sqlmock
// =============================================================================

package po

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/config"
	"workbench/internal/pkg/zentao"
)

func TestParseNoticeSubject_Extended(t *testing.T) {
	cases := []struct {
		subject   string
		wantType  string
		wantID    int64
		wantClean string
	}{
		{"[CRCB] 需求 #63412 村镇贷款借据表", "demand", 63412, "村镇贷款借据表"},
		{"【CRCB】需求 #63412 村镇贷款借据表", "demand", 63412, "村镇贷款借据表"},
		{"DEMAND #10 测试需求", "demand", 10, "测试需求"},
		{"GUIDELINE #352 建设指引说明", "guideline", 352, "建设指引说明"},
		{"REVIEW #419 评审数据库设计说明书", "review", 419, "评审数据库设计说明书"},
		{"STORY #56220 保存草稿功能", "story", 56220, "保存草稿功能"},
		{"TASK #357 产品公测试点", "task", 357, "产品公测试点"},
		{"BUG #139 资源日历报错", "bug", 139, "资源日历报错"},
		{"反馈 #2159 导出字段", "feedback", 2159, "导出字段"},
		{"KANBANCARD #81 看板沟通", "kanbancard", 81, "看板沟通"},
		{"普通广播通知", "", 0, "普通广播通知"},
	}

	for _, tc := range cases {
		gotType, gotID, gotClean := parseNoticeSubject(tc.subject)
		if gotType != tc.wantType || gotID != tc.wantID || gotClean != tc.wantClean {
			t.Fatalf("parseNoticeSubject(%q) = (%q, %d, %q), want (%q, %d, %q)",
				tc.subject, gotType, gotID, gotClean, tc.wantType, tc.wantID, tc.wantClean)
		}
	}
}

func TestNewNoticeItem_MailDemandAndApproval(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})

	// 1. 业务需求邮件通知
	demandRow := noticeRow{
		ID:         1001,
		ObjectType: "mail",
		ObjectID:   0,
		Subject:    "DEMAND #10 测试需求",
		Data:       "<p>内容详情</p>",
	}
	itemDemand := newNoticeItem(demandRow, nil)
	if itemDemand.ObjectType != "demand" || itemDemand.ObjectID != 10 {
		t.Fatalf("demand mail item object = (%s, %d), want (demand, 10)", itemDemand.ObjectType, itemDemand.ObjectID)
	}
	if itemDemand.Category != "business" {
		t.Fatalf("demand mail item category = %s, want business", itemDemand.Category)
	}
	if !strings.Contains(itemDemand.URL, "demand") {
		t.Fatalf("demand mail item URL = %s, want containing demand", itemDemand.URL)
	}

	// 2. 审批邮件通知
	charterRow := noticeRow{
		ID:         1002,
		ObjectType: "mail",
		ObjectID:   0,
		Subject:    "CHARTER #529 云销管理平台",
		Data:       "<p>尊敬的用户，当前需要您进行审批，请前往禅道进行审批。</p>",
	}
	itemCharter := newNoticeItem(charterRow, nil)
	if itemCharter.ObjectType != "charter" || itemCharter.ObjectID != 529 {
		t.Fatalf("charter mail item object = (%s, %d), want (charter, 529)", itemCharter.ObjectType, itemCharter.ObjectID)
	}
	if itemCharter.Category != "approval" {
		t.Fatalf("charter mail item category = %s, want approval", itemCharter.Category)
	}
	if !itemCharter.NeedAction {
		t.Fatalf("charter mail item needAction = false, want true")
	}

	// 3. 时效提醒邮件通知
	reminderRow := noticeRow{
		ID:         1003,
		ObjectType: "mail",
		ObjectID:   0,
		Subject:    "提醒：您有 Bug(10)",
		Data:       `<a href="http://zentao.test/bug-view-123.html">123</a>`,
	}
	itemReminder := newNoticeItem(reminderRow, nil)
	if itemReminder.ObjectType != "bug" || itemReminder.ObjectID != 123 {
		t.Fatalf("reminder mail item object = (%s, %d), want (bug, 123)", itemReminder.ObjectType, itemReminder.ObjectID)
	}
	if itemReminder.Category != "reminder" {
		t.Fatalf("reminder mail item category = %s, want reminder", itemReminder.Category)
	}
	if !itemReminder.NeedAction {
		t.Fatalf("reminder mail item needAction = false, want true")
	}
}

func TestClassifyNotice_SubjectAndDataFallback(t *testing.T) {
	cases := []struct {
		objType  string
		action   string
		subject  string
		data     string
		wantCat  string
		wantNeed bool
	}{
		{"demand", "", "DEMAND #10 测试需求", "", "business", false},
		{"charter", "", "CHARTER #529 云销平台", "", "approval", true},
		{"guideline", "", "GUIDELINE #352 建设指引", "", "approval", true},
		{"review", "", "REVIEW #419 评审数据库设计", "", "approval", true},
		{"demand", "", "需求变更审批通知", "", "approval", false},
		{"demand", "", "【催办验收】业务需求 US63224", "", "reminder", true},
		{"story", "", "STORY #123 指派给张三", "", "collaboration", false},
		{"demand", "", "业务需求挂起通知", "", "risk", false},
		{"system", "", "系统通知维护", "", "system", false},
	}

	for _, tc := range cases {
		gotCat := classifyNotice(tc.objType, tc.action, tc.subject, tc.data)
		if gotCat != tc.wantCat {
			t.Errorf("classifyNotice(%q, %q, %q) = %s, want %s", tc.objType, tc.action, tc.subject, gotCat, tc.wantCat)
		}
		gotNeed := noticeNeedsAction(tc.action, tc.subject, tc.data)
		if gotNeed != tc.wantNeed {
			t.Errorf("noticeNeedsAction(%q, %q, %q) = %v, want %v", tc.action, tc.subject, tc.data, gotNeed, tc.wantNeed)
		}
	}
}

func TestApplyNoticeFilters_DemandFilter(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, db)

	mock.ExpectQuery(`SELECT .* FROM zt_notify AS n .*FIND_IN_SET\(\?, REPLACE\(n.toList, ' ', ''\)\) > 0.*COALESCE\(a.objectType, n.objectType\) IN \('demand', 'sub_demand', 'business'\).*ORDER BY n.createdDate DESC, n.id DESC LIMIT \? OFFSET \?`).
		WithArgs("alice", "alice", 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "objectType", "subject"}).AddRow(1, "mail", "DEMAND #10 测试需求"))

	rows, err := repo.queryPagedNoticeRows(context.Background(), "alice", time.Now(), NoticeListReq{ObjectType: "demand"}, 20, 20)
	if err != nil || len(rows) != 1 {
		t.Fatalf("demand query failed: rows=%v, err=%v", rows, err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
