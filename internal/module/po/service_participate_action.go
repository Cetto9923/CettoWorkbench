package po

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
)

// prepareParticipateStoryActions 批量准备参与业务需求的研发需求排期动作。
func (s *Service) prepareParticipateStoryActions(ctx context.Context, actor *model.User, refs []itemRef, req DemandsReq, storyActions map[uint]primaryaction.PrimaryAction) (map[int]int, error) {
	result := make(map[int]int)
	if !strings.EqualFold(strings.TrimSpace(req.Relation), "participate") {
		return result, nil
	}
	demandIDs := make([]int, 0)
	for _, ref := range refs {
		if ref.kind == "demand" && ref.stageStatus == "schedule" {
			demandIDs = append(demandIDs, ref.id)
		}
	}
	result, err := s.repo.FindPrimaryStoryIDsByDemandIDs(ctx, demandIDs)
	if err != nil {
		return nil, err
	}
	storyIDs := make([]uint, 0, len(result))
	for _, id := range result {
		storyIDs = append(storyIDs, uint(id))
	}
	actions, err := s.DeriveStoryPrimaryActions(ctx, actor, storyIDs, false)
	if err != nil {
		return nil, err
	}
	for id, action := range actions {
		storyActions[id] = action
	}
	return result, nil
}

func replaceParticipateScheduleAction(current *primaryaction.PrimaryAction, storyID int, storyActions map[uint]primaryaction.PrimaryAction) *primaryaction.PrimaryAction {
	if storyID <= 0 || current == nil || current.Key != string(primaryaction.KeySchedule) {
		return current
	}
	action := storyActions[uint(storyID)]
	if action.Key != string(primaryaction.KeySchedule) {
		action = primaryaction.DisabledWithReason(string(primaryaction.KeySchedule), "排期", string(primaryaction.KindSchedule), primaryaction.ScheduleURL(uint(storyID), primaryaction.ObjectStory), "对应研发需求当前不可排期")
	}
	return &action
}
