// =============================================================================
// 文件: internal/module/po/version_follow_rules.go
// 模块: PO 工作台
// 类型: action
// 职责: 版本跟进「AI 发现」的确定性规则集。第一期不接大模型，规则命中即结论；
//       以后接大模型只是新增一个来源，页面与本文件接口不变。
// =============================================================================

package po

import (
	"fmt"
	"time"

	"workbench/internal/pkg/demandstage"
)

// vfRule 是一条确定性规则的输入上下文。
type vfRule struct {
	Item     VersionFollowItemResp
	Raw      vfDemandRow
	Now      time.Time
	AvgStage float64
	WindowID uint64
}

// vfRuleFunc 单条规则：命中返回 findings，未命中返回 nil。name 为规则展示名。
type vfRuleFunc struct {
	name string
	fn   func(vfRule) []VersionFollowAIFinding
}

// vfRules 是全部规则；顺序即展示顺序。
var vfRules = []vfRuleFunc{
	{name: "超期未提测", fn: ruleOverdueNotSubmittedTest},
	{name: "验收标准为空", fn: ruleAcceptanceCriteriaEmpty},
	{name: "缺负责人", fn: ruleMissingOwner},
	{name: "阶段落后于同窗口平均", fn: ruleStageBehindWindowAverage},
}

// runVFRules 依次跑全部规则。任一条规则 panic 都被隔离，不影响其它规则的结果。
func runVFRules(r vfRule) []VersionFollowAIFinding {
	var findings []VersionFollowAIFinding
	for _, rule := range vfRules {
		if rule.fn == nil {
			continue
		}
		for _, f := range runRuleSafely(rule, r) {
			if f.RuleName == "" {
				f.RuleName = rule.name
			}
			findings = append(findings, f)
		}
	}
	return findings
}

// runRuleSafely 单独执行一条规则；panic 时返回 nil（静默），不向上冒泡。
func runRuleSafely(rule vfRuleFunc, r vfRule) (out []VersionFollowAIFinding) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	return rule.fn(r)
}

// ruleOverdueNotSubmittedTest：已过计划提测日仍未到测试/提测后阶段。
func ruleOverdueNotSubmittedTest(r vfRule) []VersionFollowAIFinding {
	if r.Raw.SchedulePlanDate == nil {
		return nil
	}
	if dayDiff(r.Now, *r.Raw.SchedulePlanDate) >= 0 {
		return nil
	}
	if stageRank(r.Raw.Stage, r.Raw.Status) >= stageIdxDeveloping {
		return nil
	}
	return []VersionFollowAIFinding{{
		RuleName:  "",
		Field:     "计划提测日",
		Value:     r.Raw.SchedulePlanDate.Format("2006-01-02"),
		TargetURL: fmt.Sprintf("/demands/%d", r.Raw.ID),
	}}
}

// ruleAcceptanceCriteriaEmpty：已进入验收阶段但验收标准为空。
func ruleAcceptanceCriteriaEmpty(r vfRule) []VersionFollowAIFinding {
	if stageRank(r.Raw.Stage, r.Raw.Status) < stageIdxWaitAcceptance {
		return nil
	}
	if r.Raw.Acceptance != "" {
		return nil
	}
	return []VersionFollowAIFinding{{
		RuleName:  "",
		Field:     "验收标准",
		Value:     "（空）",
		TargetURL: fmt.Sprintf("/demands/%d", r.Raw.ID),
	}}
}

// ruleMissingOwner：已有负责人字段但为空。
func ruleMissingOwner(r vfRule) []VersionFollowAIFinding {
	if r.Raw.Owner != "" {
		return nil
	}
	return []VersionFollowAIFinding{{
		RuleName:  "",
		Field:     "负责人",
		Value:     "（空）",
		TargetURL: fmt.Sprintf("/demands/%d", r.Raw.ID),
	}}
}

// ruleStageBehindWindowAverage：阶段排名明显落后于同窗口平均。
func ruleStageBehindWindowAverage(r vfRule) []VersionFollowAIFinding {
	if r.AvgStage <= 0 {
		return nil
	}
	self := float64(stageRank(r.Raw.Stage, r.Raw.Status))
	if self >= r.AvgStage-1 {
		return nil
	}
	return []VersionFollowAIFinding{{
		RuleName:  "",
		Field:     "当前阶段",
		Value:     fmt.Sprintf("%s（窗口平均 %.1f 阶段）", stageLabel(r.Raw.Stage, r.Raw.Status), r.AvgStage),
		TargetURL: fmt.Sprintf("/demands/%d", r.Raw.ID),
	}}
}

// stageRank 返回阶段在首页九阶段中的序号；未知阶段按 99 计（视为最靠后）。
// 复用 demandstage.Map 做归一，不自造第二套映射。
func stageRank(stage, status string) int {
	code := demandstage.Map(stage, status)
	for i, def := range valueStreamStages[1:] {
		if def.status == code {
			return i
		}
	}
	return 99
}

// stageLabel 返回阶段中文名，取自首页价值流阶段表；未收录的阶段显示「暂无」。
func stageLabel(stage, status string) string {
	if name := valueStreamLabel(demandstage.Map(stage, status)); name != "未知" {
		return name
	}
	return "暂无"
}
