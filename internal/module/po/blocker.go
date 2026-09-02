// =============================================================================
// 文件: internal/module/po/blocker.go
// 模块: PO 工作台
// 类型: readonly
// 职责: 实现 PO 工作台首页"卡点快速响应"——基于真实日期字段（developFinish /
//       testFinish / verifyFinish / deliverDate）与 today 的关系计算等级与
//       dueLabel；从真实责任人员字段（RD / assignedTo / QD / BRA / assignedTo）
//       选优填充 owner；不再依赖任何硬编码 status→等级表。
// 依赖: internal/model
//       internal/module/po/priority.go
// =============================================================================

package po

import (
	"context"
	"fmt"
	"sort"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

// BlockerLevel 卡点等级。
type BlockerLevel string

const (
	BlockerLevelBlocked BlockerLevel = "blocked" // 关键路径已逾期且核心人未到岗（最重）
	BlockerLevelOverdue BlockerLevel = "overdue" // 已跨过截止日（任一关键日 ≤ today）
	BlockerLevelRisk    BlockerLevel = "risk"    // 关键日缺失或临近（≤ today+3）
	BlockerLevelCoord   BlockerLevel = "coord"   // 当前账号是主责任人，但日期未达
)

// BlockerStageLimit 单阶段取数上限（避免单阶段吞掉全部限额）。
const BlockerStageLimit = 4

// BlockerOverallLimit 整体上限（越上限滚动，渲染层负责可滚动）。
const BlockerOverallLimit = 24

// blockerStatuses 卡点拉取的价值流阶段（热环节）。
// 与现有 service.go 的 mysqlStageFilters key 对齐；新增 waitdeliver 是真实业务
// 状态但 PO 当前 filters 未覆盖，因此 blockerStatuses 不包含它。
var blockerStatuses = []string{
	"clarify",        // 澄清
	"schedule",       // 排期
	"developing",     // 提测
	"testing",        // 联调测试
	"waitacceptance", // 验收
	"acceptanced",    // 交付
}

// blockerReq 卡点请求（预留扩展）。
type BlockerReq struct{}

// Validate 校验（无入参）。
func (r *BlockerReq) Validate() []FieldError {
	return nil
}

// BlockerDetail 单条卡点。
type BlockerDetail struct {
	Kind        string       `json:"kind"` // demand / story
	ID          string       `json:"id"`
	Level       BlockerLevel `json:"level"`
	LevelLabel  string       `json:"levelLabel"`
	Title       string       `json:"title"`
	Owner       string       `json:"owner"`
	DueAt       string       `json:"dueAt"`    // YYYY-MM-DD
	DueLabel    string       `json:"dueLabel"` // "今日已超 3 天" / "距今 2 天" / "今日"
	Stage       string       `json:"stage"`
	ZentaoUrl   string       `json:"zentaoUrl"`
	IsOwnAction bool         `json:"isOwnAction"`
}

// BlockerResp 卡点响应。
type BlockerResp struct {
	Items []BlockerDetail `json:"items"`
}

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

// pickOwner 从多个责任人员字段里选优（首个非空且不是空字符串）。本字段选取
// 顺序仅供数据回退，非业务权威规则——以 zt_demand.RD 作为 PO 场景下最直接的主研发负责人。
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
//	dueAt: 原值（YYYY-MM-DD），空时返回 ""。
//	label: "今日已超 X 天"（逾期）/ "今日"（恰好） / "距今 X 天"（未到） / ""。
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

// 选择合理的截止日：状态决定首选关键日；用 guardMinYear 过滤掉 0000-00-00 /
// 0001-01-01 等 MySQL 零日期误解析后的超古老占位。
//
// 联调 testing → testFinish
// waitacceptance → verifyFinish
// acceptanced → deliverDate
// 其他（developing / schedule / clarify）→ developFinish
//
// 若首选日期为空或为 MySQL 零日期，会回退到其余关键日；都无效则返 nil。
func chooseDeadline(status string, d *time.Time, tf, vf, dd *time.Time) *time.Time {
	primary := pickPrimary(status, d, tf, vf, dd)
	if validDate(primary) {
		return primary
	}
	// 回退顺序：先试其他关键日（同 status 不同字段），再 try 别的 status 路径
	for _, cand := range []*time.Time{d, tf, vf, dd} {
		if validDate(cand) {
			return cand
		}
	}
	return nil
}

// pickPrimary 根据状态挑出首选截止字段（不做有效性判断）。
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
// 解析成 0001-01-01（远早于 2000），不能让 blocker 显示「今日已超 N 年」。
func validDate(t *time.Time) bool {
	if t == nil {
		return false
	}
	guardYear := 2000 // 业务上禅道需求早于 2000 的概率极低，未到这一年的视为脏值。
	return t.Year() >= guardYear
}

// isOwnAction 当前账号在该阶段是否为执行人/承接人。
// 简单判定：当前账号出现在 RD/assignedTo/QD/BRA 中任一字段即视为自身责任。
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

// classifyDateBased 依据日期状态计算等级（替代旧版本按 status 静态映射）。
//
//	空关键日 → risk
//	关键日已逾期且当前账号不是 own → blocked
//	关键日已逾期且当前账号 own → overdue
//	关键日距 today ≤ 3 → risk
//	关键日未到且 own → coord
//	其余 → risk（兜底）
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

// sortBlockers 排序：等级优先 → ownAction 优先 → stage/kind/id 字典序稳定。
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

// Blocker 返回今日卡点列表。所有等级/责任人/due 字段均从真实数据计算。
func (s *Service) Blocker(ctx context.Context, actor *model.User, _ BlockerReq) (*BlockerResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	today := time.Now()

	items := make([]BlockerDetail, 0)
	seen := make(map[string]struct{})

	appendDemand := func(row DemandRow, stageLabel, status string) {
		key := workItemKey("demand", fmt.Sprintf("%d", row.ID))
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}

		owner := pickOwner(row.RD, row.AssignedTo, row.QD, row.BRA)
		ownAction := isOwnAction(account, row.RD, row.AssignedTo, row.QD, row.BRA)
		due := chooseDeadline(status, row.DevelopFinish, row.TestFinish, row.VerifyFinish, row.DeliverDate)
		level := classifyDateBased(due, today, ownAction)
		dueAt, dueLabel, _ := dueMetrics(due, today)

		items = append(items, BlockerDetail{
			Kind:        "demand",
			ID:          fmt.Sprintf("%d", row.ID),
			Level:       level,
			LevelLabel:  levelLabel(level),
			Title:       row.Name,
			Owner:       owner,
			DueAt:       dueAt,
			DueLabel:    dueLabel,
			Stage:       stageLabel,
			ZentaoUrl:   zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
			IsOwnAction: ownAction,
		})
	}
	appendStory := func(row StoryRow, stageLabel, status string) {
		key := workItemKey("story", fmt.Sprintf("%d", row.ID))
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}

		owner := pickOwner(row.AssignedTo)
		ownAction := account != "" && row.AssignedTo != "" && row.AssignedTo == account
		due := chooseDeadline(status, row.DevelopFinish, row.TestFinish, row.VerifyFinish, row.DeliverDate)
		level := classifyDateBased(due, today, ownAction)
		dueAt, dueLabel, _ := dueMetrics(due, today)

		items = append(items, BlockerDetail{
			Kind:        "story",
			ID:          fmt.Sprintf("%d", row.ID),
			Level:       level,
			LevelLabel:  levelLabel(level),
			Title:       row.Title,
			Owner:       owner,
			DueAt:       dueAt,
			DueLabel:    dueLabel,
			Stage:       stageLabel,
			ZentaoUrl:   zentao.URL("story", "view", fmt.Sprintf("storyID=%d", row.ID)),
			IsOwnAction: ownAction,
		})
	}

	for _, status := range blockerStatuses {
		filter, ok := mysqlStageFilters[status]
		if !ok {
			continue
		}
		label := valueStreamLabelForStatus(status)

		demands, err := s.repo.FindRoleDemands(ctx, account, filter)
		if err != nil {
			return nil, err
		}
		for _, row := range demands {
			appendDemand(row, label, status)
		}
		if countByStageLabel(items, label) >= BlockerStageLimit {
			continue
		}

		if filter.scheduleIncomplete {
			stories, err := s.repo.FindScheduleStories(ctx, account)
			if err != nil {
				return nil, err
			}
			for _, row := range stories {
				appendStory(row, label, status)
			}
		}
		if filter.deliverStories {
			stories, err := s.repo.FindDeliverStories(ctx, account)
			if err != nil {
				return nil, err
			}
			for _, row := range stories {
				appendStory(row, label, status)
			}
		}
	}

	sortBlockers(items)
	if len(items) > BlockerOverallLimit {
		items = items[:BlockerOverallLimit]
	}

	return &BlockerResp{Items: items}, nil
}

func countByStageLabel(items []BlockerDetail, label string) int {
	n := 0
	for _, it := range items {
		if it.Stage == label {
			n++
		}
	}
	return n
}
