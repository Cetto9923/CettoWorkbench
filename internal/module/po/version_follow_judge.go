// =============================================================================
// 文件: internal/module/po/version_follow_judge.go
// 模块: PO 工作台
// 类型: action
// 职责: 发布判断三态的确定性推导。三态是推导结果，不入库、不新增字段。
// =============================================================================

package po

import (
	"strconv"
	"time"
)

// judgeCtx 是推导三态所需的最小输入。
type judgeCtx struct {
	Raw      vfDemandRow
	Now      time.Time
	AvgStage float64
}

// vfJudgementResult 是一次推导的结果。
type vfJudgementResult struct {
	Judgement string
	Reason    string
}

// 风险窗口天数：距计划日 3 天内算风险。
const vfRiskWindowDays = 3

// deriveJudgement 推导发布判断三态。
// 阻塞：已过计划提测日仍未提测；或缺负责人；或验收标准为空却已进入测试及之后阶段。
// 风险：距计划日 3 天内且阶段落后于计划；或已超期不足 3 天。
// 可按期：其余情况。
// 关键字段缺失（deadline 与 schedulePlanDate 均为空）：返回空判断，由页面显示「暂无判断」。
func deriveJudgement(c judgeCtx) vfJudgementResult {
	if c.Raw.Deadline == nil && c.Raw.SchedulePlanDate == nil {
		return vfJudgementResult{}
	}
	if r := judgeBlocked(c); r.Judgement != "" {
		return r
	}
	if r := judgeRisk(c); r.Judgement != "" {
		return r
	}
	return vfJudgementResult{
		Judgement: VFJudgementOnTrack,
		Reason:    "计划日期内，当前阶段未见阻塞或超期",
	}
}

// judgeBlocked 判定阻塞态；不满足返回空结果。
func judgeBlocked(c judgeCtx) vfJudgementResult {
	if c.Raw.SchedulePlanDate != nil && dayDiff(c.Now, *c.Raw.SchedulePlanDate) < 0 &&
		stageRank(c.Raw.Stage, c.Raw.Status) < stageIdxDeveloping {
		return vfJudgementResult{
			Judgement: VFJudgementBlocked,
			Reason: "计划提测日 " + c.Raw.SchedulePlanDate.Format("2006-01-02") + " 已过，当前仍在" +
				stageLabel(c.Raw.Stage, c.Raw.Status) + "阶段，未提测",
		}
	}
	if c.Raw.Owner == "" {
		return vfJudgementResult{
			Judgement: VFJudgementBlocked,
			Reason:    "负责人（assignedTo）为空，无人推进",
		}
	}
	if c.Raw.Acceptance == "" && stageRank(c.Raw.Stage, c.Raw.Status) >= stageIdxTesting {
		return vfJudgementResult{
			Judgement: VFJudgementBlocked,
			Reason:    "已进入" + stageLabel(c.Raw.Stage, c.Raw.Status) + "阶段，但验收标准为空",
		}
	}
	return vfJudgementResult{}
}

// judgeRisk 判定风险态；不满足返回空结果。
func judgeRisk(c judgeCtx) vfJudgementResult {
	// 已超期不足 3 天。
	if c.Raw.Deadline != nil {
		over := dayDiff(*c.Raw.Deadline, c.Now)
		if over > 0 && over < vfRiskWindowDays {
			return vfJudgementResult{
				Judgement: VFJudgementRisk,
				Reason:    "已超期 " + strconv.Itoa(over) + " 天（截止 " + c.Raw.Deadline.Format("2006-01-02") + "）",
			}
		}
	}
	// 距计划日 3 天内，且阶段落后于计划。
	if c.Raw.SchedulePlanDate != nil {
		days := dayDiff(c.Now, *c.Raw.SchedulePlanDate)
		if days >= 0 && days < vfRiskWindowDays && stageRank(c.Raw.Stage, c.Raw.Status) < stageIdxDeveloping {
			return vfJudgementResult{
				Judgement: VFJudgementRisk,
				Reason: "距计划提测日 " + strconv.Itoa(days) + " 天，当前仍在" +
					stageLabel(c.Raw.Stage, c.Raw.Status) + "阶段",
			}
		}
	}
	return vfJudgementResult{}
}
