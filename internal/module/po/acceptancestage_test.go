package po

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"
)

// TestAcceptanceStageMatchesHomeFirstHit 锁定验收列表与首页 CASE 首命中互斥：
// testing 阶段先占有全部 status=testing，验收列表只保留 waitacceptance 归属。
func TestAcceptanceStageMatchesHomeFirstHit(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	caseSQL, _ := repo.demandStageCase(context.Background(), "003030")
	testingAt := strings.Index(caseSQL, stageThenMark("testing"))
	acceptanceAt := strings.Index(caseSQL, stageThenMark("waitacceptance"))
	if testingAt < 0 || acceptanceAt < 0 || testingAt > acceptanceAt {
		t.Fatalf("home CASE order testing=%d acceptance=%d sql=%s", testingAt, acceptanceAt, caseSQL)
	}

	acceptanceSQL, acceptanceArgs := stageWhereSQL(t, db, "waitacceptance")
	if strings.Contains(acceptanceSQL, "testFinish") || strings.Contains(acceptanceSQL, "testing") {
		t.Fatalf("acceptance list still unions testing-due rows: %s", acceptanceSQL)
	}
	if !strings.Contains(acceptanceSQL, "status = ?") {
		t.Fatalf("acceptance list lost waitacceptance owner predicate: %s", acceptanceSQL)
	}
	if len(acceptanceArgs) != 1 || acceptanceArgs[0] != "waitacceptance" {
		t.Fatalf("acceptance args = %#v", acceptanceArgs)
	}

	for _, status := range []string{"accept", "clarify", "schedule", "developing", "testing"} {
		earlierSQL, _ := stageWhereSQL(t, db, status)
		if strings.Contains(earlierSQL, "waitacceptance") {
			t.Fatalf("stage %s overlaps acceptance status: %s", status, earlierSQL)
		}
	}
	testingSQL, _ := stageWhereSQL(t, db, "testing")
	if !strings.Contains(testingSQL, "status IN") || strings.Contains(testingSQL, "testFinish") {
		t.Fatalf("testing stage no longer claims every testing row: %s", testingSQL)
	}
	assertOtherStageMarkers(t, db)
}

func stageThenMark(status string) string {
	for index, stage := range valueStreamStages {
		if stage.status == status {
			return fmt.Sprintf("THEN %d", index)
		}
	}
	return ""
}

func stageWhereSQL(t *testing.T, db *gorm.DB, status string) (string, []interface{}) {
	t.Helper()
	filter, ok := mysqlStageFilters[status]
	if !ok {
		t.Fatalf("missing filter %s", status)
	}
	stmt := applyDemandStage(db.Table("zt_demand"), "003030", filter).
		Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
	return stmt.SQL.String(), stmt.Vars
}

func assertOtherStageMarkers(t *testing.T, db *gorm.DB) {
	t.Helper()
	markers := map[string]string{
		"accept": "status IN", "clarify": "status IN", "schedule": "mainDevelopers",
		"developing": "status IN", "acceptanced": "status IN", "publish": "status IN",
		"released": "overall",
	}
	for status, marker := range markers {
		sql, _ := stageWhereSQL(t, db, status)
		if !strings.Contains(sql, marker) || strings.Contains(sql, "accepter = ?") {
			t.Fatalf("stage %s marker %q drifted: %s", status, marker, sql)
		}
	}
	// 验证提测与发起交付已去除日期/BRA门槛（全量进格）
	devSQL, _ := stageWhereSQL(t, db, "developing")
	if strings.Contains(devSQL, "developFinish") {
		t.Fatalf("developing stage must not require developFinish threshold: %s", devSQL)
	}
	accSQL, _ := stageWhereSQL(t, db, "acceptanced")
	if strings.Contains(accSQL, "deliverDate") || strings.Contains(accSQL, "BRA = ?") {
		t.Fatalf("acceptanced stage must not require deliverDate or BRA threshold: %s", accSQL)
	}
}
