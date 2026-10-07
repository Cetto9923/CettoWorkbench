// =============================================================================
// 文件: internal/module/po/home_rules_integration_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用只读 CTE 样例在真实 MySQL/OceanBase 执行首页 SQL，并核对同批主操作。
// =============================================================================
package po

import (
	"context"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/perm"
)

type homeRuleSample struct {
	status, bra, assigned, rd, creator, originator, overall string
	selected                                                bool
	key                                                     primaryaction.ActionKey
}

var homeRuleSamples = []homeRuleSample{
	{"draft", "", "bob", "", "alice", "", "0", true, primaryaction.KeySubmitReview},
	{"draft", "alice", "alice", "", "bob", "", "0", false, ""},
	{"refuse", "", "bob", "", "alice", "", "0", true, primaryaction.KeySubmitReview},
	{"refuse", "alice", "alice", "", "bob", "", "0", false, ""},
	{"wait", "", "bob", "", "alice", "", "0", true, primaryaction.KeyApprove},
	{"wait", "", "bob", "", "bob", "", "0", true, primaryaction.KeyApprove},
	{"wait", "alice", "alice", "", "bob", "", "0", false, ""},
	{"wait", "", "bob", "", "alice", "", "0", true, primaryaction.KeyWithdrawReview},
	{"active", "alice", "bob", "", "bob", "", "0", true, primaryaction.KeyClarify},
	{"active", "bob", "alice", "", "bob", "", "0", true, primaryaction.KeyClarify},
	{"active", "alice", "alice", "", "bob", "", "0", true, primaryaction.KeyClarify},
	{"active", "bob", "bob", "", "bob", "", "0", false, primaryaction.KeyClarify},
	{"clarified", "alice", "bob", "", "bob", "", "0", true, primaryaction.KeySchedule},
	{"clarified", "bob", "alice", "", "bob", "", "0", false, primaryaction.KeySchedule},
	{"developing", "alice", "bob", "", "bob", "", "0", true, primaryaction.KeySubmitTest},
	{"developing", "", "alice", "", "bob", "", "0", true, primaryaction.KeySubmitTest},
	{"developing", "bob", "alice", "", "bob", "", "0", false, primaryaction.KeySubmitTest},
	{"testing", "bob", "bob", "", "bob", "", "0", true, primaryaction.KeyViewTestOrder},
	{"testing", "bob", "bob", "", "bob", "", "0", true, primaryaction.KeyViewTestOrder},
	{"testing", "alice", "alice", "", "bob", "", "0", false, primaryaction.KeyViewTestOrder},
	{"waitacceptance", "bob", "bob", "alice", "bob", "", "0", true, primaryaction.KeyAcceptDone},
	{"waitacceptance", "bob", "alice", "", "bob", "", "0", true, primaryaction.KeyAcceptDone},
	{"waitacceptance", "bob", "bob", "carol", "bob", "", "0", true, primaryaction.KeyRemindAccept},
	{"waitacceptance", "alice", "alice", "carol", "bob", "", "0", false, primaryaction.KeyRemindAccept},
	{"acceptanced", "alice", "bob", "", "bob", "", "0", true, primaryaction.KeyDeliver},
	{"acceptanced", "bob", "alice", "", "bob", "", "0", false, primaryaction.KeyDeliver},
	{"waitdeliver", "alice", "alice", "", "bob", "", "0", false, ""},
	{"released", "bob", "bob", "", "bob", "alice", "0", true, primaryaction.KeyEvaluate},
	{"released", "bob", "bob", "", "alice", "", "0", true, primaryaction.KeyEvaluate},
	{"released", "alice", "alice", "", "alice", "bob", "0", false, ""},
	{"released", "alice", "alice", "", "alice", "alice", "1", false, ""},
}

// CTE 只在当前 SELECT 内覆盖同名表，不创建临时表或写入真实业务数据。
func homeRuleCTE() (string, []any) {
	columns := "id,status,BRA,assignedTo,RD,createdBy,originator,overall,deleted,parent,developFinish,testFinish,verifyFinish,estimateLaunch,QD,mainDevelopers,locked"
	parts, args := []string{}, []any{}
	for i, row := range homeRuleSamples {
		parts = append(parts, "SELECT ?,?,?,?,?,?,?,?,'0',0,'2099-01-01',NULL,NULL,NULL,'','','1'")
		args = append(args, i+1, row.status, row.bra, row.assigned, row.rd, row.creator, row.originator, row.overall)
	}
	tables := []string{
		"zt_demand(" + columns + ") AS (" + strings.Join(parts, " UNION ALL ") + ")",
		"zt_demandreview AS (SELECT 5 demand,'alice' reviewer,'' result UNION ALL SELECT 6,'alice','' UNION ALL SELECT 7,'alice','pass')",
		"zt_demandclarify AS (SELECT 9 demand,'alice' PM UNION ALL SELECT 9,'bob' UNION ALL SELECT 12,'alice')",
		"zt_action AS (SELECT 1 id,'demand' objectType,23 objectID,'startacceptance' action,'bob' actor UNION ALL SELECT 2,'demand',23,'startacceptance','alice' UNION ALL SELECT 3,'demand',24,'startacceptance','alice' UNION ALL SELECT 4,'demand',24,'startacceptance','bob')",
		"zt_testtask AS (SELECT 1 id,'0' deleted,'alice' owner,'' members UNION ALL SELECT 2,'0','bob','alice,carol')",
		"zt_testrun AS (SELECT 1 task,1 `case` UNION ALL SELECT 2,2)",
		"zt_case AS (SELECT 1 id,1 story UNION ALL SELECT 2,2)",
		"zt_story AS (SELECT 1 id,'0' deleted,18 fromDemand UNION ALL SELECT 2,'0',19)",
		"zt_demandappraise AS (SELECT 0 demand,'' appraiseBy,NULL appraiseTime WHERE 1=0)",
	}
	return "WITH " + strings.Join(tables, ",") + " ", args
}

func TestHomeRulesReadOnlySQLAndActions(t *testing.T) {
	dsn := os.Getenv("WB_GAP_TEST_DSN")
	if dsn == "" {
		t.Skip("set WB_GAP_TEST_DSN for read-only MySQL/OceanBase CTE validation")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("read-only test connection unavailable")
	}
	raw, _ := db.DB()
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Error(err)
		}
	})
	repo := NewRepo(db, nil)
	cte, fixtureArgs := homeRuleCTE()
	where, whereArgs := currentHandlerDemandWhere("alice")
	var facts []struct {
		ID      int
		Handler bool
	}
	if err := db.Raw(cte+"SELECT id, "+where+" AS handler FROM zt_demand ORDER BY id", append(fixtureArgs, whereArgs...)...).Scan(&facts).Error; err != nil {
		t.Fatal(err)
	}
	actor := &model.User{Account: "alice"}
	ctx := perm.WithGranted(context.Background(), map[string]bool{perm.PoHomeList.String(): true, perm.ScheduleList.String(): true})
	for _, focusStatus := range []string{"all", "accept", "clarify", "schedule", "developing", "testing", "waitacceptance", "acceptanced", "publish", "released"} {
		assertHomeRuleSelection(t, repo, cte, fixtureArgs, focusStatus)
	}
	for _, fact := range facts {
		tc := homeRuleSamples[fact.ID-1]
		pa := deriveDemandPrimaryAction(ctx, actor, primaryActionFactsDemand{ObjectID: uint(fact.ID), Kind: primaryaction.ObjectBusinessDemand, Stage: deriveStageKey("", tc.status), Status: tc.status, IsHandler: fact.Handler, IsCreator: tc.creator == "alice", CanReview: fact.ID == 5 || fact.ID == 6, IsAcceptanceOwner: acceptanceOwner(tc.rd, tc.assigned) == "alice", HasDemandRelation: true, HasPendingEvaluate: tc.status == "released" && fact.Handler, TestsCount: 1, FirstTestURL: "/testtask/1"})
		if pa.Key != string(tc.key) {
			t.Errorf("%d %s: key=%s want=%s", fact.ID, tc.status, pa.Key, tc.key)
		}
		wantEnabled := tc.selected || tc.status == "acceptanced"
		if pa.Enabled != wantEnabled {
			t.Errorf("%d %s: selected=%v enabled=%v", fact.ID, tc.status, tc.selected, pa.Enabled)
		}
	}
}

func assertHomeRuleSelection(t *testing.T, repo *Repo, cte string, fixtureArgs []any, status string) {
	t.Helper()
	query := repo.homeFocusQueryWithReviews(t.Context(), "alice", DemandsReq{Focus: "my_action", Status: status}, nil)
	var rows []struct{ ID, StageIndex int }
	stmt := query.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	args := append(append([]any{}, fixtureArgs...), stmt.Vars...)
	if err := repo.db.Raw(cte+stmt.SQL.String(), args...).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	selected := map[int]int{}
	for _, row := range rows {
		if _, ok := selected[row.ID]; ok {
			t.Fatal("duplicate requirement")
		}
		selected[row.ID] = row.StageIndex
	}
	for i, tc := range homeRuleSamples {
		stage := deriveStageKey("", tc.status)
		expectedStage := map[primaryaction.StageKey]int{primaryaction.StageAccept: 1, primaryaction.StageClarify: 2, primaryaction.StageSchedule: 3, primaryaction.StageDeveloping: 4, primaryaction.StageTesting: 5, primaryaction.StageAcceptance: 6, primaryaction.StageDeliver: 7, primaryaction.StageFeedback: 9}[stage]
		want := tc.selected && (status == "all" || homeFocusStageIndex(status) == expectedStage)
		index, got := selected[i+1]
		if got != want || got && index != expectedStage {
			t.Errorf("%s id=%d: selected=%v stage=%d want=%v/%d", status, i+1, got, index, want, expectedStage)
		}
	}
}
