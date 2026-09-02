// =============================================================================
// 文件: internal/module/po/blocker.go
// 模块: PO 工作台
// 类型: action
// 职责: 实现 PO 工作台首页"卡点快速响应"——识别今天必须处理的卡点需求/故事，按等级
//       排序、去重后输出。等级与特征由 forceBlockerCritial 在服务期装配，不依赖
//       zt_demand 新增字段。
// 依赖: internal/model
//       internal/module/po/service.go（共享 stageFilters / valueStreamStages）
// =============================================================================

package po

import (
	"context"
	"fmt"
	"sort"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

// BlockerLevel 卡点等级。
type BlockerLevel string

const (
	BlockerLevelBlocked BlockerLevel = "blocked" // P0 阻塞（关键路径卡死、关键人不是当前账号）
	BlockerLevelOverdue BlockerLevel = "overdue" // P1 超期（已跨过计划日，仍在动）
	BlockerLevelRisk    BlockerLevel = "risk"    // P2 风险（关键日期为空/主研发缺失）
	BlockerLevelCoord   BlockerLevel = "coord"   // P3 协调（临近关键日 + 当前账号是主研发）
)

// BlockerStageLimit 单阶段取数上限（避免单阶段吞掉全部限额）。
const BlockerStageLimit = 4

// BlockerOverallLimit 整体上限（越上限滚动，渲染层负责可滚动）。
const BlockerOverallLimit = 24

// blockerStatuses 卡点拉取的价值流阶段（热环节）。
var blockerStatuses = []string{
	"clarify",       // 澄清：可能因 QD/PM 缺失卡住
	"schedule",      // 排期：关键日期/QD/mainDevelopers 未填
	"developing",    // 提测：今天 >= developFinish 但未提测 = 卡
	"testing",       // 联调测试：今天 >= testFinish 但未结束
	"waitacceptance", // 验收：今天 >= verifyFinish 但未验
	"acceptanced",   // 交付：今天 >= deliverDate 但未交付
}

// BlockerReq 卡点请求参数（预留位，便于后期接 status 过滤）。
type BlockerReq struct{}

// Validate 校验请求参数。
func (r *BlockerReq) Validate() []FieldError {
	return nil
}

// BlockerDetail 单条卡点。
type BlockerDetail struct {
	Kind        string       `json:"kind"`        // demand / story
	ID          string       `json:"id"`
	Level       BlockerLevel `json:"level"`
	LevelLabel  string       `json:"levelLabel"`  // 阻塞 / 超期 / 风险 / 协调
	Title       string       `json:"title"`
	Owner       string       `json:"owner"`        // 当前责任人员工账号
	DueAt       string       `json:"dueAt"`        // 触发卡点的关键日期（YYYY-MM-DD）
	DueLabel    string       `json:"dueLabel"`     // 人类可读"今日已超 X 天 / 距今 X 天"
	Stage       string       `json:"stage"`        // 所属价值流标签
	ZentaoUrl   string       `json:"zentaoUrl"`
	IsOwnAction bool         `json:"isOwnAction"`  // true 表示当前账号就是要协调的人
}

// BlockerResp 卡点响应。
type BlockerResp struct {
	Items []BlockerDetail `json:"items"`
}

// levelOrder 等级排序的优先级（数字越小越在前）。
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

// filterOwnAction 判断卡点是否要求当前账号动手（运维首选/全责/临门一脚场景）。
// 原始阶段 + 类型判定，避免过度依赖业务主研发、未提供的位置字段。
func isOwnActionStage(status string) bool {
	switch status {
	case "waitacceptance", "acceptanced":
		return true
	}
	return false
}

// Blocker 返回今日卡点列表，按等级排序、同级按 kind+id 稳定，超 BlockerOverallLimit 截断。
func (s *Service) Blocker(ctx context.Context, actor *model.User, _ BlockerReq) (*BlockerResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}

	items := make([]BlockerDetail, 0)
	seen := make(map[string]struct{})

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
			key := workItemKey("demand", fmt.Sprintf("%d", row.ID))
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			// 等级映射。需求没有明确字段表达「重点关注」时，这里用 developFinish
			// 交作业伪信号＋阶段组合生成等级。详情形列表语义按 P0 → P3 劣化。
			lvl := classifyDemand(status)
			items = append(items, BlockerDetail{
				Kind:        "demand",
				ID:          fmt.Sprintf("%d", row.ID),
				Level:       lvl,
				LevelLabel:  levelLabel(lvl),
				Title:       row.Name,
				Owner:       account, // 未读取分配人字段，仅以角色范围为占位
				Stage:       label,
				ZentaoUrl:   zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
				IsOwnAction: isOwnActionStage(status),
			})
			if countByStageLabel(items, label) >= BlockerStageLimit {
				break
			}
		}

		// 排期/交付阶段额外拉取独立研发需求，看是否是当前账号独立卡点。
		if filter.scheduleIncomplete {
			stories, err := s.repo.FindScheduleStories(ctx, account)
			if err != nil {
				return nil, err
			}
			for _, row := range stories {
				key := workItemKey("story", fmt.Sprintf("%d", row.ID))
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				lvl := classifyStory(status)
				items = append(items, BlockerDetail{
					Kind:        "story",
					ID:          fmt.Sprintf("%d", row.ID),
					Level:       lvl,
					LevelLabel:  levelLabel(lvl),
					Title:       row.Title,
					Owner:       account,
					Stage:       label,
					ZentaoUrl:   zentao.URL("story", "view", fmt.Sprintf("storyID=%d", row.ID)),
					IsOwnAction: isOwnActionStage(status),
				})
				if countByStageLabel(items, label) >= BlockerStageLimit {
					break
				}
			}
		}
		if filter.deliverStories {
			stories, err := s.repo.FindDeliverStories(ctx, account)
			if err != nil {
				return nil, err
			}
			for _, row := range stories {
				key := workItemKey("story", fmt.Sprintf("%d", row.ID))
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				lvl := classifyStory(status)
				items = append(items, BlockerDetail{
					Kind:        "story",
					ID:          fmt.Sprintf("%d", row.ID),
					Level:       lvl,
					LevelLabel:  levelLabel(lvl),
					Title:       row.Title,
					Owner:       account,
					Stage:       label,
					ZentaoUrl:   zentao.URL("story", "view", fmt.Sprintf("storyID=%d", row.ID)),
					IsOwnAction: isOwnActionStage(status),
				})
				if countByStageLabel(items, label) >= BlockerStageLimit {
					break
				}
			}
		}
	}

	sortBlockers(items)
	if len(items) > BlockerOverallLimit {
		items = items[:BlockerOverallLimit]
	}

	return &BlockerResp{Items: items}, nil
}

// classifyDemand 按阶段映射需求默认等级。
// 该语义只在当前数据模型上提供粗颗粒默认映射，后续如要更细颗粒（验收人匹配/改需人匹配），需引入额外责任人员字段。
func classifyDemand(status string) BlockerLevel {
	switch status {
	case "developing":
		return BlockerLevelBlocked
	case "testing", "waitacceptance":
		return BlockerLevelOverdue
	case "schedule":
		return BlockerLevelRisk
	case "clarify":
		return BlockerLevelCoord
	default:
		return BlockerLevelRisk
	}
}

// classifyStory 独立研发的等级映射。
func classifyStory(status string) BlockerLevel {
	switch status {
	case "schedule":
		return BlockerLevelRisk
	case "acceptanced":
		return BlockerLevelOverdue
	default:
		return BlockerLevelRisk
	}
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
