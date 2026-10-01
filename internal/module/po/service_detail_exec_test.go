// =============================================================================
// 文件: internal/module/po/service_detail_exec_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 锁定 buildExecution 的聚合口径，作为拆分该超长函数的行为基准：
//       1. Bug 汇总只由命中 bugMap 的研发需求累加，缺失的按 0 计入明细；
//       2. 用例执行率 / 通过率按一位小数四舍五入，分母为 0 时保持 0；
//       3. 测试单阶段由名称含 UAT 判定，进行中 / 已完成计数与状态标签对应；
//       4. F03：扫描源未接入时质量摘要恒为不可用，禁止返回虚构分数。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/driver/mysql, gorm.io/gorm
// =============================================================================

package po

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// newExecMockRepo 构造一个按 buildExecution 调用顺序应答 5 条查询的 DetailService：
// 研发需求 → 任务统计 → 缺陷统计 → 用例统计 → 测试单。
// sets[i] 是第 i 条查询返回的行（外层每个元素一行），未给出的查询返回空集。
func newExecMockRepo(t *testing.T, sets ...[][]any) *DetailService {
	t.Helper()
	gormDB, mock := newPoolEnrichMockDB(t)
	for i := 0; i < 5; i++ {
		var set [][]any
		if i < len(sets) {
			set = sets[i]
		}
		mock.MatchExpectationsInOrder(false)
		rows := sqlmock.NewRows(execColumns(i))
		for _, r := range set {
			vals := make([]driver.Value, len(r))
			for j, v := range r {
				vals[j] = v
			}
			rows.AddRow(vals...)
		}
		mock.ExpectQuery("").WillReturnRows(rows)
	}
	return NewDetailService(NewDemandDetailRepo(gormDB))
}

// execColumns 各查询返回的列名，顺序与 repo 方法的 Select 一致。
func execColumns(i int) []string {
	switch i {
	case 0: // FindDemandStories
		return []string{"id", "title", "status", "stage", "assignedTo", "assigned_to_name", "product", "product_name"}
	case 1: // FindStoryTaskCounts
		return []string{"story", "total", "done"}
	case 2: // FindStoryBugCounts
		return []string{"story", "total", "active", "resolved", "delivery_blocking"}
	case 3: // FindStoryTestCaseCounts
		return []string{"total_count", "executed_count", "passed_count", "failed_count"}
	default: // FindStoryTestTasks
		return []string{"id", "name", "status", "owner", "owner_name", "begin", "end"}
	}
}

// TestBuildExecution_BugTotalsOnlyFromHitStories 锁定 Bug 汇总的累加口径：
// 命中 bugMap 的研发需求才累加，缺失的按 0 计入明细而不虚增总量。
func TestBuildExecution_BugTotalsOnlyFromHitStories(t *testing.T) {
	svc := newExecMockRepo(t,
		[][]any{{1, "研需甲", "active", "dev", "user_a", "甲负责", 10, "产品甲"}},
		[][]any{{1, 4, 3}},       // 任务：共 4 完成 3
		[][]any{{1, 5, 2, 3, 1}}, // 缺陷：总 5 / 活跃 2 / 解决 3 / 交付阻塞 1
		[][]any{{10, 8, 6, 2}},   // 用例：总 10 / 已执行 8 / 通过 6 / 失败 2
	)
	exec, err := svc.buildExecution(t.Context(), 100)
	if err != nil {
		t.Fatalf("buildExecution 报错: %v", err)
	}
	if exec.BugSummary.TotalCount != 5 || exec.BugSummary.ActiveCount != 2 ||
		exec.BugSummary.ResolvedCount != 3 || exec.BugSummary.DeliveryBlocking != 1 {
		t.Fatalf("Bug 汇总 = %+v，期望 5/2/3/1", exec.BugSummary)
	}
	if len(exec.Stories) != 1 || exec.Stories[0].TasksDone != 3 || exec.Stories[0].TasksTotal != 4 {
		t.Fatalf("研需任务进度 = %+v，期望 3/4", exec.Stories)
	}
	if exec.Stories[0].BugsTotal != 5 || exec.Stories[0].BugsActive != 2 {
		t.Fatalf("研需缺陷 = %d/%d，期望 5/2", exec.Stories[0].BugsTotal, exec.Stories[0].BugsActive)
	}
	// 8/10 → 80.0；6/8 → 75.0，均保留一位小数。
	if exec.TestCaseSummary.ExecutionRate != 80.0 {
		t.Errorf("执行率 = %v，期望 80", exec.TestCaseSummary.ExecutionRate)
	}
	if exec.TestCaseSummary.PassRate != 75.0 {
		t.Errorf("通过率 = %v，期望 75", exec.TestCaseSummary.PassRate)
	}
	if exec.TestCaseSummary.UnexecutedCount != 2 {
		t.Errorf("未执行用例 = %d，期望 2", exec.TestCaseSummary.UnexecutedCount)
	}
}

// TestBuildExecution_StoryWithoutBugCounts 锁定缺陷统计缺行时按 0 计入：
// 不得把上一条研发需求的缺陷数带到下一条。
func TestBuildExecution_StoryWithoutBugCounts(t *testing.T) {
	svc := newExecMockRepo(t,
		[][]any{
			{1, "研需甲", "active", "dev", "u1", "甲", 10, "产品"},
			{2, "研需乙", "active", "dev", "u2", "乙", 10, "产品"},
		},
		nil,                      // 任务统计：空
		[][]any{{1, 3, 1, 2, 1}}, // 缺陷：只有 story=1 有数据
		[][]any{{7, 5, 4, 1}},    // 用例
	)
	exec, err := svc.buildExecution(t.Context(), 100)
	if err != nil {
		t.Fatalf("buildExecution 报错: %v", err)
	}
	if len(exec.Stories) != 2 {
		t.Fatalf("研需条数 = %d，期望 2", len(exec.Stories))
	}
	if exec.Stories[0].BugsTotal != 3 || exec.Stories[0].BugsActive != 1 {
		t.Errorf("story 1 缺陷 = %d/%d，期望 3/1", exec.Stories[0].BugsTotal, exec.Stories[0].BugsActive)
	}
	if exec.Stories[1].BugsTotal != 0 || exec.Stories[1].BugsActive != 0 {
		t.Errorf("story 2 缺陷应为空，实际 %d/%d", exec.Stories[1].BugsTotal, exec.Stories[1].BugsActive)
	}
	// 总量只取 story 1，不得翻倍。
	if exec.BugSummary.TotalCount != 3 || exec.BugSummary.ActiveCount != 1 {
		t.Errorf("Bug 汇总 = %+v，期望 3/1", exec.BugSummary)
	}
}

// execRateCase 一组用例统计与期望比率。
type execRateCase struct {
	name                    string
	total, executed, passed int
	wantExec, wantPass      float64
	wantUnexec              int
}

// execRateCases 覆盖一位小数四舍五入与零分母口径。
func execRateCases() []execRateCase {
	return []execRateCase{
		{"one third rounds down", 3, 1, 1, 33.3, 100.0, 2},
		{"two thirds rounds up", 3, 2, 1, 66.7, 50.0, 1},
		{"no executed means zero pass", 5, 0, 0, 0.0, 0.0, 5},
		{"no case at all", 0, 0, 0, 0.0, 0.0, 0},
		{"all executed", 4, 4, 3, 100.0, 75.0, 0},
		{"two thirds pass", 6, 6, 4, 100.0, 66.7, 0},
	}
}

// TestBuildExecution_RatesRoundToOneDecimal 锁定一位小数四舍五入与零分母口径。
func TestBuildExecution_RatesRoundToOneDecimal(t *testing.T) {
	for _, c := range execRateCases() {
		t.Run(c.name, func(t *testing.T) {
			svc := newExecMockRepo(t,
				[][]any{{1, "研需", "active", "dev", "u", "负责", 10, "产品"}},
				nil, nil,
				[][]any{{c.total, c.executed, c.passed, 0}},
			)
			exec, err := svc.buildExecution(t.Context(), 100)
			if err != nil {
				t.Fatalf("buildExecution 报错: %v", err)
			}
			if exec.TestCaseSummary.ExecutionRate != c.wantExec {
				t.Errorf("执行率 = %v，期望 %v", exec.TestCaseSummary.ExecutionRate, c.wantExec)
			}
			if exec.TestCaseSummary.PassRate != c.wantPass {
				t.Errorf("通过率 = %v，期望 %v", exec.TestCaseSummary.PassRate, c.wantPass)
			}
			if exec.TestCaseSummary.UnexecutedCount != c.wantUnexec {
				t.Errorf("未执行 = %d，期望 %d", exec.TestCaseSummary.UnexecutedCount, c.wantUnexec)
			}
		})
	}
}

// TestBuildExecution_TestOrderStageAndStatus 锁定测试单的阶段判定与状态计数。
func TestBuildExecution_TestOrderStageAndStatus(t *testing.T) {
	day := func(d int) *time.Time {
		v := time.Date(2026, 3, d, 0, 0, 0, 0, time.Local)
		return &v
	}
	svc := newExecMockRepo(t,
		[][]any{{1, "研需", "active", "dev", "u", "负责", 10, "产品"}},
		nil, nil, nil,
		[][]any{
			{1, "回归 SIT 用例", "doing", "u1", "甲", day(1), day(9)},
			{2, "UAT 验收", "done", "u2", "乙", day(2), day(8)},
			{3, "阻塞项", "blocked", "u3", "丙", nil, nil},
			{4, "未排期", "unknown", "u4", "", nil, nil},
		},
	)
	exec, err := svc.buildExecution(t.Context(), 100)
	if err != nil {
		t.Fatalf("buildExecution 报错: %v", err)
	}
	if len(exec.TestOrders) != 4 {
		t.Fatalf("测试单条数 = %d，期望 4", len(exec.TestOrders))
	}
	want := []struct{ stage, label, begin, end string }{
		{"SIT", "进行中", "2026-03-01", "2026-03-09"},
		{"UAT", "已完成", "2026-03-02", "2026-03-08"},
		{"SIT", "阻塞", "—", "—"},
		{"SIT", "未开始", "—", "—"},
	}
	for i, w := range want {
		got := exec.TestOrders[i]
		if got.Stage != w.stage || got.StatusLabel != w.label ||
			got.BeginDate != w.begin || got.EndDate != w.end {
			t.Errorf("第 %d 条测试单 = %+v，期望 stage=%s label=%s begin=%s end=%s",
				i, got, w.stage, w.label, w.begin, w.end)
		}
	}
	sum := exec.TestOrderSummary
	if sum.TotalCount != 4 || sum.DoingCount != 1 || sum.DoneCount != 1 || sum.StoriesCount != 1 {
		t.Errorf("测试单汇总 = %+v，期望 4/1/1/1", sum)
	}
}

// TestBuildExecution_QualityUnavailable 锁定 F03：扫描源未接入时质量摘要恒不可用。
func TestBuildExecution_QualityUnavailable(t *testing.T) {
	svc := newExecMockRepo(t, [][]any{{1, "研需", "active", "dev", "u", "负责", 10, "产品"}})
	exec, err := svc.buildExecution(t.Context(), 100)
	if err != nil {
		t.Fatalf("buildExecution 报错: %v", err)
	}
	if exec.QualitySummary.Available || exec.QualitySummary.Source != "none" || exec.QualitySummary.AvgScore != 0 {
		t.Errorf("质量摘要应恒为不可用，实际 %+v", exec.QualitySummary)
	}
	if exec.QualityOverview.Available || exec.QualityOverview.Source != "none" {
		t.Errorf("质量总览应恒为不可用，实际 %+v", exec.QualityOverview)
	}
	if exec.AppQualityTree != nil {
		t.Errorf("未接入扫描源时应用质量树应为空，实际 %+v", exec.AppQualityTree)
	}
	if exec.LeadsMatrix.TestLead != "—" {
		t.Errorf("测试负责人未接入时应为占位符，实际 %q", exec.LeadsMatrix.TestLead)
	}
}

// TestBuildExecution_DevLeadsDeduped 锁定研发负责人去重：
// 空值与占位符 "—" 不计入，重复负责人只出现一次。
func TestBuildExecution_DevLeadsDeduped(t *testing.T) {
	svc := newExecMockRepo(t, [][]any{
		{1, "研需甲", "active", "dev", "u1", "张三", 10, "产品"},
		{2, "研需乙", "active", "dev", "u1", "张三", 10, "产品"},
		{3, "研需丙", "active", "dev", "u2", "李四", 10, "产品"},
		{4, "研需丁", "active", "dev", "", "", 10, "产品"},
		{5, "研需戊", "active", "dev", "u3", "—", 10, "产品"},
	})
	exec, err := svc.buildExecution(t.Context(), 100)
	if err != nil {
		t.Fatalf("buildExecution 报错: %v", err)
	}
	if len(exec.LeadsMatrix.DevLeads) != 2 {
		t.Fatalf("研发负责人 = %v，期望去重后 2 人", exec.LeadsMatrix.DevLeads)
	}
	seen := map[string]bool{}
	for _, l := range exec.LeadsMatrix.DevLeads {
		if seen[l] {
			t.Errorf("研发负责人 %q 重复", l)
		}
		seen[l] = true
	}
	if !seen["张三"] || !seen["李四"] {
		t.Errorf("研发负责人 = %v，期望含张三与李四", exec.LeadsMatrix.DevLeads)
	}
}

// TestBuildExecution_EmptyStoriesStillBuilds 执行单为空时仍须返回非 nil 结构，
// 且速率为 0、AppQualityTree 为空，避免前端拿到 nil 指针。
func TestBuildExecution_EmptyStoriesStillBuilds(t *testing.T) {
	svc := newExecMockRepo(t)
	exec, err := svc.buildExecution(t.Context(), 100)
	if err != nil {
		t.Fatalf("buildExecution 报错: %v", err)
	}
	if exec == nil {
		t.Fatal("无研发需求时也应返回非 nil 结构")
	}
	if len(exec.Stories) != 0 || len(exec.TestOrders) != 0 {
		t.Errorf("无数据时明细应为空，实际 stories=%d orders=%d", len(exec.Stories), len(exec.TestOrders))
	}
	if exec.TestCaseSummary.ExecutionRate != 0 || exec.TestCaseSummary.PassRate != 0 {
		t.Errorf("无用例时速率应为 0，实际 %+v", exec.TestCaseSummary)
	}
}
