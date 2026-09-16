package po

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"workbench/internal/model"
)

func TestDeliverHandlerChecksBlockersForBothBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{`{"comment":"交付"}`, `{"deliverDate":"2026-09-20","verifyPlan":"验证","verifier":"alice","verifyDate":"1","isCarReview":"0","isGrayVerifyPlan":"0"}`} {
		for _, failedQuery := range []bool{false, true} {
			t.Run(body+map[bool]string{false: "blocked", true: "query-error"}[failedQuery], func(t *testing.T) {
				svc, mock := newServiceForDeliverTest(t)
				mock.ExpectQuery("(?s)SELECT.*FROM .?zt_demand.?.*WHERE id = ").WillReturnRows(sqlmock.NewRows([]string{"id", "status", "deleted", "assignedTo", "BRA"}).AddRow(1001, "acceptanced", "0", "alice", "alice"))
				q := mock.ExpectQuery("(?s)SELECT.*severe_count.*open_count.*FROM zt_bug").WithArgs(uint(1001))
				wantStatus := 409
				wantMessage := "严重缺陷未关闭"
				if failedQuery {
					q.WillReturnError(errors.New("blocker query unavailable"))
					wantStatus = 500
					wantMessage = "检查交付阻塞缺陷失败"
				} else {
					q.WillReturnRows(sqlmock.NewRows([]string{"severe_count", "open_count"}).AddRow(2, 3))
				}
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Params = gin.Params{{Key: "id", Value: "1001"}}
				c.Set("currentUser", &model.User{Account: "alice"})
				c.Request = httptest.NewRequest("POST", "/demands/1001/deliver", strings.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				(&Handler{svc: svc}).DeliverDemand(c)
				if w.Code != wantStatus || !strings.Contains(w.Body.String(), wantMessage) {
					t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestMinimalDeliverBlockedReturns409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, mock := newServiceForDeliverTest(t)
	mock.ExpectQuery("(?s)SELECT.*FROM `zt_demand`.*WHERE id = ").WithArgs(uint(1002), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "deleted", "assignedTo"}).AddRow(1002, "acceptanced", "0", "alice"))
	mock.ExpectQuery("(?s)SELECT.*severe_count.*open_count.*FROM zt_bug").WithArgs(uint(1002)).
		WillReturnRows(sqlmock.NewRows([]string{"severe_count", "open_count"}).AddRow(3, 1))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1002"}}
	c.Set("currentUser", &model.User{Account: "alice"})
	// 极简 body：仅 comment，走 DeliverHomeDemand -> homeAction("deliver")，同样受严重缺陷拦截
	c.Request = httptest.NewRequest("POST", "/demands/1002/deliver", strings.NewReader(`{"comment":"交付"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	(&Handler{svc: svc}).DeliverDemand(c)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "严重缺陷未关闭") {
		t.Fatalf("minimal deliver with severe bugs should be 409, got status=%d body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMinimalDeliverHandlerAllowsClearDemand(t *testing.T) {
	svc, mock := newServiceForDeliverTest(t)
	mock.ExpectQuery("(?s)SELECT.*FROM `zt_demand`.*WHERE id = ").WithArgs(uint(1001), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "deleted", "assignedTo"}).AddRow(1001, "acceptanced", "0", "alice"))
	mock.ExpectQuery("(?s)SELECT.*severe_count.*open_count.*FROM zt_bug").WithArgs(uint(1001)).
		WillReturnRows(sqlmock.NewRows([]string{"severe_count", "open_count"}).AddRow(0, 0))
	mock.ExpectBegin()
	mock.ExpectQuery("(?s)SELECT id, status, deleted, product FROM `zt_demand`.*FOR UPDATE").WithArgs(uint(1001), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "deleted", "product"}).AddRow(1001, "acceptanced", "0", "1"))
	mock.ExpectExec("UPDATE `zt_demand`").WithArgs("alice", sqlmock.AnyArg(), "waitdeliver", uint(1001), "acceptanced").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `zt_action`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1001"}}
	c.Set("currentUser", &model.User{Account: "alice"})
	c.Request = httptest.NewRequest("POST", "/demands/1001/deliver", strings.NewReader(`{"comment":"交付"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	(&Handler{svc: svc}).DeliverDemand(c)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"redirectUrl":"/home"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
