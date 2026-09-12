package po

import (
	"github.com/DATA-DOG/go-sqlmock"
	"strings"
	"testing"
	"time"
	"workbench/internal/config"
	"workbench/internal/pkg/zentao"
)

func TestFeedbackFilterIncludesMailSubjectBeforePagination(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, db)
	mock.ExpectQuery(`SELECT .* FROM zt_notify AS n .*n.subject REGEXP \?.*ORDER BY n.createdDate DESC, n.id DESC LIMIT \? OFFSET \?`).
		WithArgs("alice", "alice", "feedback", `^(反馈|FEEDBACK|Feedback)[[:space:]]*#[[:space:]]*[0-9]+`, 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "objectType", "subject"}).AddRow(42, "mail", "反馈 #2159 标题"))
	rows, err := repo.queryPagedNoticeRows(t.Context(), "alice", time.Now(), NoticeListReq{ObjectType: "feedback"}, 20, 20)
	if err != nil || len(rows) != 1 {
		t.Fatalf("feedback page: %v, %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNoticeMailBugLinksUseActualTargets(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	item := newNoticeItem(noticeRow{ObjectType: "mail", ObjectID: 36698, Subject: "提醒：您有 Bug(13)", Data: `<a href="http://old.test/bug-view-4027.html">4027</a><a href="http://old.test/bug-view-4041.html">4041</a><a href="http://old.test/bug-view-4027.html">重复</a>`}, nil)
	if item.ObjectType != "bug" || item.ObjectID != 4027 || len(item.RelatedObjects) != 2 {
		t.Fatalf("wrong bug links: %+v", item.RelatedObjects)
	}
	for _, link := range item.RelatedObjects {
		if !strings.HasPrefix(link.URL, "http://zentao.test/") {
			t.Fatalf("must use configured origin: %s", link.URL)
		}
	}
	if !strings.Contains(item.URL, "4027") {
		t.Fatalf("wrong primary target: %s", item.URL)
	}
}

func TestDoneHistoryAuxiliaryActionsHaveLabels(t *testing.T) {
	for key, want := range map[string]string{"withdrawreview": "撤回评审", "submitted": "提交评审", "reviewrejected": "评审驳回", "edited": "编辑"} {
		if got := doneHistoryActionLabel("demand", key); got != want {
			t.Errorf("%s: got %s want %s", key, got, want)
		}
	}
}
