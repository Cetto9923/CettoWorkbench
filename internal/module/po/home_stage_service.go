package po

import (
	"context"
	"fmt"
	"math"

	"workbench/internal/model"
)

// countAllStageBreakdown assigns each demand/story to its first matching stage.
// This keeps the stage cards and the deduplicated "all" card on one partition.
func (s *Service) countAllStageBreakdown(ctx context.Context, account string) ([]ValueStreamStage, error) {
	rows, err := s.repo.countStageRefs(ctx, account)
	if err != nil {
		return nil, err
	}
	breakdown := make([]ValueStreamStage, len(valueStreamStages))
	for i, def := range valueStreamStages {
		breakdown[i] = ValueStreamStage{
			Label:           def.label,
			Status:          def.status,
			Valid:           true,
			AvgDurationText: "—",
		}
	}
	type durStat struct {
		totalDuration int64
		durationCount int64
	}
	stageDur := make([]durStat, len(valueStreamStages))
	var totalDemandDuration int64
	var totalDemandDurationCount int64

	for _, row := range rows {
		stage := &breakdown[row.StageIndex]
		stage.Count += row.Count
		if row.Kind == "demand" {
			stage.DemandCount += row.Count
			stageDur[row.StageIndex].totalDuration += row.TotalDuration
			stageDur[row.StageIndex].durationCount += row.DurationCount

			totalDemandDuration += row.TotalDuration
			totalDemandDurationCount += row.DurationCount
		} else {
			stage.StoryCount += row.Count
		}
	}

	for i := range breakdown {
		if breakdown[i].Status == "all" {
			continue
		}
		st := stageDur[i]
		if st.durationCount > 0 {
			avg := int(math.Round(float64(st.totalDuration) / float64(st.durationCount)))
			if avg < 1 {
				avg = 1
			}
			breakdown[i].AvgDurationDays = avg
			breakdown[i].AvgDurationText = fmt.Sprintf("均%d天", avg)
		} else {
			breakdown[i].AvgDurationDays = 0
			breakdown[i].AvgDurationText = "—"
		}
	}

	if len(breakdown) > 0 && breakdown[0].Status == "all" {
		if totalDemandDurationCount > 0 {
			avg := int(math.Round(float64(totalDemandDuration) / float64(totalDemandDurationCount)))
			if avg < 1 {
				avg = 1
			}
			breakdown[0].AvgDurationDays = avg
			breakdown[0].AvgDurationText = fmt.Sprintf("均%d天", avg)
		} else {
			breakdown[0].AvgDurationDays = 0
			breakdown[0].AvgDurationText = "—"
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
	return s.populateWorkItems(ctx, actor, refs, total, req.Page, req.PageSize, displayMap, req)
}
