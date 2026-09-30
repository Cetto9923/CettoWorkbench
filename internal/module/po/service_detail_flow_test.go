// =============================================================================
// 文件: internal/module/po/service_detail_flow_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证 B3 流程与审批（四块）及管理信息（重要检查项、实际时间）的组装、
//       降级与无数据时不输出（omitempty）逻辑。
// =============================================================================

package po

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// 1. 无流转数据时：FlowApproval 返回 nil，JSON 序列化后无 flowApproval 字段。
func TestPopulateFlowApproval_Empty_YieldsNil(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	// 探针检查表均存在
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandmanagerreview").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demandmanagerreview`).
		WithArgs(uint(300)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandchange").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demandchange`).
		WithArgs(uint(300)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_hanglog").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_hanglog`).
		WithArgs(uint(300)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandreviewrecord").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demandreviewrecord`).
		WithArgs(uint(300)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	flow, err := svc.populateFlowApproval(context.Background(), 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow != nil {
		t.Fatalf("expected nil flowApproval when empty, got %+v", flow)
	}

	resp := &DemandDetailResp{Success: true, FlowApproval: flow}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var rawMap map[string]interface{}
	_ = json.Unmarshal(b, &rawMap)
	if _, exists := rawMap["flowApproval"]; exists {
		t.Fatal("flowApproval key should be omitted when nil")
	}
}

// 2. 有流转数据时：四块及关联子表正确组装并输出中文标签。
func TestPopulateFlowApproval_WithData(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mockFlowApprovalTablesAndData(mock, 301)

	flow, err := svc.populateFlowApproval(context.Background(), 301)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow == nil {
		t.Fatal("expected non-nil FlowApproval")
	}

	verifyManagerReview(t, flow.ManagerReviews)
	verifyDemandChange(t, flow.DemandChanges)
	verifyHangLog(t, flow.HangLogs)
	verifyReviewRecord(t, flow.ReviewRecords)
}

func mockFlowApprovalTablesAndData(mock sqlmock.Sqlmock, demandID uint) {
	mockManagerReviewData(mock, demandID)
	mockDemandChangeData(mock, demandID)
	mockHangLogData(mock, demandID)
	mockReviewRecordData(mock, demandID)
}

func mockManagerReviewData(mock sqlmock.Sqlmock, demandID uint) {
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandmanagerreview").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	revDate := time.Date(2026, 9, 20, 10, 30, 0, 0, time.UTC)
	subDate := time.Date(2026, 9, 19, 14, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demandmanagerreview`).
		WithArgs(demandID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "result", "reviewer", "comment", "submitedBy", "submitedDate", "reviewDate", "resultStatus"}).
			AddRow(10, "pass", "mgr01", "同意上线", "sub01", &subDate, &revDate, `{"mgr01":"pass"}`))

	mock.ExpectQuery(`SELECT.*account.*realname.*FROM.*zt_user.*WHERE.*deleted = '0' AND account IN \(\?,\?\)`).
		WithArgs("mgr01", "sub01").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("mgr01", "李主管").
			AddRow("sub01", "张提交"))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandmanagerreviewdetail").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demandmanagerreviewdetail d[\s\S]*WHERE d\.rid IN \(\?\)`).
		WithArgs(uint(10)).
		WillReturnRows(sqlmock.NewRows([]string{"rid", "product_name", "department", "departmentReviewer"}).
			AddRow(10, "核心系统", "研发一部", "李主管"))
}

func mockDemandChangeData(mock sqlmock.Sqlmock, demandID uint) {
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandchange").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	chgDate := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demandchange`).
		WithArgs(demandID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "demand", "changeBy", "changeDate", "changeType", "reasonType", "changeReason", "desc", "oldDesc", "changeMainDept", "result", "createdDate", "estimateLaunchChange", "developFinishChange", "testFinishChange", "verifyFinishChange"}).
			AddRow(20, demandID, "dev01", &chgDate, "changeDemandContent", "tech", "优化逻辑", "新描述", "旧描述", "科技部", "pass", &chgDate, nil, nil, nil, nil))

	mock.ExpectQuery(`SELECT.*account.*realname.*FROM.*zt_user.*WHERE.*deleted = '0' AND account IN \(\?\)`).
		WithArgs("dev01").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("dev01", "王开发"))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandchangereview").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT changeID, reviewer FROM zt_demandchangereview WHERE changeID IN \(\?\) AND deleted = '0'`).
		WithArgs(uint(20)).
		WillReturnRows(sqlmock.NewRows([]string{"changeID", "reviewer"}).
			AddRow(20, "rev01"))

	mock.ExpectQuery(`SELECT.*account.*realname.*FROM.*zt_user.*WHERE.*deleted = '0' AND account IN \(\?\)`).
		WithArgs("rev01").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("rev01", "赵评审"))
}

func mockHangLogData(mock sqlmock.Sqlmock, demandID uint) {
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_hanglog").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	hangDate := time.Date(2026, 9, 10, 8, 30, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_hanglog`).
		WithArgs(demandID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "account", "action", "hangUpType", "date", "extra", "comment"}).
			AddRow(30, "admin", "hangup", "business", &hangDate, "等待外部接口", "备注说明"))

	mock.ExpectQuery(`SELECT.*account.*realname.*FROM.*zt_user.*WHERE.*deleted = '0' AND account IN \(\?\)`).
		WithArgs("admin").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("admin", "管理员"))
}

func mockReviewRecordData(mock sqlmock.Sqlmock, demandID uint) {
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandreviewrecord").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	rvDate := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demandreviewrecord`).
		WithArgs(demandID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "reviewType", "reviewDate", "reviewResult", "reviewStatus", "createdBy", "createdDate"}).
			AddRow(40, "demand", &rvDate, "评审意见通过", "reviewPassed", "po01", &rvDate))

	mock.ExpectQuery(`SELECT.*account.*realname.*FROM.*zt_user.*WHERE.*deleted = '0' AND account IN \(\?\)`).
		WithArgs("po01").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("po01", "李PO"))
}

// 3. 管理信息全空/默认值：返回 nil，JSON 序列化后无 managementInfo 字段。
func TestPopulateManagementInfo_Empty_YieldsNil(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	// 返回未设置/全空值（对齐 63450 库中实际数据）
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand WHERE id = \? AND deleted = '0'`).
		WithArgs(uint(302)).
		WillReturnRows(sqlmock.NewRows([]string{
			"multiLegalPersonLogo", "isRelatedAccounts", "isImportantOrder", "isNeedReview",
			"onetimeAcceptance", "isCarReview", "verifyDate",
			"clarifyDate", "actualDevStartDate", "actualTestStartDate",
			"submitAcceptanceDate", "acceptancedDate", "reviewedDate", "deliverDate",
		}).AddRow("-1", "-1", "0", "", "", "0", "", nil, nil, nil, nil, nil, nil, nil))

	// 无研发需求关联
	mock.ExpectQuery(`SELECT associatedPublicationID FROM zt_story WHERE fromDemand = \? AND deleted = '0'`).
		WithArgs(uint(302)).
		WillReturnRows(sqlmock.NewRows([]string{"associatedPublicationID"}))

	mgmt, err := svc.populateManagementInfo(context.Background(), 302)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgmt != nil {
		t.Fatalf("expected nil ManagementInfo when all empty, got %+v", mgmt)
	}

	resp := &DemandDetailResp{Success: true, ManagementInfo: mgmt}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var rawMap map[string]interface{}
	_ = json.Unmarshal(b, &rawMap)
	if _, exists := rawMap["managementInfo"]; exists {
		t.Fatal("managementInfo key should be omitted when nil")
	}
}

// 4. 管理信息有数据：7 项检查项与 8 项实际时间均正确解析并输出中文。
func TestPopulateManagementInfo_WithChecksAndTimes(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	d1 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	d4 := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	d5 := time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)
	d6 := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	d7 := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand WHERE id = \? AND deleted = '0'`).
		WithArgs(uint(303)).
		WillReturnRows(sqlmock.NewRows([]string{
			"multiLegalPersonLogo", "isRelatedAccounts", "isImportantOrder", "isNeedReview",
			"onetimeAcceptance", "isCarReview", "verifyDate",
			"clarifyDate", "actualDevStartDate", "actualTestStartDate",
			"submitAcceptanceDate", "acceptancedDate", "reviewedDate", "deliverDate",
		}).AddRow("2", "1", "1", "1", "yes", "1", "2", &d1, &d2, &d3, &d4, &d5, &d6, &d7))

	// 关联发布查询实际发布时间
	mock.ExpectQuery(`SELECT associatedPublicationID FROM zt_story WHERE fromDemand = \? AND deleted = '0'`).
		WithArgs(uint(303)).
		WillReturnRows(sqlmock.NewRows([]string{"associatedPublicationID"}).AddRow("501"))

	relDate := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT date, releaseStatus FROM zt_release WHERE id IN \(\?\)`).
		WithArgs("501").
		WillReturnRows(sqlmock.NewRows([]string{"date", "releaseStatus"}).AddRow(&relDate, "online"))

	mgmt, err := svc.populateManagementInfo(context.Background(), 303)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgmt == nil {
		t.Fatal("expected non-nil ManagementInfo")
	}

	verifyImportantChecks(t, mgmt.ImportantChecks)
	verifyActualTimes(t, mgmt.ActualTimes)
}

// 5. 表不存在探针降级：不报错且平滑返回空切片。
func TestTableFallback_Graceful(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)

	// 探针返回 0（表不存在）
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema\.TABLES`).
		WithArgs("zt_demandchange").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	chgs, err := repo.FindDemandChanges(context.Background(), 999)
	if err != nil {
		t.Fatalf("table missing should not return error, got %v", err)
	}
	if len(chgs) != 0 {
		t.Fatalf("expected empty changes, got %v", chgs)
	}
}

func verifyManagerReview(t *testing.T, reviews []DemandManagerReviewItem) {
	t.Helper()
	if len(reviews) != 1 {
		t.Fatalf("want 1 manager review, got %d", len(reviews))
	}
	mr := reviews[0]
	if mr.ResultLabel != "通过" || mr.ReviewerName != "李主管" || mr.SubmitedByName != "张提交" {
		t.Fatalf("unexpected manager review item: %+v", mr)
	}
	if len(mr.Products) != 1 || mr.Products[0] != "核心系统" {
		t.Fatalf("unexpected products: %+v", mr.Products)
	}
}

func verifyDemandChange(t *testing.T, changes []DemandChangeItem) {
	t.Helper()
	if len(changes) != 1 {
		t.Fatalf("want 1 change, got %d", len(changes))
	}
	chg := changes[0]
	if chg.ChangeTypeLabel != "需求内容变更" {
		t.Fatalf("unexpected change type label: %s", chg.ChangeTypeLabel)
	}
	if chg.ResultLabel != "通过" {
		t.Fatalf("unexpected change result label: %s", chg.ResultLabel)
	}
	if chg.ChangeByName != "王开发" {
		t.Fatalf("unexpected change by name: %s", chg.ChangeByName)
	}
}

func verifyHangLog(t *testing.T, logs []DemandHangLogItem) {
	t.Helper()
	if len(logs) == 0 {
		t.Fatal("expected non-empty hang logs")
	}
	hl := logs[0]
	if hl.ActionLabel != "挂起" {
		t.Fatalf("unexpected action label: %s", hl.ActionLabel)
	}
	if hl.HangUpTypeLabel != "业务原因" {
		t.Fatalf("unexpected hang up type label: %s", hl.HangUpTypeLabel)
	}
	if hl.Comment != "备注说明" || hl.AccountName != "管理员" {
		t.Fatalf("unexpected hang log comment/account: %s, %s", hl.Comment, hl.AccountName)
	}
}

func verifyReviewRecord(t *testing.T, records []DemandReviewRecordItem) {
	t.Helper()
	if len(records) < 1 {
		t.Fatal("expected at least 1 review record")
	}
	rr := records[0]
	if rr.ReviewTypeLabel != "需求评审" {
		t.Fatalf("unexpected review type label: %s", rr.ReviewTypeLabel)
	}
	if rr.ReviewStatusLabel != "评审通过" {
		t.Fatalf("unexpected review status label: %s", rr.ReviewStatusLabel)
	}
	if rr.CreatedByName != "李PO" || rr.ReviewDate != "2026-09-05" {
		t.Fatalf("unexpected review record: %+v", rr)
	}
}

func verifyImportantChecks(t *testing.T, c *DemandImportantChecks) {
	t.Helper()
	if c == nil {
		t.Fatal("expected non-nil ImportantChecks")
	}
	if c.MultiLegalPersonLogoLabel != "常熟&村镇" {
		t.Fatalf("want '常熟&村镇', got %q", c.MultiLegalPersonLogoLabel)
	}
	if c.IsRelatedAccountsLabel != "是" || c.IsImportantOrderLabel != "是" || c.IsNeedReviewLabel != "是" {
		t.Fatalf("unexpected yes/no check labels: %+v", c)
	}
	if c.OnetimeAcceptanceLabel != "是" || c.IsCarReviewLabel != "是" {
		t.Fatalf("unexpected onetime/car labels: %+v", c)
	}
	if c.VerifyDateLabel != "次日验证" {
		t.Fatalf("want '次日验证', got %q", c.VerifyDateLabel)
	}
}

func verifyActualTimes(t *testing.T, tm *DemandActualTimes) {
	t.Helper()
	if tm == nil {
		t.Fatal("expected non-nil ActualTimes")
	}
	if tm.ClarifyDate != "2026-09-01" {
		t.Fatalf("want 2026-09-01, got %q", tm.ClarifyDate)
	}
	if tm.FirstToStoryDate != "2026-09-05" || tm.ActualDevStartDate != "2026-09-05" {
		t.Fatalf("want 2026-09-05, got %q", tm.FirstToStoryDate)
	}
	if tm.ActualTestStartDate != "2026-09-15" || tm.ActualDevCompletionDate != "2026-09-15" {
		t.Fatalf("want 2026-09-15, got %q", tm.ActualTestStartDate)
	}
	if tm.SubmitAcceptanceDate != "2026-09-20" || tm.ActualTestCompletionDate != "2026-09-20" {
		t.Fatalf("want 2026-09-20, got %q", tm.SubmitAcceptanceDate)
	}
	if tm.AcceptancedDate != "2026-09-22" {
		t.Fatalf("want 2026-09-22, got %q", tm.AcceptancedDate)
	}
	if tm.ReviewedDate != "2026-08-28" {
		t.Fatalf("want 2026-08-28, got %q", tm.ReviewedDate)
	}
	if tm.DeliverDate != "2026-09-25" {
		t.Fatalf("want 2026-09-25, got %q", tm.DeliverDate)
	}
	if tm.RealReleaseDate != "2026-09-28" {
		t.Fatalf("want 2026-09-28, got %q", tm.RealReleaseDate)
	}
}
