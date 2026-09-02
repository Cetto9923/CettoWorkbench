// =============================================================================
// 文件: internal/module/po/blocker.go
// 模块: PO 工作台
// 类型: readonly
// 职责: 实现 PO 工作台首页"卡点快速响应"主 Service 方法。
//       数据契约:基于真实日期字段(developFinish / testFinish / verifyFinish /
//       deliverDate)与 today 的关系计算等级与 dueLabel;从真实责任人员字段
//       (RD / assignedTo / QD / BRA)选优填充 owner;不再依赖任何硬编码
//       status→等级表。
//       本文件只保留 Blocker Service 方法 + 内部计数辅助;常量与分类函数已
//       拆到 blockertypes.go / blockerclassify.go。
// 依赖: internal/model
//       internal/module/po/priority.go
//       internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

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
