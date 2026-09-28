// =============================================================================
// 文件: internal/module/po/repoboardmetrics_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 小组效能周期查询不得引用 zt_story 不存在的 finishedDate，并用发布日计算实施周期。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestComputeCyclesUsesStoryReleasedDate(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(
		func(_, actual string) error {
			if strings.Contains(actual, "finishedDate") {
				return errors.New("zt_story 没有 finishedDate，周期查询不能引用该列")
			}
			if !strings.Contains(actual, "st.releasedDate") || !strings.Contains(actual, "st.assignedDate") {
				return errors.New("实施周期应使用 releasedDate 与 assignedDate")
			}
			return nil
		},
	)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("zt_story").WillReturnRows(sqlmock.NewRows([]string{"delivery", "implement", "finishedCnt", "overIterCnt"}).AddRow(12.5, 8, 4, 1))

	repo := NewRepo(gormDB, gormDB)
	ms := boardGroupMetrics()
	repo.computeCycles(context.Background(), ms, []string{"po1"})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if got := metricOf(ms, "delivery").Value; got != "12.5天" {
		t.Fatalf("delivery = %s", got)
	}
	if got := metricOf(ms, "implement").Value; got != "8.0天" {
		t.Fatalf("implement = %s", got)
	}
	if got := metricOf(ms, "overIteration").Value; got != "25.0%" {
		t.Fatalf("overIteration = %s", got)
	}
}
