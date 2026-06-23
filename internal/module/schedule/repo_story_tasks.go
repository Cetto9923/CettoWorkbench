// =============================================================================
// 文件: internal/module/schedule/repo_story_tasks.go
// 模块: 排期工作台
// 类型: action
// 职责: 维护任务弹窗研发需求详情只读查询。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strings"
)

type storyTaskDetailRow struct {
	ID         uint   `gorm:"column:id"`
	Title      string `gorm:"column:title"`
	Product    uint   `gorm:"column:product"`
	FromDemand uint   `gorm:"column:fromDemand"`
	AssignedTo string `gorm:"column:assignedTo"`
	Status     string `gorm:"column:status"`
	Spec       string `gorm:"column:spec"`
	Verify     string `gorm:"column:verify"`
}

type storyWindowDetailRow struct {
	WindowID    uint   `gorm:"column:windowID"`
	WindowName  string `gorm:"column:windowName"`
	ReleaseDate string `gorm:"column:releaseDate"`
}

type storyAttachmentRow struct {
	ID        uint   `gorm:"column:id"`
	Title     string `gorm:"column:title"`
	Extension string `gorm:"column:extension"`
}

type recentProjectRow struct {
	ProjectID   uint `gorm:"column:projectId"`
	ExecutionID uint `gorm:"column:executionId"`
}

// StoryTaskDetail 维护任务弹窗研发需求详情（Repo 层聚合）。
type StoryTaskDetail struct {
	StoryID           uint
	Title             string
	ProductID         uint
	FromDemand        uint
	AssignedTo        string
	Status            string
	Spec              string
	Verify            string
	DemandName        string
	WindowName        string
	ReleaseDate       string
	DefaultProjectID  uint
	DefaultExecutionID uint
	Attachments       []StoryAttachmentItem
}

// GetStoryTaskDetail 查询维护任务弹窗所需的研发需求详情。
func (r *Repo) GetStoryTaskDetail(ctx context.Context, storyID uint) (*StoryTaskDetail, error) {
	if storyID == 0 {
		return nil, errors.New("研发需求 ID 无效")
	}

	const storyQuery = `
SELECT
  s.id,
  s.title,
  s.product,
  s.fromDemand,
  s.assignedTo,
  s.status,
  IFNULL(ss.spec, '') AS spec,
  IFNULL(ss.verify, '') AS verify
FROM zt_story s
LEFT JOIN zt_storyspec ss ON ss.story = s.id AND ss.version = s.version
WHERE s.id = ?
  AND s.deleted = '0'
LIMIT 1`

	var row storyTaskDetailRow
	if err := r.db.WithContext(ctx).Raw(storyQuery, storyID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errors.New("研发需求不存在")
	}

	detail := &StoryTaskDetail{
		StoryID:    row.ID,
		Title:      strings.TrimSpace(row.Title),
		ProductID:  row.Product,
		FromDemand: row.FromDemand,
		AssignedTo: strings.TrimSpace(row.AssignedTo),
		Status:     strings.TrimSpace(row.Status),
		Spec:       strings.TrimSpace(row.Spec),
		Verify:     strings.TrimSpace(row.Verify),
	}

	if row.FromDemand > 0 {
		name, err := r.findDemandName(ctx, row.FromDemand)
		if err != nil {
			return nil, err
		}
		detail.DemandName = name
	}

	windowName, releaseDate, err := r.findStoryWindowDetail(ctx, storyID)
	if err != nil {
		return nil, err
	}
	detail.WindowName = windowName
	detail.ReleaseDate = releaseDate

	defaultProjectID, defaultExecutionID, err := r.findRecentProductTaskProject(ctx, row.Product)
	if err != nil {
		return nil, err
	}
	detail.DefaultProjectID = defaultProjectID
	detail.DefaultExecutionID = defaultExecutionID

	attachments, err := r.findStoryAttachments(ctx, storyID)
	if err != nil {
		return nil, err
	}
	detail.Attachments = attachments

	return detail, nil
}

func (r *Repo) findDemandName(ctx context.Context, demandID uint) (string, error) {
	const query = `
SELECT CONCAT('REQ-', id, ' ', name) AS name
FROM zt_demand
WHERE id = ?
  AND deleted = '0'
LIMIT 1`

	var name string
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&name).Error; err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

func (r *Repo) findStoryWindowDetail(ctx context.Context, storyID uint) (windowName, releaseDate string, err error) {
	const query = `
SELECT
  vw.id AS windowID,
  vw.name AS windowName,
  DATE_FORMAT(vw.releaseDate, '%Y-%m-%d') AS releaseDate
FROM zt_planstory ps
INNER JOIN zt_versionwindowproduct vwp
  ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
INNER JOIN zt_versionwindow vw
  ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
WHERE ps.story = ?
ORDER BY vw.id ASC
LIMIT 1`

	var row storyWindowDetailRow
	if err := r.db.WithContext(ctx).Raw(query, storyID).Scan(&row).Error; err != nil {
		return "", "", err
	}
	return strings.TrimSpace(row.WindowName), formatZenTaoDate(row.ReleaseDate), nil
}

func (r *Repo) findRecentProductTaskProject(ctx context.Context, productID uint) (projectID, executionID uint, err error) {
	if productID == 0 {
		return 0, 0, nil
	}

	const query = `
SELECT t.project AS projectId, t.execution AS executionId
FROM zt_task t
INNER JOIN zt_story s ON s.id = t.story AND s.deleted = '0'
WHERE s.product = ?
  AND t.deleted = '0'
  AND t.project > 0
ORDER BY t.id DESC
LIMIT 1`

	var row recentProjectRow
	if err := r.db.WithContext(ctx).Raw(query, productID).Scan(&row).Error; err != nil {
		return 0, 0, err
	}
	return row.ProjectID, row.ExecutionID, nil
}

func (r *Repo) findStoryAttachments(ctx context.Context, storyID uint) ([]StoryAttachmentItem, error) {
	const query = `
SELECT id, title, extension
FROM zt_file
WHERE objectType = 'story'
  AND objectID = ?
  AND deleted = '0'
ORDER BY id ASC`

	var rows []storyAttachmentRow
	if err := r.db.WithContext(ctx).Raw(query, storyID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]StoryAttachmentItem, 0, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		title := strings.TrimSpace(row.Title)
		ext := strings.TrimSpace(row.Extension)
		if title == "" {
			title = "附件"
			if ext != "" {
				title += "." + ext
			}
		}
		out = append(out, StoryAttachmentItem{
			ID:    row.ID,
			Title: title,
		})
	}
	return out, nil
}

// GetTaskExecution 查询任务所属执行 ID。
func (r *Repo) GetTaskExecution(ctx context.Context, taskID uint) (uint, error) {
	if taskID == 0 {
		return 0, errors.New("任务 ID 无效")
	}

	const query = `
SELECT execution
FROM zt_task
WHERE id = ?
  AND deleted = '0'
LIMIT 1`

	var executionID uint
	if err := r.db.WithContext(ctx).Raw(query, taskID).Scan(&executionID).Error; err != nil {
		return 0, err
	}
	if executionID == 0 {
		return 0, errors.New("任务不存在")
	}
	return executionID, nil
}
