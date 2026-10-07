// =============================================================================
// 文件: internal/module/po/home_focus_handler_golden_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 锁定办理人条件生成的 SQL 文本与参数顺序，任何重构都必须逐字一致。
// =============================================================================
package po

import (
	"fmt"
	"strings"
	"testing"
)

const goldenReviewSQL = "EXISTS (SELECT 1 FROM zt_demandreview dr WHERE dr.demand = zt_demand.id AND dr.reviewer = ? AND (dr.result IS NULL OR dr.result = ''))"

// goldenWhere 记录基准版本的完整条件文本。拆段重构后必须与本值逐字相同，
// 否则说明列表查询口径被改动，而不是等价重构。
const goldenWhere = `((status IN ('draft', 'refuse') AND createdBy = ?)
 OR (status = 'wait' AND (createdBy = ? OR ` + goldenReviewSQL + `))
 OR (status = 'active' AND (assignedTo = ? OR BRA = ?))
 OR (status = 'clarified' AND BRA = ?)
 OR (status = 'developing' AND (BRA = ? OR (IFNULL(BRA, '') = '' AND assignedTo = ?)))
 OR (status = 'testing' AND EXISTS (SELECT 1 FROM zt_testtask tt JOIN zt_testrun tr ON tr.task = tt.id JOIN zt_case c ON c.id = tr.case JOIN zt_story s ON s.id = c.story AND s.deleted = '0' WHERE s.fromDemand = zt_demand.id AND tt.deleted = '0' AND (tt.owner = ? OR FIND_IN_SET(?, REPLACE(tt.members, ' ', '')) > 0)))
 OR (status = 'waitacceptance' AND ((RD = ? OR (IFNULL(RD, '') = '' AND assignedTo = ?)) OR ((SELECT a.actor FROM zt_action a WHERE a.objectType = 'demand' AND a.objectID = zt_demand.id AND a.action = 'startacceptance' ORDER BY a.id DESC LIMIT 1) = ? AND NOT (RD = ? OR (IFNULL(RD, '') = '' AND assignedTo = ?)))))
 OR (status = 'acceptanced' AND BRA = ?)
 OR (status = 'released' AND overall = '0' AND (originator = ? OR (IFNULL(originator, '') = '' AND createdBy = ?))))`

// goldenAccountArgs 显式写出 18 个账号占位，避免测试依赖被测代码里的辅助函数。
// 评审列表为空时评审子句的 1 个占位被移除，故为前 2 个 + 后 16 个。
var goldenAccountArgs = []interface{}{
	"alice", "alice", "alice", "alice", "alice", "alice", "alice", "alice", "alice",
	"alice", "alice", "alice", "alice", "alice", "alice", "alice", "alice", "alice",
}

var goldenEmptyReviewArgs = append(append([]interface{}{}, goldenAccountArgs[:2]...), goldenAccountArgs[3:]...)

type goldenCase struct {
	name    string
	got     string
	gotArgs []interface{}
	want    string
	wantArg []interface{}
}

func checkGoldenCases(t *testing.T, cases []goldenCase) {
	t.Helper()
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s SQL 变化：\n期望 %s\n实际 %s", tc.name, tc.want, tc.got)
		}
		if fmt.Sprint(tc.gotArgs) != fmt.Sprint(tc.wantArg) {
			t.Errorf("%s 参数变化：\n期望 %v\n实际 %v", tc.name, tc.wantArg, tc.gotArgs)
		}
	}
}

func TestHandlerDemandWhereGolden(t *testing.T) {
	baseSQL, baseArgs := currentHandlerDemandWhere("alice")
	emptySQL, emptyArgs := currentHandlerDemandWhereWithReviews("alice", []int{})
	idsSQL, idsArgs := currentHandlerDemandWhereWithReviews("alice", []int{5, 6})
	nilSQL, nilArgs := currentHandlerDemandWhereWithReviews("alice", nil)

	withIDs := append(append([]interface{}{}, goldenAccountArgs[:2]...), append([]interface{}{[]int{5, 6}}, goldenAccountArgs[3:]...)...)

	checkGoldenCases(t, []goldenCase{
		{"基准条件", baseSQL, baseArgs, goldenWhere, goldenAccountArgs},
		{"评审列表为空", emptySQL, emptyArgs, strings.Replace(goldenWhere, goldenReviewSQL, "1 = 0", 1), goldenEmptyReviewArgs},
		{"评审列表非空", idsSQL, idsArgs, strings.Replace(goldenWhere, goldenReviewSQL, "id IN (?)", 1), withIDs},
		{"评审列表未加载", nilSQL, nilArgs, goldenWhere, goldenAccountArgs},
	})
}

func TestCurrentHandlerDemandKeywordWhereGolden(t *testing.T) {
	keyword := currentHandlerDemandKeywordWhere()
	if strings.Count(keyword, "?") != 18 {
		t.Fatalf("关键词占位符数量变化：%d", strings.Count(keyword, "?"))
	}
	if !strings.Contains(keyword, "LOWER(IFNULL(dr.reviewer, '')) LIKE ?") {
		t.Fatal("评审人关键词条件缺失")
	}
	if !strings.Contains(keyword, "LOWER(IFNULL(tt.members, '')) LIKE ?") {
		t.Fatal("测试成员关键词条件缺失")
	}
}
