// =============================================================================
// 文件: internal/module/po/version_follow_judge_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 发布判断三态与确定性规则的表驱动单测，覆盖三态命中与关键字段缺失。
// =============================================================================

package po

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func vfTestNow() time.Time {
	return time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
}

func vfDay(offset int) *time.Time {
	d := vfTestNow().AddDate(0, 0, offset)
	return &d
}

// vfJudgeCase 是三态推导的单条用例。
type vfJudgeCase struct {
	name      string
	raw       vfDemandRow
	wantState string
	wantEmpty bool
}

// vfJudgeCases 覆盖阻塞 3 条、风险 2 条、可按期 1 条、暂无判断 2 条。
var vfJudgeCases = []vfJudgeCase{
	{
		name:      "阻塞-已过计划提测日仍未提测",
		raw:       vfDemandRow{Stage: "wait", Status: "wait", Owner: "张三", SchedulePlanDate: vfDay(-5)},
		wantState: VFJudgementBlocked,
	},
	{
		name:      "阻塞-缺负责人",
		raw:       vfDemandRow{Stage: "wait", Status: "wait", SchedulePlanDate: vfDay(10)},
		wantState: VFJudgementBlocked,
	},
	{
		name: "阻塞-已进测试但验收标准为空",
		raw: vfDemandRow{
			Stage: "testing", Status: "testing", Owner: "李四",
			SchedulePlanDate: vfDay(10), Acceptance: "",
		},
		wantState: VFJudgementBlocked,
	},
	{
		name: "风险-已超期2天",
		raw: vfDemandRow{
			Stage: "developing", Status: "developing", Owner: "王五", Acceptance: "有标准",
			SchedulePlanDate: vfDay(20), Deadline: vfDay(-2),
		},
		wantState: VFJudgementRisk,
	},
	{
		name: "风险-距计划提测日2天且阶段落后",
		raw: vfDemandRow{
			Stage: "clarify", Status: "active", Owner: "赵六", Acceptance: "有标准",
			SchedulePlanDate: vfDay(2),
		},
		wantState: VFJudgementRisk,
	},
	{
		name: "可按期-计划日期内无异常",
		raw: vfDemandRow{
			Stage: "developing", Status: "developing", Owner: "钱七", Acceptance: "有标准",
			SchedulePlanDate: vfDay(15), Deadline: vfDay(20),
		},
		wantState: VFJudgementOnTrack,
	},
	{
		name:      "暂无判断-计划字段全缺",
		raw:       vfDemandRow{Stage: "wait", Status: "wait", Owner: "孙八", Acceptance: "有标准"},
		wantEmpty: true,
	},
	{
		name:      "暂无判断-仅有deadline为空且无plan",
		raw:       vfDemandRow{Stage: "testing", Status: "testing", Owner: "周九", Acceptance: "有标准"},
		wantEmpty: true,
	},
}

func TestDeriveJudgement(t *testing.T) {
	for _, tc := range vfJudgeCases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveJudgement(judgeCtx{Raw: tc.raw, Now: vfTestNow()})
			if got.Judgement != tc.wantState {
				t.Fatalf("判断 = %q，期望 %q（原因 %q）", got.Judgement, tc.wantState, got.Reason)
			}
			if tc.wantEmpty {
				if got.Reason != "" {
					t.Fatalf("暂无判断时原因应为空，实际 %q", got.Reason)
				}
				return
			}
			if got.Reason == "" {
				t.Fatalf("判断 %q 必须带一句原因，实际为空", got.Judgement)
			}
		})
	}
}

// vfRuleCase 是确定性规则的单条用例。
type vfRuleCase struct {
	name      string
	raw       vfDemandRow
	avgStage  float64
	wantRules []string
}

// vfRuleCases 覆盖四条规则各命中一次，外加全部不命中。
var vfRuleCases = []vfRuleCase{
	{
		name:      "超期未提测命中",
		raw:       vfDemandRow{ID: 1001, Stage: "wait", Status: "wait", Owner: "张三", SchedulePlanDate: vfDay(-3)},
		wantRules: []string{"超期未提测"},
	},
	{
		name:      "缺负责人命中",
		raw:       vfDemandRow{ID: 1002, Stage: "clarify", Status: "active"},
		wantRules: []string{"缺负责人"},
	},
	{
		name:      "验收标准为空命中",
		raw:       vfDemandRow{ID: 1003, Stage: "acceptanced", Status: "acceptanced", Owner: "李四"},
		wantRules: []string{"验收标准为空"},
	},
	{
		name:      "阶段落后于窗口平均命中",
		raw:       vfDemandRow{ID: 1004, Stage: "wait", Status: "wait", Owner: "王五", Acceptance: "有"},
		avgStage:  5,
		wantRules: []string{"阶段落后于同窗口平均"},
	},
	{
		name: "全部不命中",
		raw: vfDemandRow{
			ID: 1005, Stage: "developing", Status: "developing",
			Owner: "赵六", Acceptance: "有标准", SchedulePlanDate: vfDay(15),
		},
		avgStage:  3,
		wantRules: nil,
	},
}

func TestRunVFRules(t *testing.T) {
	for _, tc := range vfRuleCases {
		t.Run(tc.name, func(t *testing.T) {
			got := runVFRules(vfRule{Raw: tc.raw, Now: vfTestNow(), AvgStage: tc.avgStage})
			assertRuleFindings(t, got, tc.wantRules)
		})
	}
}

// assertRuleFindings 校验命中条数、规则名顺序与必填字段。
func assertRuleFindings(t *testing.T, got []VersionFollowAIFinding, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("命中数 = %d，期望 %d（%v）", len(got), len(want), got)
	}
	for i, name := range want {
		if got[i].RuleName != name {
			t.Fatalf("第 %d 条规则名 = %q，期望 %q", i, got[i].RuleName, name)
		}
		if got[i].Field == "" || got[i].TargetURL == "" {
			t.Fatalf("第 %d 条发现缺少字段或跳转对象：%+v", i, got[i])
		}
	}
}

// TestRunVFRulesPanicSilent 验证规则 panic 时静默、不阻断主流程。
func TestRunVFRulesPanicSilent(t *testing.T) {
	origin := vfRules
	defer func() { vfRules = origin }()
	vfRules = append([]vfRuleFunc{
		{name: "会 panic 的规则", fn: func(vfRule) []VersionFollowAIFinding { panic("boom") }},
	}, origin...)

	got := runVFRules(vfRule{Raw: vfDemandRow{ID: 1, Owner: ""}, Now: vfTestNow()})
	if len(got) == 0 {
		t.Fatal("panic 规则不应吞掉其它规则的结果")
	}
}

func TestAIPilotEnabledDefault(t *testing.T) {
	if aiPilotEnabled {
		t.Fatal("aiPilotEnabled 提交时必须为 false")
	}
}

// TestVFTestedStatusesLock locks the "post-test stage" statistic to zt_demand.status.
func TestVFTestedStatusesLock(t *testing.T) {
	want := map[string]bool{
		"testing": true, "delivered": true, "acceptanced": true,
		"waitacceptance": true, "waitdeliver": true, "released": true,
	}
	if len(vfTestedStatuses) != len(want) {
		t.Fatalf("vfTestedStatuses 长度 = %d，期望 %d：%v", len(vfTestedStatuses), len(want), vfTestedStatuses)
	}
	for _, s := range vfTestedStatuses {
		if !want[s] {
			t.Fatalf("vfTestedStatuses 含非测试后阶段的取值 %q", s)
		}
	}
}

// TestRepoSQLUsesEnumString 保证 repo 不再用数字 0 比较 enum 列。
// zt_demand.deleted 是 enum('0','1')，用数字 0 在 OceanBase 上不匹配，会让统计恒为 0。
func TestRepoSQLUsesEnumString(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位当前测试文件")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(self), "repo_version_follow.go"))
	if err != nil {
		t.Fatalf("读取 repo_version_follow.go 失败: %v", err)
	}
	text := string(src)
	if strings.Contains(text, "deleted = 0") {
		t.Fatal("repo 中仍存在 deleted = 0 的数字比较；zt_demand.deleted 是 enum，必须用 '0'")
	}
	if !strings.Contains(text, "deleted = '0'") {
		t.Fatal("repo 中未找到 deleted = '0' 的字符串比较")
	}
}
