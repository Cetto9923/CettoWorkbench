package agileteam

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"

	"workbench/internal/model"
)

func TestAdjustmentHoursBoundary(t *testing.T) {
	for _, action := range []string{ActionAdd, ActionRoleChange} {
		for _, hours := range []float64{-1, 0, 7, 24, 25} {
			req := SubmitAdjustmentReq{TeamgroupID: 3, Items: []AdjustItemReq{{Account: "dev1", ActionType: action, Role: "研发", AvailableHours: hours}}}
			err := validateAdjustmentWriteBoundary(req)
			if (err == nil) != (hours >= 0 && hours <= 24) {
				t.Fatalf("action=%s hours=%v error=%v", action, hours, err)
			}
		}
	}
}

func TestSubmitZeroHoursReachesTeamLookup(t *testing.T) {
	svc, mock := newTestService(t)
	lookupErr := errors.New("team lookup unavailable")
	mock.ExpectQuery("(?s)SELECT tg.id.*FROM zt_teamgroup.*WHERE tg.id = ").WithArgs(uint(3)).WillReturnError(lookupErr)
	_, err := svc.SubmitAdjustment(context.Background(), &model.User{Account: "po1"}, SubmitAdjustmentReq{TeamgroupID: 3, Items: []AdjustItemReq{{Account: "dev1", ActionType: ActionAdd, Role: "研发", AvailableHours: 0}}}, false)
	if !errors.Is(err, lookupErr) {
		t.Fatalf("zero hours should pass validation and reach actual lookup: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitZeroHoursHandlerCreatesPendingAdjustment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, mock := newTestService(t)
	teamRows := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"id", "name", "PO"}).AddRow(3, "小组", "po1") }
	mock.ExpectQuery("(?s)SELECT tg.id.*FROM zt_teamgroup.*WHERE tg.id = ").WithArgs(uint(3)).WillReturnRows(teamRows())
	mock.ExpectQuery("(?s)SELECT account FROM `zt_user`").WithArgs("dev1").WillReturnRows(sqlmock.NewRows([]string{"account"}).AddRow("dev1"))
	mock.ExpectQuery("(?s)SELECT t.account.*FROM zt_team t.*t.type = 'teamgroup'").WithArgs(uint(3), "dev1").WillReturnRows(sqlmock.NewRows([]string{"account", "role", "hours"}))
	mock.ExpectBegin()
	mock.ExpectQuery("(?s)SELECT id, parent, grade.*FROM zt_teamgroup.*FOR UPDATE").WithArgs(uint(3)).WillReturnRows(sqlmock.NewRows([]string{"id", "parent", "grade", "path"}).AddRow(3, 0, 1, ",3,"))
	mock.ExpectExec("UPDATE `zt_wb_agileteam_adjustment`").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT `adjustNo` FROM `zt_wb_agileteam_adjustment`").WillReturnRows(sqlmock.NewRows([]string{"adjustNo"}))
	mock.ExpectExec("INSERT INTO `zt_wb_agileteam_adjustment`").WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectExec("INSERT INTO `zt_wb_agileteam_adjustment_item`").
		WithArgs(int64(11), "dev1", ActionAdd, "研发", "", float64(0), float64(0), "po1", "po1", sqlmock.AnyArg(), sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO `zt_wb_agileteam_history`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("(?s)SELECT .* FROM `zt_wb_agileteam_adjustment`").WithArgs(int64(11), 1).WillReturnRows(sqlmock.NewRows([]string{"id", "teamgroupId", "submittedBy", "status"}).AddRow(11, 3, "po1", StatusPending))
	mock.ExpectQuery("(?s)SELECT tg.id.*FROM zt_teamgroup.*WHERE tg.id = ").WithArgs(uint(3)).WillReturnRows(teamRows())
	mock.ExpectQuery("(?s)SELECT .* FROM `zt_wb_agileteam_adjustment_item`").WithArgs(int64(11)).WillReturnRows(sqlmock.NewRows([]string{"id", "account", "actionType", "role", "availableHours"}).AddRow(1, "dev1", ActionAdd, "研发", 0))
	mock.ExpectQuery("(?s)SELECT account, realname FROM `zt_user`").WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).AddRow("po1", "负责人").AddRow("dev1", "开发员"))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	c.Set("currentUser", &model.User{Account: "po1"})
	c.Request = httptest.NewRequest("POST", "/workbench/api/agile-teams/3/adjustments", strings.NewReader(`{"reason":"扩编","items":[{"account":"dev1","actionType":"add","role":"研发","availableHours":0}]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	(&Handler{svc: svc}).SubmitAdjustment(c)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"success":true`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
