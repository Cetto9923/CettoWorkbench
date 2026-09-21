// =============================================================================
// 文件: internal/module/schedule/service_story_history.go
// 模块: 排期工作台
// 类型: action
// 职责: 编辑业务需求排期、研发需求或任务时对比旧值并写入禅道历史。
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

func (s *Service) saveEditedStory(
	ctx context.Context,
	txRepo *Repo,
	account string,
	storyID uint,
	productID uint,
	estimateLaunch string,
	developFinish string,
	testFinish string,
	storyReq SaveSchedulingStory,
) error {
	snapshot, err := txRepo.FindStoryHistorySnapshot(ctx, storyID)
	if err != nil {
		return fmt.Errorf("load story %d: %w", storyID, err)
	}

	title := strings.TrimSpace(storyReq.Title)
	assignedTo := strings.TrimSpace(storyReq.AssignedTo)
	updates := map[string]interface{}{
		"title":          title,
		"assignedTo":     assignedTo,
		"product":        productID,
		"estimateLaunch": nullableSchedulingDate(estimateLaunch),
		"developFinish":  nullableSchedulingDate(developFinish),
		"testFinish":     nullableSchedulingDate(testFinish),
		"lastEditedBy":   account,
		"lastEditedDate": time.Now(),
	}
	// 对齐禅道 buildStoryForEdit：指派变化才刷新 assignedDate，且该字段不进 history。
	if assignedTo != strings.TrimSpace(snapshot.AssignedTo) {
		updates["assignedDate"] = time.Now()
	}
	if err := txRepo.UpdateStory(ctx, storyID, updates); err != nil {
		return fmt.Errorf("update story %d: %w", storyID, err)
	}

	changes := []ztaction.Change{
		{Field: "title", Old: strings.TrimSpace(snapshot.Title), New: title},
		{Field: "assignedTo", Old: strings.TrimSpace(snapshot.AssignedTo), New: assignedTo},
		{Field: "product", Old: strconv.FormatUint(uint64(snapshot.Product), 10), New: strconv.FormatUint(uint64(productID), 10)},
		{Field: "estimateLaunch", Old: normalizeHistoryDate(snapshot.EstimateLaunch), New: normalizeHistoryDate(estimateLaunch)},
		{Field: "developFinish", Old: normalizeHistoryDate(snapshot.DevelopFinish), New: normalizeHistoryDate(developFinish)},
		{Field: "testFinish", Old: normalizeHistoryDate(snapshot.TestFinish), New: normalizeHistoryDate(testFinish)},
	}
	if err := txRepo.LogEditedHistory(ctx, "story", storyID, account, productID, 0, 0, changes); err != nil {
		return fmt.Errorf("create story action: %w", err)
	}
	return nil
}

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

func (s *Service) saveEditedTask(
	ctx context.Context,
	txRepo *Repo,
	account string,
	productID uint,
	projectID uint,
	taskReq SaveSchedulingTask,
) error {
	snapshot, err := txRepo.FindTaskHistorySnapshot(ctx, taskReq.ID)
	if err != nil {
		return fmt.Errorf("load task %d: %w", taskReq.ID, err)
	}

	name := strings.TrimSpace(taskReq.Name)
	taskType := strings.TrimSpace(taskReq.Type)
	pri := normalizeTaskPriority(taskReq.Pri)
	assignedTo := strings.TrimSpace(taskReq.AssignedTo)
	estStarted := nullableDateValue(taskReq.EstStarted)
	deadline := nullableDateValue(taskReq.Deadline)
	updates := map[string]interface{}{
		"name":           name,
		"type":           taskType,
		"pri":            pri,
		"assignedTo":     assignedTo,
		"estimate":       taskReq.Estimate,
		"left":           taskReq.Estimate,
		"estStarted":     estStarted,
		"deadline":       deadline,
		"execution":      taskReq.ExecutionID,
		"project":        projectID,
		"lastEditedBy":   account,
		"lastEditedDate": time.Now(),
	}
	// 对齐禅道 buildTaskForEdit：指派变化才刷新 assignedDate，且该字段不进 history。
	if assignedTo != strings.TrimSpace(snapshot.AssignedTo) {
		updates["assignedDate"] = time.Now()
	}
	if err := txRepo.UpdateTask(ctx, taskReq.ID, updates); err != nil {
		return fmt.Errorf("update task %d: %w", taskReq.ID, err)
	}

	changes := []ztaction.Change{
		{Field: "name", Old: strings.TrimSpace(snapshot.Name), New: name},
		{Field: "type", Old: strings.TrimSpace(snapshot.Type), New: taskType},
		{Field: "pri", Old: strconv.Itoa(snapshot.Pri), New: strconv.Itoa(pri)},
		{Field: "assignedTo", Old: strings.TrimSpace(snapshot.AssignedTo), New: assignedTo},
		{Field: "estimate", Old: formatHistoryNumber(snapshot.Estimate), New: formatHistoryNumber(taskReq.Estimate)},
		{Field: "left", Old: formatHistoryNumber(snapshot.Left), New: formatHistoryNumber(taskReq.Estimate)},
		{Field: "estStarted", Old: normalizeHistoryDate(snapshot.EstStarted), New: normalizeHistoryDate(estStarted)},
		{Field: "deadline", Old: normalizeHistoryDate(snapshot.Deadline), New: normalizeHistoryDate(deadline)},
		{Field: "execution", Old: strconv.FormatUint(uint64(snapshot.Execution), 10), New: strconv.FormatUint(uint64(taskReq.ExecutionID), 10)},
		{Field: "project", Old: strconv.FormatUint(uint64(snapshot.Project), 10), New: strconv.FormatUint(uint64(projectID), 10)},
	}
	if err := txRepo.LogEditedHistory(ctx, "task", taskReq.ID, account, productID, projectID, taskReq.ExecutionID, changes); err != nil {
		return fmt.Errorf("create task action: %w", err)
	}
	return nil
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

func formatHistoryNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
