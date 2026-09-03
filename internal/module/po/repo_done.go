// =============================================================================
// 文件: internal/module/po/repo_done.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的已办列表（V10.1 02 节：本人真实执行的正式业务动作真源 = zt_action）。
//       待办消失不能自动变已办（已办形成条件 = actor + action + objectType/objectID + date）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/pkg/zentao"
)

func (r *Repo) FindDoneActions(ctx context.Context, account string, timeRange TimeRange, from, to string, page, pageSize int) ([]DoneAction, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	q := r.db.WithContext(ctx).Table("zt_action AS a").
		Where("a.actor = ?", account)

	// 时间段 bound
	now := time.Now()
	switch timeRange {
	case TimeRangeToday:
		q = q.Where("DATE(a.date) = CURDATE()")
	case TimeRange7d:
		q = q.Where("a.date >= ?", now.AddDate(0, 0, -7))
	case TimeRangeWeek:
		q = q.Where("YEARWEEK(a.date, 3) = YEARWEEK(CURDATE(), 3)")
	case TimeRange30d:
		q = q.Where("a.date >= ?", now.AddDate(0, 0, -30))
	case TimeRangeMonth:
		q = q.Where("DATE_FORMAT(a.date, '%Y-%m') = DATE_FORMAT(CURDATE(), '%Y-%m')")
	case TimeRangeLastMonth:
		q = q.Where("DATE_FORMAT(a.date, '%Y-%m') = DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 1 MONTH), '%Y-%m')")
	case TimeRangeQuarter:
		q = q.Where("QUARTER(a.date) = QUARTER(CURDATE()) AND YEAR(a.date) = YEAR(CURDATE())")
	case TimeRangeCustom:
		if from != "" {
			q = q.Where("a.date >= ?", from)
		}
		if to != "" {
			q = q.Where("a.date <= ?", to)
		}
	case TimeRangeAll:
		// 不加时间过滤
	default:
		// 兜底：当作 all
	}

	// 总数
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	type row struct {
		ID         int64   `gorm:"column:id"`
		ObjectType string  `gorm:"column:objectType"`
		ObjectID   int64   `gorm:"column:objectID"`
		Action     string  `gorm:"column:action"`
		Date       time.Time `gorm:"column:date"`
	}
	var rows []row
	if err := q.
		Order("a.id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Select("a.id, a.objectType, a.objectID, a.action, a.date").
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	// 批量加载对象标题
	demandIDs := make([]int64, 0)
	taskIDs := make([]int64, 0)
	bugIDs := make([]int64, 0)
	storyIDs := make([]int64, 0)
	for _, row := range rows {
		switch row.ObjectType {
		case "demand":
			demandIDs = append(demandIDs, row.ObjectID)
		case "task":
			taskIDs = append(taskIDs, row.ObjectID)
		case "bug":
			bugIDs = append(bugIDs, row.ObjectID)
		case "story":
			storyIDs = append(storyIDs, row.ObjectID)
		}
	}

	nameByKey := make(map[string]string)
	if len(demandIDs) > 0 {
		var ns []struct {
			ID   int64  `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := r.db.WithContext(ctx).Table("zt_demand").
			Where("id IN ?", demandIDs).
			Select("id, name").
			Find(&ns).Error; err == nil {
			for _, n := range ns {
				nameByKey[fmt.Sprintf("demand:%d", n.ID)] = n.Name
			}
		}
	}
	if len(taskIDs) > 0 {
		var ns []struct {
			ID   int64  `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := r.db.WithContext(ctx).Table("zt_task").
			Where("id IN ?", taskIDs).
			Select("id, name").
			Find(&ns).Error; err == nil {
			for _, n := range ns {
				nameByKey[fmt.Sprintf("task:%d", n.ID)] = n.Name
			}
		}
	}
	if len(bugIDs) > 0 {
		var ns []struct {
			ID    int64  `gorm:"column:id"`
			Title string `gorm:"column:title"`
		}
		if err := r.db.WithContext(ctx).Table("zt_bug").
			Where("id IN ?", bugIDs).
			Select("id, title").
			Find(&ns).Error; err == nil {
			for _, n := range ns {
				nameByKey[fmt.Sprintf("bug:%d", n.ID)] = n.Title
			}
		}
	}
	if len(storyIDs) > 0 {
		var ns []struct {
			ID    int64  `gorm:"column:id"`
			Title string `gorm:"column:title"`
		}
		if err := r.db.WithContext(ctx).Table("zt_story").
			Where("id IN ?", storyIDs).
			Select("id, title").
			Find(&ns).Error; err == nil {
			for _, n := range ns {
				nameByKey[fmt.Sprintf("story:%d", n.ID)] = n.Title
			}
		}
	}

	displayMap, _ := r.loadAccountDisplayMap(ctx)
	actor := displayMap[account]
	if actor == "" {
		actor = account
	}

	items := make([]DoneAction, 0, len(rows))
	for _, row := range rows {
		items = append(items, DoneAction{
			ID:         row.ID,
			Actor:      actor,
			Action:     row.Action,
			ObjectType: row.ObjectType,
			ObjectID:   row.ObjectID,
			ObjectName: nameByKey[fmt.Sprintf("%s:%d", row.ObjectType, row.ObjectID)],
			Date:       row.Date.Format("2006-01-02 15:04:05"),
			URL:        objectViewURL(row.ObjectType, uint(row.ObjectID)),
		})
	}

	return items, total, nil
}

// objectViewURL 按 zentao 对象类型拼详情页链接。
func objectViewURL(objectType string, id uint) string {
	switch objectType {
	case "demand":
		return zentao.DemandViewURL(id)
	case "story":
		return zentao.StoryViewURL(id)
	case "task":
		return zentao.TaskViewURL(id)
	case "bug":
		return zentao.BugViewURL(id)
	case "testtask":
		return zentao.TesttaskViewURL(id)
	}
	return ""
}
