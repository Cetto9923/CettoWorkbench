// =============================================================================
// 文件: internal/module/schedule/service_story_history.go
// 模块: 排期工作台
// 类型: action
// 职责: 独立研需日期编辑、业需排期编辑与任务编辑时对比旧值并写入禅道历史；任务编辑走禅道 PUT。
// 依赖: internal/model
//       internal/pkg/ztaction
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/ztaction"
)

func (s *Service) saveEditedStoryDates(
	ctx context.Context,
	txRepo *Repo,
	account string,
	storyID uint,
	productID uint,
	estimateLaunch string,
	developFinish string,
	testFinish string,
	verifyFinish string,
) error {
	snapshot, err := txRepo.FindStoryHistorySnapshot(ctx, storyID)
	if err != nil {
		return fmt.Errorf("load story %d: %w", storyID, err)
	}
	if err := txRepo.UpdateStory(ctx, storyID, map[string]interface{}{
		"estimateLaunch": nullableSchedulingDate(estimateLaunch),
		"developFinish":  nullableSchedulingDate(developFinish),
		"testFinish":     nullableSchedulingDate(testFinish),
		"verifyFinish":   nullableSchedulingDate(verifyFinish),
		"lastEditedBy":   account,
		"lastEditedDate": time.Now(),
	}); err != nil {
		return err
	}
	changes := []ztaction.Change{
		{Field: "estimateLaunch", Old: normalizeHistoryDate(snapshot.EstimateLaunch), New: normalizeHistoryDate(estimateLaunch)},
		{Field: "developFinish", Old: normalizeHistoryDate(snapshot.DevelopFinish), New: normalizeHistoryDate(developFinish)},
		{Field: "testFinish", Old: normalizeHistoryDate(snapshot.TestFinish), New: normalizeHistoryDate(testFinish)},
		{Field: "verifyFinish", Old: normalizeHistoryDate(snapshot.VerifyFinish), New: normalizeHistoryDate(verifyFinish)},
	}
	return txRepo.LogEditedHistory(ctx, "story", storyID, account, productID, 0, 0, changes)
}

// saveEditedTask 调用禅道 PUT /tasks/{id} 编辑任务。
// 维护任务弹窗会把未改动的已有任务也标成 edit；与库中旧值一致时不请求禅道。
func (s *Service) saveEditedTask(ctx context.Context, taskReq SaveSchedulingTask) error {
	if taskReq.ID == 0 {
		return fmt.Errorf("任务 ID 无效")
	}
	snapshot, err := s.repo.FindTaskHistorySnapshot(ctx, taskReq.ID)
	if err != nil {
		return fmt.Errorf("load task %d: %w", taskReq.ID, err)
	}
	if !schedulingTaskChanged(snapshot, taskReq) {
		return nil
	}
	if err := updateTask(ctx, s.ztAPI, taskReq.ID, buildZentaoTaskEditBody(taskReq)); err != nil {
		return fmt.Errorf("update task %d: %w", taskReq.ID, err)
	}
	return nil
}

// schedulingTaskChanged 对比禅道 PUT 会写入的字段。字段集合须与 buildZentaoTaskEditBody 一致。
func schedulingTaskChanged(snapshot *taskHistorySnapshot, taskReq SaveSchedulingTask) bool {
	if snapshot == nil {
		return true
	}
	if strings.TrimSpace(snapshot.Name) != strings.TrimSpace(taskReq.Name) {
		return true
	}
	if strings.TrimSpace(snapshot.Type) != strings.TrimSpace(taskReq.Type) {
		return true
	}
	if snapshot.Pri != normalizeTaskPriority(taskReq.Pri) {
		return true
	}
	if strings.TrimSpace(snapshot.AssignedTo) != strings.TrimSpace(taskReq.AssignedTo) {
		return true
	}
	if !sameTaskEstimate(snapshot.Estimate, taskReq.Estimate) {
		return true
	}
	if normalizeHistoryDate(snapshot.EstStarted) != normalizeHistoryDate(taskReq.EstStarted) {
		return true
	}
	if normalizeHistoryDate(snapshot.Deadline) != normalizeHistoryDate(taskReq.Deadline) {
		return true
	}
	return snapshot.Execution != taskReq.ExecutionID
}

func sameTaskEstimate(current, next float64) bool {
	diff := current - next
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.0005
}

func (s *Service) saveEditedDemandScheduling(
	ctx context.Context,
	txRepo *Repo,
	account string,
	demandID uint,
	mainSystemID uint,
	req *SaveSchedulingReq,
	estimateLaunch string,
	window *model.VersionWindow,
) error {
	snapshot, err := txRepo.FindDemandSchedulingSnapshot(ctx, demandID)
	if err != nil {
		return fmt.Errorf("load demand %d: %w", demandID, err)
	}
	oldWindowID, oldWindowName, err := txRepo.findDemandLevelWindow(ctx, demandID)
	if err != nil {
		return fmt.Errorf("load demand window %d: %w", demandID, err)
	}
	if err := txRepo.UpdateDemandScheduling(ctx, demandID, buildDemandSchedulingUpdates(req, account, estimateLaunch)); err != nil {
		return fmt.Errorf("update demand scheduling: %w", err)
	}

	newWindowName := ""
	var newWindowID uint
	if window != nil {
		newWindowID = uint(window.ID)
		newWindowName = strings.TrimSpace(window.Name)
	}
	if newWindowName == "" {
		newWindowName = strconv.FormatUint(uint64(newWindowID), 10)
	}
	if oldWindowID == 0 {
		oldWindowName = ""
	} else if strings.TrimSpace(oldWindowName) == "" {
		oldWindowName = strconv.FormatUint(uint64(oldWindowID), 10)
	} else {
		oldWindowName = strings.TrimSpace(oldWindowName)
	}
	changes := []ztaction.Change{
		{Field: "QD", Old: strings.TrimSpace(snapshot.QD), New: strings.TrimSpace(req.QD)},
		{Field: "RD", Old: strings.TrimSpace(snapshot.RD), New: strings.TrimSpace(req.Accepter)},
		{Field: "estimateLaunch", Old: normalizeHistoryDate(snapshot.EstimateLaunch), New: normalizeHistoryDate(estimateLaunch)},
		{Field: "developFinish", Old: normalizeHistoryDate(snapshot.DevelopFinish), New: normalizeHistoryDate(req.DevelopFinish)},
		{Field: "testFinish", Old: normalizeHistoryDate(snapshot.TestFinish), New: normalizeHistoryDate(req.TestFinish)},
		{Field: "verifyFinish", Old: normalizeHistoryDate(snapshot.VerifyFinish), New: normalizeHistoryDate(req.AcceptancedDate)},
	}
	if oldWindowID != newWindowID {
		changes = append(changes, ztaction.Change{Field: "versionWindow", Old: oldWindowName, New: newWindowName})
	}
	if err := txRepo.LogEditedRecord(ctx, ztaction.Record{
		ObjectType: "demand",
		ObjectID:   demandID,
		Product:    snapshot.Product,
		ProductID:  mainSystemID,
		Actor:      account,
	}, changes); err != nil {
		return fmt.Errorf("create demand action: %w", err)
	}
	return nil
}
