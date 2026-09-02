// =============================================================================
// 文件: internal/module/po/blockerclassify.go
// 模块: PO 工作台
// 类型: readonly
// 职责: 卡点等级/截止日/排序等纯函数。从 blocker.go 拆出,避免主 Service 文件
//       超过 500 行硬线。所有函数无副作用,易单测。
// 依赖: 无(只依赖标准库)
// =============================================================================

package po

import (
	"fmt"
	"sort"
	"time"
)

// levelOrder 等级排序优先级。
func levelOrder(l BlockerLevel) int {
	switch l {
	case BlockerLevelBlocked:
		return 0
	case BlockerLevelOverdue:
		return 1
	case BlockerLevelRisk:
		return 2
	case BlockerLevelCoord:
		return 3
	default:
		return 99
	}
}

func levelLabel(l BlockerLevel) string {
	switch l {
	case BlockerLevelBlocked:
		return "阻塞"
	case BlockerLevelOverdue:
		return "超期"
	case BlockerLevelRisk:
		return "风险"
	case BlockerLevelCoord:
		return "协调"
	default:
		return string(l)
	}
}

// pickOwner 从多个责任人员字段里选优(首个非空且不是空字符串)。本字段选取
// 顺序仅供数据回退,非业务权威规则——以 zt_demand.RD 作为 PO 场景下最直接的主研发负责人。
func pickOwner(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// dueMetrics 计算给定截止日期与 today 的关系。
// 返回字段:
//
//	dueAt: 原值(YYYY-MM-DD),空时返回 ""。
//	label: "今日已超 X 天"(逾期) / "今日"(恰好) / "距今 X 天"(未到) / ""。
//	isOverdue: true 表示已逾期。
func dueMetrics(due *time.Time, today time.Time) (dueAt, label string, isOverdue bool) {
	if due == nil {
		return "", "", false
	}
	dueDay := due.Truncate(24 * time.Hour)
	todayDay := today.Truncate(24 * time.Hour)
	dueAt = dueDay.Format("2006-01-02")
	delta := int(todayDay.Sub(dueDay).Hours() / 24)
	switch {
	case delta > 0:
		return dueAt, fmt.Sprintf("今日已超 %d 天", delta), true
	case delta < 0:
		return dueAt, fmt.Sprintf("距今 %d 天", -delta), false
	default:
		return dueAt, "今日", false
	}
}

// 选择合理的截止日:状态决定首选关键日;用 guardMinYear 过滤掉 0000-00-00 /
// 0001-01-01 等 MySQL 零日期误解析后的超古老占位。
//
// 联调 testing → testFinish
// waitacceptance → verifyFinish
// acceptanced → deliverDate
// 其他(developing / schedule / clarify)→ developFinish
//
// 若首选日期为空或为 MySQL 零日期,会回退到其余关键日;都无效则返 nil。
func chooseDeadline(status string, d *time.Time, tf, vf, dd *time.Time) *time.Time {
	primary := pickPrimary(status, d, tf, vf, dd)
	if validDate(primary) {
		return primary
	}
	// 回退顺序:先试其他关键日(同 status 不同字段),再 try 别的 status 路径
	for _, cand := range []*time.Time{d, tf, vf, dd} {
		if validDate(cand) {
			return cand
		}
	}
	return nil
}

// pickPrimary 根据状态挑出首选截止字段(不做有效性判断)。
func pickPrimary(status string, d *time.Time, tf, vf, dd *time.Time) *time.Time {
	switch status {
	case "testing":
		return tf
	case "waitacceptance":
		return vf
	case "acceptanced":
		return dd
	}
	return d
}

// validDate 过滤 MySQL 零日期伪值。MySQL DATE '0000-00-00' 经 parseTime=true 时会被
// 解析成 0001-01-01(远早于 2000),不能让 blocker 显示「今日已超 N 年」。
func validDate(t *time.Time) bool {
	if t == nil {
		return false
	}
	guardYear := 2000 // 业务上禅道需求早于 2000 的概率极低,未到这一年的视为脏值。
	return t.Year() >= guardYear
}

// isOwnAction 当前账号在该阶段是否为执行人/承接人。
// 简单判定:当前账号出现在 RD/assignedTo/QD/BRA 中任一字段即视为自身责任。
func isOwnAction(account string, rd, qd, bra, assigned string) bool {
	if account == "" {
		return false
	}
	for _, v := range []string{rd, qd, bra, assigned} {
		if v != "" && v == account {
			return true
		}
	}
	return false
}

// classifyDateBased 依据日期状态计算等级(替代旧版本按 status 静态映射)。
//
//	空关键日 → risk
//	关键日已逾期且当前账号不是 own → blocked
//	关键日已逾期且当前账号 own → overdue
//	关键日距 today ≤ 3 → risk
//	关键日未到且 own → coord
//	其余 → risk(兜底)
func classifyDateBased(due *time.Time, today time.Time, ownAction bool) BlockerLevel {
	if due == nil {
		return BlockerLevelRisk
	}
	delta := int(today.Truncate(24*time.Hour).Sub(due.Truncate(24*time.Hour)).Hours() / 24)
	switch {
	case delta > 0 && !ownAction:
		return BlockerLevelBlocked
	case delta > 0:
		return BlockerLevelOverdue
	case delta >= -3:
		return BlockerLevelRisk
	default:
		if ownAction {
			return BlockerLevelCoord
		}
		return BlockerLevelRisk
	}
}

// sortBlockers 排序:等级优先 → ownAction 优先 → stage/kind/id 字典序稳定。
func sortBlockers(items []BlockerDetail) {
	sort.SliceStable(items, func(i, j int) bool {
		li, lj := levelOrder(items[i].Level), levelOrder(items[j].Level)
		if li != lj {
			return li < lj
		}
		if items[i].IsOwnAction != items[j].IsOwnAction {
			return items[i].IsOwnAction
		}
		if items[i].Stage != items[j].Stage {
			return items[i].Stage < items[j].Stage
		}
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].ID < items[j].ID
	})
}
