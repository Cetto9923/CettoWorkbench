// =============================================================================
// 文件: internal/module/po/reponotice_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 验证通知分类和筛选请求的核心口径。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"workbench/internal/config"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

func TestClassifyNotice(t *testing.T) {
	cases := []struct{ objectType, action, want string }{
		{"demand", "reviewed", "approval"},
		{"task", "reminded", "reminder"},
		{"story", "assigned", "collaboration"},
		{"bug", "bugconfirmed", "risk"},
		{"system", "", "system"},
		{"demand", "activated", "business"},
	}
	for _, testCase := range cases {
		if actual := classifyNotice(testCase.objectType, testCase.action); actual != testCase.want {
			t.Fatalf("classifyNotice(%q, %q) = %q, want %q", testCase.objectType, testCase.action, actual, testCase.want)
		}
	}
}

func TestNoticeListReqValidate(t *testing.T) {
	req := NoticeListReq{}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("default request errors: %#v", errs)
	}
	if req.Category != "all" || req.Page != 1 || req.PageSize != 20 {
		t.Fatalf("defaults = %#v", req)
	}
	invalid := NoticeListReq{Category: "guessed"}
	if errs := invalid.Validate(); len(errs) != 1 || errs[0].Field != "category" {
		t.Fatalf("invalid category errors: %#v", errs)
	}
}

func TestFilterNoticeRowsTreatsApprovalAsAnObjectFilter(t *testing.T) {
	rows := []noticeRow{{ObjectType: "demand", ActionCode: "reviewed"}, {ObjectType: "demand", ActionCode: "activated"}}
	filtered := filterNoticeRows(rows, NoticeListReq{ObjectType: "approval"})
	if len(filtered) != 1 || filtered[0].ActionCode != "reviewed" {
		t.Fatalf("approval filter = %#v", filtered)
	}
}

func TestCleanNoticeText(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		maxRunes int
		want     string
	}{
		{
			name:     "strip script and style along with content",
			input:    "<style>body { color: red; }</style><script>alert('xss')</script>你好世界",
			maxRunes: 200,
			want:     "你好世界",
		},
		{
			name:     "strip entity-encoded script and style along with content",
			input:    "&lt;script&gt;alert(1)&lt;/script&gt;需求已更新",
			maxRunes: 200,
			want:     "需求已更新",
		},
		{
			name:     "decode html entities once",
			input:    "&quot;需求&quot; &amp; &lt;测试&gt;",
			maxRunes: 200,
			want:     "\"需求\" & <测试>",
		},
		{
			name:     "double-escaped entities decoded once",
			input:    "&amp;lt;script&amp;gt;",
			maxRunes: 200,
			want:     "&lt;script&gt;",
		},
		{
			name:     "malicious tags stripped",
			input:    "<img src=x onerror=alert(1)>任务已更新",
			maxRunes: 200,
			want:     "任务已更新",
		},
		{
			name:     "empty and whitespace normalized",
			input:    "   <p>   \n\t  </p>  ",
			maxRunes: 200,
			want:     "",
		},
		{
			name:     "rune-based truncation",
			input:    "这是关于PO工作台的中文通知说明文本",
			maxRunes: 10,
			want:     "这是关于PO工作台的…",
		},
		{
			name:     "html comments stripped",
			input:    "<!-- 注释内容 -->正式通知正文",
			maxRunes: 200,
			want:     "正式通知正文",
		},
		{
			name:     "css residue stripped",
			input:    "body { color: red; margin: 0; } 通知正文",
			maxRunes: 200,
			want:     "通知正文",
		},
		{
			name:     "css block only stripped to empty",
			input:    "{ color: red; }",
			maxRunes: 200,
			want:     "",
		},
		{
			name:     "chinese tags preserved",
			input:    "更新 <需求名称> 正文",
			maxRunes: 200,
			want:     "更新 <需求名称> 正文",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanNoticeText(tc.input, tc.maxRunes)
			if got != tc.want {
				t.Fatalf("cleanNoticeText(%q, %d) = %q, want %q", tc.input, tc.maxRunes, got, tc.want)
			}
		})
	}
}

func TestNewNoticeItemDeduplicatesIdenticalSubjectAndData(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	row := noticeRow{
		ID:         1,
		ObjectType: "demand",
		ObjectID:   101,
		Subject:    "<b>需求评审通知</b>",
		Data:       "<p>需求评审通知</p>",
		ActionCode: "reviewed",
		CreatedBy:  "alice",
	}
	item := newNoticeItem(row, nil)
	if item.Subject != "需求评审通知" {
		t.Fatalf("item.Subject = %q, want %q", item.Subject, "需求评审通知")
	}
	if item.Data != "" {
		t.Fatalf("item.Data should be empty when matching subject, got %q", item.Data)
	}
	if item.URL == "" {
		t.Fatal("item.URL should not be empty for positive ObjectID")
	}
}

func TestNewNoticeItemZeroObjectIDHasEmptyURL(t *testing.T) {
	row := noticeRow{
		ID:         2,
		ObjectType: "demand",
		ObjectID:   0,
		Subject:    "系统全员广播",
		Data:       "详细内容",
		ActionCode: "submit",
		CreatedBy:  "system",
	}
	item := newNoticeItem(row, nil)
	if item.URL != "" {
		t.Fatalf("item.URL should be empty when ObjectID <= 0, got %q", item.URL)
	}
}

func TestCheckNoticeAccess(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, db)
	ctx := context.Background()

	// 1. Notice does not exist
	mock.ExpectQuery(`SELECT id, toList FROM `+"`zt_notify`"+` WHERE id = \?`).
		WithArgs(999, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	exists, auth, err := repo.CheckNoticeAccess(ctx, "alice", 999)
	if err != nil {
		t.Fatalf("CheckNoticeAccess unexpected error: %v", err)
	}
	if exists || auth {
		t.Fatalf("expected exists=false, auth=false for nonexistent notice, got exists=%v, auth=%v", exists, auth)
	}

	// 2. Notice exists but alice is not in toList
	mock.ExpectQuery(`SELECT id, toList FROM `+"`zt_notify`"+` WHERE id = \?`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "toList"}).AddRow(1, "bob,charlie"))

	exists, auth, err = repo.CheckNoticeAccess(ctx, "alice", 1)
	if err != nil {
		t.Fatalf("CheckNoticeAccess unexpected error: %v", err)
	}
	if !exists || auth {
		t.Fatalf("expected exists=true, auth=false for unauthorized notice, got exists=%v, auth=%v", exists, auth)
	}

	// 3. Notice exists and alice is in toList
	mock.ExpectQuery(`SELECT id, toList FROM `+"`zt_notify`"+` WHERE id = \?`).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "toList"}).AddRow(2, "bob, alice ,charlie"))

	exists, auth, err = repo.CheckNoticeAccess(ctx, "alice", 2)
	if err != nil {
		t.Fatalf("CheckNoticeAccess unexpected error: %v", err)
	}
	if !exists || !auth {
		t.Fatalf("expected exists=true, auth=true for authorized notice, got exists=%v, auth=%v", exists, auth)
	}
}

func TestMarkNoticeRead_ObjectAuth(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewRepo(db, db)
	svc := NewService(repo, nil, nil, nil)
	ctx := context.Background()
	actor := &model.User{Account: "alice"}

	// 1. Nonexistent notice -> 404
	mock.ExpectQuery(`SELECT id, toList FROM `+"`zt_notify`"+` WHERE id = \?`).
		WithArgs(999, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	err := svc.NoticeMarkRead(ctx, actor, 999)
	if err == nil {
		t.Fatal("expected error for nonexistent notice, got nil")
	}
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeNotFound {
		t.Fatalf("expected ErrCodeNotFound (404), got: %v", err)
	}

	// 2. Notice belongs to another user -> 403
	mock.ExpectQuery(`SELECT id, toList FROM `+"`zt_notify`"+` WHERE id = \?`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "toList"}).AddRow(1, "bob"))

	err = svc.NoticeMarkRead(ctx, actor, 1)
	if err == nil {
		t.Fatal("expected error for unauthorized notice, got nil")
	}
	bizErr, ok = errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden (403), got: %v", err)
	}

	// 3. Authorized notice -> SaveNoticeRead called
	mock.ExpectQuery(`SELECT id, toList FROM `+"`zt_notify`"+` WHERE id = \?`).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "toList"}).AddRow(2, "alice"))
	mock.ExpectExec(`(?s)INSERT INTO zt_workbench_notify_reads`).
		WithArgs("alice", "alice", int64(2), "alice").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = svc.NoticeMarkRead(ctx, actor, 2)
	if err != nil {
		t.Fatalf("unexpected error for authorized notice: %v", err)
	}
}

func TestNewNoticeItem_DTOFieldsPopulated(t *testing.T) {
	row := noticeRow{
		ID:         10,
		ObjectType: "demand",
		ObjectID:   105,
		Subject:    "&quot;新需求评审&quot;",
		Data:       "请各负责人准时参会",
		ActionCode: "reviewed",
		CreatedBy:  "alice",
	}
	item := newNoticeItem(row, nil)
	if item.Title != "\"新需求评审\"" {
		t.Fatalf("item.Title = %q, want %q", item.Title, "\"新需求评审\"")
	}
	if item.Summary != "请各负责人准时参会" {
		t.Fatalf("item.Summary = %q, want %q", item.Summary, "请各负责人准时参会")
	}
	if item.Content != "请各负责人准时参会" {
		t.Fatalf("item.Content = %q, want %q", item.Content, "请各负责人准时参会")
	}
	if item.Subject != item.Title {
		t.Fatalf("item.Subject should match Title, got %q", item.Subject)
	}
	if item.Data != item.Content {
		t.Fatalf("item.Data should match Content, got %q", item.Data)
	}
}
