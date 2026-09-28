// =============================================================================
// 文件: internal/module/po/boardderive.go
// 模块: PO 工作台
// 类型: action
// 职责: 需求看板纯派生逻辑（状态→阶段/动作标签、负责人与优先级规整、叶节点摘要）。
//       无数据库访问，独立成文件便于单测与复用（Rules §17.1 职责拆分）。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"
	"time"
)

// computeDemandSummary 遍历叶节点统计：待澄清/阻塞/超期（同 V1.3 顶栏快捷筛选口径；
// 前端会用 DOM 行二次精确计算，此处仅作后端口径参考）。
// 显式栈迭代替代递归（Rules：严禁递归）；超期按本地日期字符串比较，
// 避免按 UTC 截断的 Truncate(24h) 在本地 0~8 点窗口内错判昨天到期。
func (r *Repo) computeDemandSummary(out []*BoardDemandItem) BoardDemandSummary {
	s := BoardDemandSummary{}
	today := time.Now().Format("2006-01-02")
	stack := make([]*BoardDemandItem, 0, len(out))
	stack = append(stack, out...)
	for len(stack) > 0 {
		it := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if len(it.Children) > 0 {
			stack = append(stack, it.Children...)
			continue
		}
		// 仅叶节点（最细有效推进对象）进入统计，避免父节点重复累计
		s.Total++
		if strings.Contains(it.Stage, "受理") || strings.Contains(it.Stage, "澄清") {
			s.PendingReview++
		}
		if it.Status == "refuse" || it.Status == "hang" {
			s.Blocked++
		}
		if it.Deadline != "" && it.Status != "done" && it.Status != "released" && it.Deadline < today {
			s.Overdue++
		}
	}
	return s
}

// deriveStageFromStatus 状态映射 V1.3 五大阶段（不可拖拽，状态派生）。
func deriveStageFromStatus(s string) string {
	switch s {
	case "draft", "wait", "active", "refuse":
		return "受理/澄清"
	case "clarified":
		return "排期"
	case "developing":
		return "研发/提测"
	case "testing", "waitacceptance":
		return "联调/验收"
	case "acceptanced", "waitdeliver", "released":
		return "交付/评价"
	}
	return s
}

// deriveStoryStage 研需阶段由 zt_story.status/stage 派生。
func deriveStoryStage(status, stage string) string {
	switch status {
	case "draft", "wait", "active":
		if stage == "" || stage == "wait" {
			return "受理/澄清"
		}
	case "clarified", "planned", "projected", "designed", "designing":
		return "排期"
	case "developing", "developed":
		return "研发/提测"
	case "testing", "tested", "verified", "reviewing":
		return "联调/验收"
	case "delivering", "delivered", "releasing", "released":
		return "交付/评价"
	}
	// 兜底按 stage 字段
	switch stage {
	case "developing", "developed":
		return "研发/提测"
	case "testing", "tested", "verified":
		return "联调/验收"
	case "delivering", "delivered", "released":
		return "交付/评价"
	}
	return "受理/澄清"
}

// deriveStoryAction 研需行顶栏主操作：未进入执行且无任务为「去梳理」，其余一律「查看任务」。
func deriveStoryAction(status string, taskTotal int) string {
	if (status == "draft" || status == "wait" || status == "active") && taskTotal == 0 {
		return "去梳理"
	}
	return "查看任务"
}

// deriveActionLabel 顶部主操作按钮文本（V1.3 节点级操作）。
func deriveActionLabel(s string) string {
	switch s {
	case "draft", "wait", "refuse", "active":
		return "去梳理"
	case "clarified":
		return "去排期"
	case "developing", "testing", "waitacceptance":
		return "查看任务"
	case "acceptanced":
		return "跟进发布"
	}
	return "查看任务"
}

// reqOwner 需求负责人：优先指派人，回退创建人。
func reqOwner(account, assignedTo string) string {
	if strings.TrimSpace(assignedTo) != "" {
		return assignedTo
	}
	return account
}

// priOf 优先级字符串 → "P1" 形式（pri 可能为空串）。
func priOf(p string) string {
	p = strings.TrimSpace(p)
	if p != "" && p != "0" {
		return "P" + p
	}
	return ""
}
