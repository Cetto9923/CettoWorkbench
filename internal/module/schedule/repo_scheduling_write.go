// =============================================================================
// 文件: internal/module/schedule/repo_scheduling_write.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化「确认并同步」写库操作。
// 依赖: internal/model
//       internal/model/zentao
//       internal/pkg/ztaction
//       internal/module/schedule/form.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
	ztmodel "workbench/internal/model/zentao"
	"workbench/internal/pkg/ztaction"
)

// ZtStoryInsert 禅道 zt_story 写入字段。
type ZtStoryInsert struct {
	Product                 uint
	Title                   string
	AssignedTo              string
	Estimate                float64
	EstimateLaunch          string // 默认取排期弹窗预计上线（窗口 releaseDate）
	DevelopFinish           string // 弹窗提测/开发 → zt_story.developFinish
	TestFinish              string // 弹窗测试完成 → zt_story.testFinish
	FromDemand              uint
	IsMainSystemAssociation string
	OpenedBy                string
}

// ZtStorySpec 禅道 zt_storyspec 写入字段。
type ZtStorySpec struct {
	Story   uint
	Version int
	Title   string
	Spec    string
}

// ZtTaskInsert 禅道 zt_task 写入字段。
type ZtTaskInsert struct {
	Name       string
	Type       string
	Pri        int
	Story      uint
	Project    uint
	Execution  uint
	AssignedTo string
	Estimate   float64
	EstStarted string
	Deadline   string
	OpenedBy   string
}

// ZtTaskSpec 禅道 zt_taskspec 写入字段。
type ZtTaskSpec struct {
	Task       uint
	Version    int
	Name       string
	EstStarted string
	Deadline   string
}

// FindWindowProductPlan 查窗口下某产品的关联计划。
func (r *Repo) FindWindowProductPlan(ctx context.Context, windowID uint, productID uint) (*model.VersionWindowProduct, error) {
	if windowID == 0 || productID == 0 {
		return nil, nil
	}
	var row model.VersionWindowProduct
	err := r.db.WithContext(ctx).
		Where("versionWindow = ? AND product = ?", uint64(windowID), productID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetProjectIDByExecution 从 executionID 沿 parent 链推导 projectID。
func (r *Repo) GetProjectIDByExecution(ctx context.Context, executionID uint) (uint, error) {
	if executionID == 0 {
		return 0, errors.New("执行 ID 无效")
	}
	current := executionID
	for i := 0; i < 20; i++ {
		var row struct {
			ID     uint   `gorm:"column:id"`
			Parent uint   `gorm:"column:parent"`
			Type   string `gorm:"column:type"`
		}
		err := r.db.WithContext(ctx).
			Raw(`SELECT id, parent, type FROM zt_project WHERE id = ? AND deleted = '0' LIMIT 1`, current).
			Scan(&row).Error
		if err != nil {
			return 0, err
		}
		if row.ID == 0 {
			return 0, errors.New("执行不存在")
		}
		if strings.TrimSpace(row.Type) == "project" {
			return row.ID, nil
		}
		if row.Parent == 0 {
			return 0, errors.New("未找到所属项目")
		}
		current = row.Parent
	}
	return 0, errors.New("项目层级过深")
}

// UserCanAccessProduct 判断用户是否有权操作指定产品/系统。
func (r *Repo) UserCanAccessProduct(ctx context.Context, account string, productID uint) (bool, error) {
	account = strings.TrimSpace(account)
	if account == "" || productID == 0 {
		return false, nil
	}
	isAdmin, err := r.IsAdmin(ctx, account)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}
	products, err := r.GetUserProducts(ctx, account)
	if err != nil {
		return false, err
	}
	for _, product := range products {
		if product.ID == productID {
			return true, nil
		}
	}
	return false, nil
}

// GetDemandMainSystem 查询业需主系统 ID。
func (r *Repo) GetDemandMainSystem(ctx context.Context, demandID uint) (uint, error) {
	if demandID == 0 {
		return 0, errors.New("业需 ID 无效")
	}
	const query = `SELECT mainSystem FROM zt_demand WHERE id = ? AND deleted = '0' LIMIT 1`
	var mainSystem string
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&mainSystem).Error; err != nil {
		return 0, err
	}
	return parseUintString(mainSystem), nil
}

// GetStoryProductID 查询独立研发需求的主系统 ID（zt_story.product）。
func (r *Repo) GetStoryProductID(ctx context.Context, storyID uint) (uint, error) {
	if storyID == 0 {
		return 0, errors.New("研发需求 ID 无效")
	}
	const query = `SELECT product FROM zt_story WHERE id = ? AND deleted = '0' LIMIT 1`
	var product uint
	if err := r.db.WithContext(ctx).Raw(query, storyID).Scan(&product).Error; err != nil {
		return 0, err
	}
	return product, nil
}

// UpdateWindowProductPlanID 更新窗口-产品关联的计划 ID。
func (r *Repo) UpdateWindowProductPlanID(ctx context.Context, id uint64, planID uint, account string) error {
	return r.db.WithContext(ctx).
		Model(&model.VersionWindowProduct{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"plan":       planID,
			"planSynced": uint8(1),
			"updatedBy":  account,
		}).Error
}

// CreateStory 创建研发需求。
func (r *Repo) CreateStory(ctx context.Context, story *ZtStoryInsert) (uint, error) {
	if story == nil {
		return 0, errors.New("story is nil")
	}
	now := time.Now()
	row := ztmodel.ZtStoryCreate{
		Product:                 story.Product,
		Branch:                  "0",
		Module:                  0,
		Plan:                    "",
		Source:                  "",
		SourceNote:              "",
		Title:                   strings.TrimSpace(story.Title),
		Type:                    "story",
		Pri:                     3,
		Grade:                   1,
		Estimate:                story.Estimate,
		Status:                  "active",
		Stage:                   "planned",
		SourceType:              "demandpool",
		FromDemand:              story.FromDemand,
		Version:                 1,
		OpenedBy:                story.OpenedBy,
		OpenedDate:              now,
		AssignedTo:              strings.TrimSpace(story.AssignedTo),
		IsMainSystemAssociation: story.IsMainSystemAssociation,
		EstimateLaunch:          parseSchedulingDatePtr(story.EstimateLaunch),
		DevelopFinish:           parseSchedulingDatePtr(story.DevelopFinish),
		TestFinish:              parseSchedulingDatePtr(story.TestFinish),
		VerifyPlan:              "",
		Deleted:                 "0",
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// CreateStorySpec 创建研发需求描述。
func (r *Repo) CreateStorySpec(ctx context.Context, spec *ZtStorySpec) error {
	if spec == nil {
		return errors.New("story spec is nil")
	}
	row := ztmodel.ZtStoryspec{
		Story:   spec.Story,
		Version: spec.Version,
		Title:   strings.TrimSpace(spec.Title),
		Spec:    spec.Spec,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

// CreatePlanStory 关联计划与研发需求。
func (r *Repo) CreatePlanStory(ctx context.Context, planID uint, storyID uint) error {
	if planID == 0 || storyID == 0 {
		return errors.New("plan or story id is invalid")
	}
	row := ztmodel.ZtPlanstory{Plan: planID, Story: storyID, Order: 0}
	return r.db.WithContext(ctx).Create(&row).Error
}

// UpdateStory 更新研发需求字段。
func (r *Repo) UpdateStory(ctx context.Context, storyID uint, updates map[string]interface{}) error {
	if storyID == 0 {
		return errors.New("story id is invalid")
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Table("zt_story").
		Where("id = ? AND deleted = '0'", storyID).
		Updates(updates).Error
}

// zt_action.extra 的标记位 禅道「已删除」列表列的就是 action='deleted' AND extra=1
const storyActionCanUndeleted = "1"

// DeleteStory 软删除研发需求（对齐禅道 story/control.php delete：置 deleted='1'）。
func (r *Repo) DeleteStory(ctx context.Context, storyID uint) error {
	if storyID == 0 {
		return errors.New("story id is invalid")
	}
	return r.db.WithContext(ctx).
		Table("zt_story").
		Where("id = ? AND deleted = '0'", storyID).
		Update("deleted", "1").Error
}

// CreateTask 创建任务。
func (r *Repo) CreateTask(ctx context.Context, task *ZtTaskInsert) (uint, error) {
	if task == nil {
		return 0, errors.New("task is nil")
	}
	now := time.Now()
	row := ztmodel.ZtTask{
		Name:       strings.TrimSpace(task.Name),
		Type:       strings.TrimSpace(task.Type),
		Pri:        task.Pri,
		Story:      task.Story,
		Project:    task.Project,
		Execution:  task.Execution,
		AssignedTo: strings.TrimSpace(task.AssignedTo),
		Estimate:   task.Estimate,
		Consumed:   0,
		Left:       task.Estimate,
		EstStarted: nullableDateValue(task.EstStarted),
		Deadline:   nullableDateValue(task.Deadline),
		Status:     "wait",
		OpenedBy:   task.OpenedBy,
		OpenedDate: now,
		Version:    1,
		Deleted:    "0",
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// CreateTaskSpec 创建任务描述。
func (r *Repo) CreateTaskSpec(ctx context.Context, spec *ZtTaskSpec) error {
	if spec == nil {
		return errors.New("task spec is nil")
	}
	row := ztmodel.ZtTaskspec{
		Task:       spec.Task,
		Version:    spec.Version,
		Name:       strings.TrimSpace(spec.Name),
		EstStarted: nullableDateValue(spec.EstStarted),
		Deadline:   nullableDateValue(spec.Deadline),
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

// UpdateTask 更新任务字段。
func (r *Repo) UpdateTask(ctx context.Context, taskID uint, updates map[string]interface{}) error {
	if taskID == 0 {
		return errors.New("task id is invalid")
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Table("zt_task").
		Where("id = ? AND deleted = '0'", taskID).
		Updates(updates).Error
}

// CloseTask 关闭任务。
func (r *Repo) CloseTask(ctx context.Context, taskID uint, actor string) error {
	return r.UpdateTask(ctx, taskID, map[string]interface{}{
		"status":       "closed",
		"closedBy":     actor,
		"closedDate":   time.Now(),
		"closedReason": "done",
		"assignedTo":   "closed",
	})
}

// DeleteTask 软删除任务（对齐禅道 task/control.php delete：置 deleted='1'）。
func (r *Repo) DeleteTask(ctx context.Context, taskID uint) error {
	if taskID == 0 {
		return errors.New("task id is invalid")
	}
	return r.db.WithContext(ctx).
		Table("zt_task").
		Where("id = ? AND deleted = '0'", taskID).
		Update("deleted", "1").Error
}

// CreateAction 创建禅道操作日志。
func (r *Repo) CreateAction(ctx context.Context, objectType string, objectID uint, action string, actor string, productID uint, projectID uint, executionID uint, extra string) error {
	_, err := ztaction.Create(ctx, r.db, ztaction.Record{
		ObjectType: objectType,
		ObjectID:   objectID,
		ProductID:  productID,
		Project:    projectID,
		Execution:  executionID,
		Actor:      actor,
		Action:     action,
		Extra:      extra,
	})
	return err
}

// UpdateDemandScheduling 更新业需排期字段。
func (r *Repo) UpdateDemandScheduling(ctx context.Context, demandID uint, updates map[string]interface{}) error {
	if demandID == 0 {
		return errors.New("demand id is invalid")
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Table("zt_demand").
		Where("id = ? AND deleted = '0'", demandID).
		Updates(updates).Error
}

// SaveDemandLevelWindow 保存业需级窗口关联（硬删除旧记录后 INSERT 单行）。
// 用 Unscoped 绕开 gorm 软删除以释放 uk_demand_story 唯一键位，避免残留行撞键；
// 实际调用方 service 将本方法包在事务内以保证两步原子性。
func (r *Repo) SaveDemandLevelWindow(ctx context.Context, demandID uint, windowID uint64, account string) error {
	if err := r.db.WithContext(ctx).
		Unscoped().
		Where("demand = ? AND story = 0", demandID).
		Delete(&model.DemandWindow{}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(&model.DemandWindow{
		DemandID:  demandID,
		StoryID:   0,
		WindowID:  windowID,
		CreatedBy: account,
		UpdatedBy: account,
	}).Error
}
