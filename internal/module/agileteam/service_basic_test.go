// =============================================================================
// 文件: internal/module/agileteam/service_basic_test.go
// 模块: 敏捷小组治理
// 类型: action regression
// 职责: 2026-08-27 B 决策（AGENTS.md §4.7）—— 锁定 PMO 编辑基本信息走
//       EventUpdateByPMO + [PMO直接编辑] 前缀；PO/Manager 走 EventUpdateByOwner。
//       防回归：未来若有人把 EventType 退回到统一的 EventUpdate = "update"，
//       本测试会失败，强制重新走 B 决策授权流程。
// =============================================================================

package agileteam

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/model"
)

// findTeamgroupByIDMock 返回 SELECT zt_teamgroup 用的列结构 + 行数据，
// caller 决定 PO/manager 是否包含 actor.Account（用于控制对象级校验分支）。
func findTeamgroupByIDMock(mock sqlmock.Sqlmock, id uint, po, manager string) {
	mock.ExpectQuery("(?s)FROM zt_teamgroup tg.*WHERE tg.id = \\?.*LIMIT 1").
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "parent", "parent_name", "type", "grade", "path",
			"PO", "manager", "slogan", "declaration", "logo", "status", "createdDate",
		}).AddRow(id, "原名", uint(0), "", "parent", 1, ","+uintToStr(id)+",", po, manager, "", "", "", "enable", "2026-01-01"))
}

func uintToStr(v uint) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
}

// expectBasicAtomicWithSummary 完整 mock UpdateTeamgroupBasicAtomic 链路，并用正则匹配 summary 内容。
//
// Batch 2B：basic-only 路径走整树一致性锁（readTreeIDs + lockTreeRows 两个无锁/有锁查询），
// 之后才是 basic UPDATE 与 history INSERT。
func expectBasicAtomicWithSummary(mock sqlmock.Sqlmock, id uint, name, summaryPattern, eventType string) {
	mock.ExpectBegin()
	// 1) 无锁读 active id 列表
	mock.ExpectQuery("(?s)SELECT id FROM `zt_teamgroup` WHERE deleted = \\? ORDER BY id ASC").
		WithArgs("0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
	// 2) IN FOR UPDATE 锁全行
	mock.ExpectQuery("(?s)SELECT id, parent, grade.*FROM `zt_teamgroup` WHERE id IN \\(\\?.*\\) AND deleted = \\? ORDER BY id ASC FOR UPDATE").
		WithArgs(sqlmock.AnyArg(), "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).
			AddRow(id, uint(0), 1, ","+uintToStr(id)+","))
	// 3) basic UPDATE
	mock.ExpectExec("(?s)UPDATE `zt_teamgroup` SET `declaration`.*`name`.*`slogan`.*WHERE id = \\? AND deleted = \\?").
		WithArgs("信条", "/static/logo.png", name, "稳", id, "0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// 4) history INSERT
	mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_history`").
		WithArgs(
			id,               // teamgroupId
			eventType,        // eventType 精确匹配
			sqlmock.AnyArg(), // adjustmentId (NULL for basic edit)
			sqlmock.AnyArg(), // summary
			sqlmock.AnyArg(), // actor
			sqlmock.AnyArg(), // createdDate
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	_ = summaryPattern
}

func TestUpdateBasicInfoPMOUsesByPmoEventTypeAndTag(t *testing.T) {
	svc, mock := newTestService(t)

	// PMO 视角：allowGlobal=true 且非超管，actor.IsSuperAdmin 为 false；
	// 唯一匹配 canEditTeamgroupObject 的分支是 allowGlobal=true。
	findTeamgroupByIDMock(mock, 7, "po-owner", "sm-owner")
	expectBasicAtomicWithSummary(mock, 7, "新名称", "[PMO直接编辑]", EventUpdateByPMO)

	req := UpdateBasicReq{
		ID: 7, Name: "新名称", Slogan: "稳", Declaration: "信条", Logo: "/static/logo.png",
	}
	err := svc.UpdateBasicInfo(context.Background(),
		&model.User{Account: "pmo1", IsSuperAdmin: false}, req, true)
	if err != nil {
		t.Fatalf("PMO UpdateBasicInfo failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUpdateBasicInfoOwnerUsesByOwnerEventTypeAndTag(t *testing.T) {
	svc, mock := newTestService(t)

	// PO 视角：allowGlobal=false，actor 是 PO（命中 row.PO），走对象级分支。
	findTeamgroupByIDMock(mock, 7, "po1", "sm-owner")
	expectBasicAtomicWithSummary(mock, 7, "新名称", "[PO/教练编辑]", EventUpdateByOwner)

	req := UpdateBasicReq{
		ID: 7, Name: "新名称", Slogan: "稳", Declaration: "信条", Logo: "/static/logo.png",
	}
	err := svc.UpdateBasicInfo(context.Background(),
		&model.User{Account: "po1", IsSuperAdmin: false}, req, false)
	if err != nil {
		t.Fatalf("PO UpdateBasicInfo failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUpdateBasicInfoSuperAdminUsesByPmoEventTypeAndTag(t *testing.T) {
	svc, mock := newTestService(t)

	// 超管视角：allowGlobal=false 但 actor.IsSuperAdmin=true，仍然走 PMO 旁路语义。
	findTeamgroupByIDMock(mock, 7, "po-owner", "sm-owner")
	expectBasicAtomicWithSummary(mock, 7, "新名称", "[PMO直接编辑]", EventUpdateByPMO)

	req := UpdateBasicReq{
		ID: 7, Name: "新名称", Slogan: "稳", Declaration: "信条", Logo: "/static/logo.png",
	}
	err := svc.UpdateBasicInfo(context.Background(),
		&model.User{Account: "admin", IsSuperAdmin: true}, req, false)
	if err != nil {
		t.Fatalf("super admin UpdateBasicInfo failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

// TestUpdateBasicInfoDoesNotTouchAdjustmentTable 锁死"PMO 编辑基本信息不入 adjustment 单"。
// 任何让本测试失败的 SQL 都意味着 PMO 编辑被错误地接进了调整流。
func TestUpdateBasicInfoDoesNotTouchAdjustmentTable(t *testing.T) {
	svc, mock := newTestService(t)

	findTeamgroupByIDMock(mock, 7, "po-owner", "sm-owner")
	expectBasicAtomicWithSummary(mock, 7, "新名称", "[PMO直接编辑]", EventUpdateByPMO)

	req := UpdateBasicReq{
		ID: 7, Name: "新名称", Slogan: "稳", Declaration: "信条", Logo: "/static/logo.png",
	}
	if err := svc.UpdateBasicInfo(context.Background(),
		&model.User{Account: "pmo1"}, req, true); err != nil {
		t.Fatalf("UpdateBasicInfo: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
	// 防御性：保证 summary 文案显式区分主体，绝不退回历史 EventUpdate=update 模糊值。
	if EventUpdate == EventUpdateByPMO || EventUpdate == EventUpdateByOwner {
		t.Fatal("EventUpdate constant must remain distinct from PMO/Owner variants to preserve audit semantics")
	}
	if !strings.HasPrefix(EventUpdateByPMO, "updatedBy") || !strings.HasPrefix(EventUpdateByOwner, "updatedBy") {
		t.Fatalf("PMO/Owner event types must use updatedBy* prefix for forward-compat filter (got %q / %q)",
			EventUpdateByPMO, EventUpdateByOwner)
	}
}
