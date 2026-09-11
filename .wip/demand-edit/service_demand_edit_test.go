// =============================================================================
// 文件: internal/module/po/service_demand_edit_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 测试业务需求草稿/驳回状态编辑与删除服务的权限鉴权、状态机边界与软删除。
// =============================================================================

package po

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func newServiceForDemandEditTest(t *testing.T) (*Service, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	repo := NewRepo(gdb)
	repo.writeDB = gdb
	s := NewService(repo, nil, nil, nil)
	return s, mock
}

func TestGetDemandEditData_Unauthenticated(t *testing.T) {
	s, _ := newServiceForDemandEditTest(t)
	_, err := s.GetDemandEditData(context.Background(), nil, 63421)
	if err == nil {
		t.Fatal("expected error for unauthenticated actor, got nil")
	}
	if biz, ok := errorx.IsBizError(err); !ok || biz.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got %v", err)
	}
}

func TestGetDemandEditData_NonCreatorForbidden(t *testing.T) {
	s, mock := newServiceForDemandEditTest(t)
	actor := &model.User{Account: "other_user"}

	// 查询需求返回创建人为 user_creator
	rows := sqlmock.NewRows([]string{
		"id", "name", "status", "deleted", "pri", "category", "source", "sourceNote",
		"pool", "pool_name", "product", "product_name", "reviewer", "reviewer_name",
		"estimateLaunch", "desc", "verifyPlan", "createdBy", "created_by_name",
	}).AddRow(
		63421, "测试草稿需求", "draft", "0", "3", "feature", "业务提出", "",
		1, "核心需求池", "1", "核心系统", "rev_user", "评审人",
		nil, "需求描述", "验收标准", "user_creator", "创建人",
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT\n  d.id, d.name, d.status, d.deleted")).
		WithArgs(int64(63421)).
		WillReturnRows(rows)

	_, err := s.GetDemandEditData(context.Background(), actor, 63421)
	if err == nil {
		t.Fatal("expected forbidden error for non-creator, got nil")
	}
	if biz, ok := errorx.IsBizError(err); !ok || biz.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got %v", err)
	}
}

func TestGetDemandEditData_WrongStatusConflict(t *testing.T) {
	s, mock := newServiceForDemandEditTest(t)
	actor := &model.User{Account: "user_creator"}

	// 需求处于 wait（审核中）状态
	rows := sqlmock.NewRows([]string{
		"id", "name", "status", "deleted", "pri", "category", "source", "sourceNote",
		"pool", "pool_name", "product", "product_name", "reviewer", "reviewer_name",
		"estimateLaunch", "desc", "verifyPlan", "createdBy", "created_by_name",
	}).AddRow(
		63421, "审核中需求", "wait", "0", "3", "feature", "业务提出", "",
		1, "核心需求池", "1", "核心系统", "rev_user", "评审人",
		nil, "需求描述", "验收标准", "user_creator", "创建人",
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT\n  d.id, d.name, d.status, d.deleted")).
		WithArgs(int64(63421)).
		WillReturnRows(rows)

	_, err := s.GetDemandEditData(context.Background(), actor, 63421)
	if err == nil {
		t.Fatal("expected conflict error for status=wait, got nil")
	}
	if biz, ok := errorx.IsBizError(err); !ok || biz.Code != errorx.ErrCodeConflict {
		t.Fatalf("expected ErrCodeConflict, got %v", err)
	}
}

func TestGetDemandEditData_CreatorSuccess(t *testing.T) {
	s, mock := newServiceForDemandEditTest(t)
	actor := &model.User{Account: "user_creator"}

	rows := sqlmock.NewRows([]string{
		"id", "name", "status", "deleted", "pri", "category", "source", "sourceNote",
		"pool", "pool_name", "product", "product_name", "reviewer", "reviewer_name",
		"estimateLaunch", "desc", "verifyPlan", "createdBy", "created_by_name",
	}).AddRow(
		63421, "草稿需求", "draft", "0", "3", "feature", "业务提出", "",
		1, "核心需求池", "1", "核心系统", "rev_user", "评审人",
		nil, "需求描述", "验收标准", "user_creator", "创建人",
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT\n  d.id, d.name, d.status, d.deleted")).
		WithArgs(int64(63421)).
		WillReturnRows(rows)

	// mock options
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name FROM zt_demandpool")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("1", "需求池1"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name FROM zt_product")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("1", "产品1"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT account AS value, CONCAT(realname")).
		WillReturnRows(sqlmock.NewRows([]string{"value", "label"}).AddRow("rev_user", "评审人 (rev_user)"))

	resp, err := s.GetDemandEditData(context.Background(), actor, 63421)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if resp == nil || resp.Demand.Name != "草稿需求" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	if !resp.CanEdit || !resp.CanDelete {
		t.Fatalf("expected CanEdit and CanDelete to be true")
	}
}

func TestDeleteDemand_NonCreatorForbidden(t *testing.T) {
	s, mock := newServiceForDemandEditTest(t)
	actor := &model.User{Account: "stranger"}

	rows := sqlmock.NewRows([]string{
		"id", "name", "status", "deleted", "pri", "category", "source", "sourceNote",
		"pool", "pool_name", "product", "product_name", "reviewer", "reviewer_name",
		"estimateLaunch", "desc", "verifyPlan", "createdBy", "created_by_name",
	}).AddRow(
		63421, "草稿需求", "draft", "0", "3", "feature", "业务提出", "",
		1, "核心需求池", "1", "核心系统", "rev_user", "评审人",
		nil, "需求描述", "验收标准", "user_creator", "创建人",
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT\n  d.id, d.name, d.status, d.deleted")).
		WithArgs(int64(63421)).
		WillReturnRows(rows)

	err := s.DeleteDemand(context.Background(), actor, DeleteDemandReq{ID: 63421})
	if err == nil {
		t.Fatal("expected forbidden error for non-creator delete, got nil")
	}
	if biz, ok := errorx.IsBizError(err); !ok || biz.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got %v", err)
	}
}

func TestDeleteDemand_ActiveStatusConflict(t *testing.T) {
	s, mock := newServiceForDemandEditTest(t)
	actor := &model.User{Account: "user_creator"}

	rows := sqlmock.NewRows([]string{
		"id", "name", "status", "deleted", "pri", "category", "source", "sourceNote",
		"pool", "pool_name", "product", "product_name", "reviewer", "reviewer_name",
		"estimateLaunch", "desc", "verifyPlan", "createdBy", "created_by_name",
	}).AddRow(
		63421, "已通过需求", "active", "0", "3", "feature", "业务提出", "",
		1, "核心需求池", "1", "核心系统", "rev_user", "评审人",
		nil, "需求描述", "验收标准", "user_creator", "创建人",
	)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT\n  d.id, d.name, d.status, d.deleted")).
		WithArgs(int64(63421)).
		WillReturnRows(rows)

	err := s.DeleteDemand(context.Background(), actor, DeleteDemandReq{ID: 63421})
	if err == nil {
		t.Fatal("expected conflict error for status=active delete, got nil")
	}
	if biz, ok := errorx.IsBizError(err); !ok || biz.Code != errorx.ErrCodeConflict {
		t.Fatalf("expected ErrCodeConflict, got %v", err)
	}
}

func TestDeleteDemand_CreatorSuccess(t *testing.T) {
	s, mock := newServiceForDemandEditTest(t)
	actor := &model.User{Account: "user_creator"}

	// 1. FindDemandForEdit 预检
	mock.ExpectQuery(regexp.QuoteMeta("SELECT\n  d.id, d.name, d.status, d.deleted")).
		WithArgs(int64(63421)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "deleted", "pri", "category", "source", "sourceNote",
			"pool", "pool_name", "product", "product_name", "reviewer", "reviewer_name",
			"estimateLaunch", "desc", "verifyPlan", "createdBy", "created_by_name",
		}).AddRow(
			63421, "草稿需求", "draft", "0", "3", "feature", "业务提出", "",
			1, "核心需求池", "1", "核心系统", "rev_user", "评审人",
			nil, "需求描述", "验收标准", "user_creator", "创建人",
		))

	// 2. 事务锁行并软删除
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, status, deleted, createdBy, product FROM `zt_demand` WHERE id = ? LIMIT ? FOR UPDATE")).
		WithArgs(int64(63421), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "deleted", "createdBy", "product"}).
			AddRow(63421, "draft", "0", "user_creator", "1"))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE `zt_demand` SET")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE `zt_story` SET `fromDemand`=? WHERE fromDemand = ? AND deleted = '0'")).
		WithArgs(0, int64(63421)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `zt_action`")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err := s.DeleteDemand(context.Background(), actor, DeleteDemandReq{ID: 63421, Comment: "测试删除"})
	if err != nil {
		t.Fatalf("expected successful delete, got %v", err)
	}
}
