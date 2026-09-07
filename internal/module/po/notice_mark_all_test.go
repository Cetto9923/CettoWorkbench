package po

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"workbench/internal/model"
)

// 检查真实执行语句的边界，而不是让 mock 仅接受任意 INSERT。
func TestNoticeMarkAllRead_FilteredAcrossPages(t *testing.T) {
	for _, readState := range []string{"unread", "read"} {
		t.Run(readState, func(t *testing.T) {
			matcher := sqlmock.QueryMatcherFunc(func(_, actual string) error {
				for _, part := range []string{
					"INSERT INTO zt_workbench_notify_reads", "LEFT JOIN zt_action", "nr.account = ?",
					"FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0", "LOWER(n.subject) LIKE ?",
					"COALESCE(a.objectType, n.objectType) = ?", "n.createdDate >= ?",
					"nr.id IS NULL", "NOT (", "END = ?", "ON DUPLICATE KEY UPDATE readAt = zt_workbench_notify_reads.readAt",
				} {
					if !strings.Contains(actual, part) {
						return fmt.Errorf("missing scoped predicate %q in %s", part, actual)
					}
				}
				if readState == "read" && !strings.Contains(actual, "nr.id IS NOT NULL") {
					return fmt.Errorf("read filter was discarded")
				}
				for _, forbidden := range []string{"LIMIT", "OFFSET", "DELETE", "UPDATE zt_notify"} {
					if strings.Contains(actual, forbidden) {
						return fmt.Errorf("unexpected %s", forbidden)
					}
				}
				return nil
			})
			sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			repo := NewRepo(nil, db)
			req := NoticeListReq{QuickView: "all", Category: "business", ObjectType: "task", TimeRange: "3d", ReadState: readState, NeedAction: "none", Keyword: "100%_", Page: 9, PageSize: 1}
			want := int64(27)
			if readState == "read" {
				want = 0
			}
			mock.ExpectExec("scoped insert").WithArgs("alice", sqlmock.AnyArg(), "alice", "alice", "%100\\%\\_%", "%100\\%\\_%", "%100\\%\\_%", "task", sqlmock.AnyArg(), "business").WillReturnResult(sqlmock.NewResult(0, want))
			got, err := repo.SaveAllNoticeReads(context.Background(), "alice", req)
			if err != nil || got != want {
				t.Fatalf("got %d, %v", got, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNoticeMarkAllRead_RejectsInvalidRequestsBeforeRepo(t *testing.T) {
	svc := &Service{}
	for _, actor := range []*model.User{nil, {Account: " "}} {
		if _, err := svc.NoticeMarkAllRead(context.Background(), actor, NoticeListReq{}); err == nil {
			t.Fatal("missing actor accepted")
		}
	}
	for _, req := range []NoticeListReq{{QuickView: "bad"}, {Category: "bad"}, {ReadState: "bad"}, {NeedAction: "bad"}, {TimeRange: "bad"}} {
		if _, err := svc.NoticeMarkAllRead(context.Background(), &model.User{Account: "alice"}, req); err == nil {
			t.Fatal("invalid filters accepted")
		}
	}
}

func TestNoticeMarkAllRead_RequiresExplicitFilterPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{"", "null", "{}", `{"filters":null}`, `{"filters":{"category":"bad"}}`} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/notice/read-all", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		(&Handler{}).NoticeMarkAllRead(c)
		if w.Code != 400 {
			t.Fatalf("body %q status %d", body, w.Code)
		}
	}
}
