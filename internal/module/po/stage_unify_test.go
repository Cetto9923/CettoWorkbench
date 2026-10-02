// =============================================================================
// 文件: internal/module/po/stage_unify_test.go
// 模块: PO 工作台
// 类型: test
// 职责: A10 阶段表统一回归：详情页当前阶段编码、阶段条与列表行同源、版本跟进按显示名筛选命中。
// =============================================================================

package po

import "testing"

// findStage 返回指定 key 的阶段节点。
func findStage(stages []ValueStreamItem, key string) (ValueStreamItem, bool) {
	for _, s := range stages {
		if s.Key == key {
			return s, true
		}
	}
	return ValueStreamItem{}, false
}

// A10 核心回归：需求处于 developing 时，「提测」节点必须自己是 current。
// 此前 Map 把 developing 映射成 submittest，导致「提测」被误标为当前、「研发」被标为已完成。
func TestBuildValueStreamDevelopingIsCurrent(t *testing.T) {
	svc := &DetailService{}
	vs := svc.buildValueStream(&DemandDetailRow{ID: 1, Status: "developing"})

	node, ok := findStage(vs.Stages, "developing")
	if !ok {
		t.Fatalf("阶段条缺少 developing 节点：%+v", vs.Stages)
	}
	if node.Status != "current" {
		t.Fatalf("developing 节点状态 = %q，期望 current", node.Status)
	}
	if node.Label != "提测" {
		t.Fatalf("developing 节点显示名 = %q，期望与首页一致的「提测」", node.Label)
	}
	// 旧的 submittest 编码不再产出，页面不应再出现第二个「提测」节点。
	if _, dup := findStage(vs.Stages, "submittest"); dup {
		t.Fatal("阶段条不应再出现 submittest 节点")
	}
	// 排在 developing 之前的节点才算完成。
	if prev, _ := findStage(vs.Stages, "schedule"); prev.Status != "done" {
		t.Fatalf("schedule 节点状态 = %q，期望 done", prev.Status)
	}
	if next, _ := findStage(vs.Stages, "testing"); next.Status != "future" {
		t.Fatalf("testing 节点状态 = %q，期望 future", next.Status)
	}
}

// 详情页阶段条必须与首页价值流同源：节点数 = 首页九阶段 + 已关闭终态。
func TestBuildValueStreamMatchesHomeStages(t *testing.T) {
	svc := &DetailService{}
	vs := svc.buildValueStream(&DemandDetailRow{ID: 2, Status: "testing"})

	if len(vs.Stages) != len(valueStreamStages) {
		t.Fatalf("阶段节点数 = %d，期望 %d（首页九阶段 + 已关闭）", len(vs.Stages), len(valueStreamStages))
	}
	for i, def := range valueStreamStages[1:] {
		if vs.Stages[i].Key != def.status || vs.Stages[i].Label != def.label {
			t.Fatalf("节点[%d] = (%q,%q)，期望 (%q,%q)", i,
				vs.Stages[i].Key, vs.Stages[i].Label, def.status, def.label)
		}
	}
	if last := vs.Stages[len(vs.Stages)-1]; last.Key != "closed" || last.Label != "已关闭" {
		t.Fatalf("末节点 = (%q,%q)，期望 (closed,已关闭)", last.Key, last.Label)
	}
}

// 验收在首页是两个阶段，详情页不得再合并；未映射状态回退「未知阶段」节点。
func TestBuildValueStreamAcceptanceSplitAndUnknown(t *testing.T) {
	svc := &DetailService{}
	wa := svc.buildValueStream(&DemandDetailRow{ID: 3, Status: "waitacceptance"})
	if n, _ := findStage(wa.Stages, "waitacceptance"); n.Status != "current" {
		t.Fatalf("waitacceptance 节点状态 = %q，期望 current", n.Status)
	}

	unknown := svc.buildValueStream(&DemandDetailRow{ID: 4, Status: "nonexistent"})
	if n, ok := findStage(unknown.Stages, "unknown"); !ok || n.Status != "current" {
		t.Fatalf("未映射状态应补 unknown 当前节点，实际 %+v", unknown.Stages)
	}
}

// A11 遗留修复：阶段条显示名与列表行 Stage 同源，按「联调测试」筛选必须能命中。
// 此前阶段条用 stageLabels 的「测试」，行用 Map 的「测试中」，点格子必然筛空。
func TestVersionFollowStageFilterMatchesStageBar(t *testing.T) {
	svc := &Service{}
	now := nowFunc()
	rows := []vfDemandRow{
		{ID: 1, Stage: "testing", Status: "testing"},
		{ID: 2, Stage: "testing", Status: "testing"},
		{ID: 3, Stage: "waitacceptance", Status: "waitacceptance"},
	}
	items := svc.buildVFItems(nil, rows, now, VersionFollowListReq{})

	var resp VersionFollowListResp
	svc.fillVFSummary(&resp, items, rows, now)

	counts := map[string]int{}
	for _, sc := range resp.StageCounts {
		counts[sc.Stage] = sc.Count
	}
	bar, ok := counts["联调测试"]
	if !ok {
		t.Fatalf("阶段条缺少「联调测试」：%+v", resp.StageCounts)
	}
	if bar != 2 {
		t.Fatalf("阶段条「联调测试」计数 = %d，期望 2", bar)
	}

	// 同一个显示名传给筛选，命中条数必须等于格子计数。
	got := 0
	for _, it := range items {
		if vfMatchStage(it, "联调测试") {
			got++
		}
	}
	if got != bar {
		t.Fatalf("按「联调测试」筛选命中 %d 条，阶段条计数 %d，两者必须一致", got, bar)
	}

	// 阶段条上的每个格子都必须能在列表里筛出对应条数，不能有筛空的格子。
	for _, sc := range resp.StageCounts {
		hit := 0
		for _, it := range items {
			if vfMatchStage(it, sc.Stage) {
				hit++
			}
		}
		if hit != sc.Count {
			t.Fatalf("阶段条「%s」计数 %d，按该名字筛选命中 %d 条", sc.Stage, sc.Count, hit)
		}
	}
}

// stageRank 与 stageLabel 必须落在首页价值流同一张表上。
func TestStageRankAndLabelShareHomeTable(t *testing.T) {
	cases := []struct {
		status string
		want   int
	}{
		{"wait", stageIdxAccept},
		{"active", stageIdxClarify},
		{"clarified", stageIdxSchedule},
		{"developing", stageIdxDeveloping},
		{"testing", stageIdxTesting},
		{"waitacceptance", stageIdxWaitAcceptance},
		{"acceptanced", stageIdxAcceptanced},
		{"waitdeliver", stageIdxPublish},
		{"released", stageIdxReleased},
	}
	for _, tc := range cases {
		if got := stageRank("", tc.status); got != tc.want {
			t.Errorf("stageRank(%q) = %d，期望 %d", tc.status, got, tc.want)
		}
	}
	if got := stageLabel("", "unknown-status"); got != "暂无" {
		t.Errorf("未收录阶段应显示「暂无」，实际 %q", got)
	}
	if got := stageLabel("", "testing"); got != "联调测试" {
		t.Errorf("stageLabel(testing) = %q，期望与首页一致的「联调测试」", got)
	}
}
