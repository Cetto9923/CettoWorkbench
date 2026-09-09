package po

import (
	"context"
	"workbench/internal/model"
)

// countAllStageUniq 各阶段只查 ID，按 kind+id 去重后返回业需/研需数量（与 listAllStageDemands 并集语义一致）。
func (s *Service) countAllStageUniq(ctx context.Context, account string) (demandSum, storySum int64, err error) {
	breakdown, err := s.countAllStageBreakdown(ctx, account)
	if err != nil {
		return 0, 0, err
	}
	for _, stage := range breakdown {
		demandSum += stage.DemandCount
		storySum += stage.StoryCount
	}
	return demandSum, storySum, nil
}

// countAllStageBreakdown assigns each demand/story to its first matching stage.
// This keeps the stage cards and the deduplicated "all" card on one partition.
func (s *Service) countAllStageBreakdown(ctx context.Context, account string) ([]ValueStreamStage, error) {
	rows, err := s.repo.countStageRefs(ctx, account)
	if err != nil {
		return nil, err
	}
	breakdown := make([]ValueStreamStage, len(valueStreamStages))
	for i, def := range valueStreamStages {
		breakdown[i] = ValueStreamStage{Label: def.label, Status: def.status, Valid: true}
	}
	for _, row := range rows {
		stage := &breakdown[row.StageIndex]
		stage.Count += row.Count
		if row.Kind == "demand" {
			stage.DemandCount += row.Count
		} else {
			stage.StoryCount += row.Count
		}
	}
	return breakdown, nil
}

type itemRef struct {
	kind        string
	id          int
	stageStatus string
	actionRank  int
}

// listAllStageDemands 「全部」列表 = 其余各阶段列表按阶段顺序拼接，按 kind+id 去重（保留首次出现），并分页返回。
func (s *Service) listAllStageDemands(ctx context.Context, actor *model.User, req DemandsReq, displayMap map[string]string) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	refs, total, err := s.repo.FindAllStageRefsPaged(ctx, account, req)
	if err != nil {
		return nil, err
	}
	return s.populateWorkItems(ctx, actor, refs, total, req.Page, req.PageSize, displayMap)
}
