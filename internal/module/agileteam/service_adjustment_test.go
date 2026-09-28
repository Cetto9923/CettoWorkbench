// =============================================================================
// 文件: internal/module/agileteam/service_adjustment_test.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 确认才写 zt_team；驳回不改正式成员；状态流转保持原子性。
// =============================================================================

package agileteam

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func TestConfirmAdjustmentWritesTeamAtomically(t *testing.T) {
	for _, hours := range []float64{0, 7} {
		t.Run(fmt.Sprint(hours), func(t *testing.T) {
			svc, mock := newTestService(t)
			now := time.Now()
			mock.ExpectQuery("(?s)SELECT \\* FROM `zt_wb_agileteam_adjustment`.*id = \\?.*LIMIT").
				WithArgs(int64(10), 1).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "teamgroupId", "adjustNo", "status", "reason",
					"submittedBy", "confirmedBy", "confirmedDate", "rejectedBy", "rejectedDate", "rejectReason",
					"createdBy", "createdDate", "updatedBy", "updatedDate", "deletedAt",
				}).AddRow(int64(10), uint(3), "ADJ-010", StatusPending, "扩编",
					"po1", "", nil, "", nil, "",
					"po1", now, "po1", now, nil))
			mock.ExpectQuery("(?s)SELECT \\* FROM `zt_wb_agileteam_adjustment_item`.*adjustmentId").
				WithArgs(int64(10)).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "adjustmentId", "account", "actionType", "role", "prevRole",
					"availableHours", "prevHours", "createdBy", "createdDate", "updatedBy", "updatedDate", "deletedAt",
				}).AddRow(int64(1), int64(10), "newbie", ActionAdd, "研发", "", hours, 0.0, "po1", now, "po1", now, nil))
			mock.ExpectQuery("(?s)SELECT account FROM `zt_user`.*account IN.*deleted = '0'").
				WithArgs("newbie").
				WillReturnRows(sqlmock.NewRows([]string{"account"}).AddRow("newbie"))

			mock.ExpectBegin()
			mock.ExpectExec("(?s)UPDATE `zt_wb_agileteam_adjustment`").
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery("(?s)SELECT count\\(\\*\\) FROM `zt_team`").
				WithArgs(uint(3), "newbie").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			mock.ExpectExec("(?s)INSERT INTO zt_team").
				WithArgs(uint(3), "newbie", "研发", sqlmock.AnyArg(), hours).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_history`").
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			actor := &model.User{Account: "admin", IsSuperAdmin: true}
			if err := svc.ConfirmAdjustment(context.Background(), actor, ConfirmReq{AdjustmentID: 10}, true); err != nil {
				t.Fatalf("confirm: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sql expectations: %v", err)
			}
		})
	}
}

func TestConfirmAdjustmentConcurrentTransitionRollsBack(t *testing.T) {
	svc, mock := newTestService(t)
	now := time.Now()
	mock.ExpectQuery("(?s)SELECT \\* FROM `zt_wb_agileteam_adjustment`.*id = \\?.*LIMIT").
		WithArgs(int64(12), 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "teamgroupId", "adjustNo", "status", "reason",
			"submittedBy", "confirmedBy", "confirmedDate", "rejectedBy", "rejectedDate", "rejectReason",
			"createdBy", "createdDate", "updatedBy", "updatedDate", "deletedAt",
		}).AddRow(int64(12), uint(3), "ADJ-012", StatusPending, "扩编",
			"po1", "", nil, "", nil, "",
			"po1", now, "po1", now, nil))
	mock.ExpectQuery("(?s)SELECT \\* FROM `zt_wb_agileteam_adjustment_item`.*adjustmentId").
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "adjustmentId", "account", "actionType", "role", "prevRole",
			"availableHours", "prevHours", "createdBy", "createdDate", "updatedBy", "updatedDate", "deletedAt",
		}).AddRow(int64(2), int64(12), "newbie", ActionAdd, "研发", "", 7.0, 0.0, "po1", now, "po1", now, nil))
	mock.ExpectQuery("(?s)SELECT account FROM `zt_user`.*account IN.*deleted = '0'").
		WithArgs("newbie").
		WillReturnRows(sqlmock.NewRows([]string{"account"}).AddRow("newbie"))
	mock.ExpectBegin()
	mock.ExpectExec("(?s)UPDATE `zt_wb_agileteam_adjustment`").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err := svc.ConfirmAdjustment(context.Background(), &model.User{Account: "pmo1"}, ConfirmReq{AdjustmentID: 12}, true)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != "conflict" {
		t.Fatalf("want conflict after concurrent transition, got %#v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestConfirmAdjustmentRejectsInvalidHours(t *testing.T) {
	for _, hours := range []float64{-1, 25} {
		t.Run(fmt.Sprint(hours), func(t *testing.T) {
			svc, mock := newTestService(t)
			now := time.Now()
			mock.ExpectQuery("(?s)SELECT \\* FROM `zt_wb_agileteam_adjustment`.*id = \\?.*LIMIT").
				WithArgs(int64(20), 1).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "teamgroupId", "adjustNo", "status", "reason",
					"submittedBy", "confirmedBy", "confirmedDate", "rejectedBy", "rejectedDate", "rejectReason",
					"createdBy", "createdDate", "updatedBy", "updatedDate", "deletedAt",
				}).AddRow(int64(20), uint(3), "ADJ-020", StatusPending, "扩编",
					"po1", "", nil, "", nil, "",
					"po1", now, "po1", now, nil))
			mock.ExpectQuery("(?s)SELECT \\* FROM `zt_wb_agileteam_adjustment_item`.*adjustmentId").
				WithArgs(int64(20)).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "adjustmentId", "account", "actionType", "role", "prevRole",
					"availableHours", "prevHours", "createdBy", "createdDate", "updatedBy", "updatedDate", "deletedAt",
				}).AddRow(int64(1), int64(20), "newbie", ActionAdd, "研发", "", hours, 0.0, "po1", now, "po1", now, nil))

			actor := &model.User{Account: "admin", IsSuperAdmin: true}
			err := svc.ConfirmAdjustment(context.Background(), actor, ConfirmReq{AdjustmentID: 20}, true)
			biz, ok := errorx.IsBizError(err)
			if !ok || biz.Code != "invalid" {
				t.Fatalf("want invalid for hours=%v, got %#v", hours, err)
			}
			if !strings.Contains(err.Error(), "必须在 0 到 24 之间") {
				t.Fatalf("want unified hours message, got %q", err.Error())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sql expectations: %v", err)
			}
		})
	}
}

func TestRejectAdjustmentDoesNotTouchTeam(t *testing.T) {
	svc, mock := newTestService(t)
	now := time.Now()
	mock.ExpectQuery("(?s)SELECT \\* FROM `zt_wb_agileteam_adjustment`.*id = \\?.*LIMIT").
		WithArgs(int64(11), 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "teamgroupId", "adjustNo", "status", "reason",
			"submittedBy", "confirmedBy", "confirmedDate", "rejectedBy", "rejectedDate", "rejectReason",
			"createdBy", "createdDate", "updatedBy", "updatedDate", "deletedAt",
		}).AddRow(int64(11), uint(3), "ADJ-011", StatusPending, "扩编",
			"po1", "", nil, "", nil, "",
			"po1", now, "po1", now, nil))
	mock.ExpectBegin()
	mock.ExpectExec("(?s)UPDATE `zt_wb_agileteam_adjustment`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("(?s)INSERT INTO `zt_wb_agileteam_history`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	actor := &model.User{Account: "pmo1"}
	if err := svc.RejectAdjustment(context.Background(), actor, RejectReq{AdjustmentID: 11, Reason: "暂缓"}, true); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
