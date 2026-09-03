// =============================================================================
// 文件: internal/module/po/repo_todo.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办聚合查询（V10.1 02 节 7 维 AND）。本期打通业务需求 + 任务 + Bug；
//       研发需求/测试单/审批作为扩展点预留。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (r *Repo) FindTodoItems(ctx context.Context, account string, filter TodoScopeFilter) ([]TodoItem, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	keyword := strings.TrimSpace(filter.Keyword)

	// 业务需求：actor scope + status not closed + 关键词
	demandBase := r.roleDemandBase(ctx, account).
		Where("status NOT IN ?", []string{"closed", "cancel"})

	var demandIDs []int64
	if err := demandBase.
		Where(`(name LIKE ? OR CAST(id AS CHAR) LIKE ?)`, "%"+keyword+"%", "%"+keyword+"%").
		Order("zt_demand.id DESC").
		Limit(200).
		Pluck("zt_demand.id", &demandIDs).Error; err != nil {
		return nil, 0, err
	}

	// 任务：assignedTo = account + status not in done/closed/cancel + 关键词
	var taskIDs []int64
	if err := r.db.WithContext(ctx).Table("zt_task").
		Where("deleted = ?", "0").
		Where("assignedTo = ?", account).
		Where("status NOT IN ?", []string{"done", "closed", "cancel"}).
		Where(`(name LIKE ? OR CAST(id AS CHAR) LIKE ?)`, "%"+keyword+"%", "%"+keyword+"%").
		Order("zt_task.id DESC").
		Limit(200).
		Pluck("zt_task.id", &taskIDs).Error; err != nil {
		return nil, 0, err
	}

	// Bug：assignedTo = account + status not in resolved/closed + 关键词
	var bugIDs []int64
	if err := r.db.WithContext(ctx).Table("zt_bug").
		Where("deleted = ?", "0").
		Where("assignedTo = ?", account).
		Where("status NOT IN ?", []string{"resolved", "closed"}).
		Where(`(title LIKE ? OR CAST(id AS CHAR) LIKE ?)`, "%"+keyword+"%", "%"+keyword+"%").
		Order("zt_bug.id DESC").
		Limit(200).
		Pluck("zt_bug.id", &bugIDs).Error; err != nil {
		return nil, 0, err
	}

	items := make([]TodoItem, 0, len(demandIDs)+len(taskIDs)+len(bugIDs))
	now := time.Now().Format("2006-01-02")

	if len(demandIDs) > 0 {
		type row struct {
			ID       int64   `gorm:"column:id"`
			Name     string  `gorm:"column:name"`
			Status   string  `gorm:"column:status"`
			Pri      string  `gorm:"column:pri"`
			QD       string  `gorm:"column:QD"`
			RD       string  `gorm:"column:RD"`
			Deadline *time.Time `gorm:"column:deadline"`
		}
		var rows []row
		if err := r.db.WithContext(ctx).Table("zt_demand").
			Where("id IN ?", demandIDs).
			Select("id, name, status, pri, QD, RD, deadline").
			Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		displayMap, _ := r.loadAccountDisplayMap(ctx)
		for _, row := range rows {
			deadline := ""
			if row.Deadline != nil {
				deadline = row.Deadline.Format("2006-01-02")
			}
			owner := ""
			if strings.TrimSpace(row.QD) != "" {
				owner = displayMap[row.QD]
			} else if strings.TrimSpace(row.RD) != "" {
				owner = displayMap[row.RD]
			}
			priority := ""
			if row.Pri != "" {
				priority = "P" + row.Pri
			}
			items = append(items, TodoItem{
				Kind:         "demand",
				ID:           row.ID,
				DisplayID:    fmt.Sprintf("US%d", row.ID),
				Title:        row.Name,
				Type:         "业务需求",
				Stage:        row.Status,
				Priority:     priority,
				Relation:     "我负责",
				Responsibility: "待我处理",
				Reason:       row.Status,
				Deadline:     deadline,
				Owner:        owner,
				URL:          "",
			})
		}
	}

	if len(taskIDs) > 0 {
		type row struct {
			ID       int64   `gorm:"column:id"`
			Name     string  `gorm:"column:name"`
			Status   string  `gorm:"column:status"`
			Pri      int     `gorm:"column:pri"`
			Deadline *time.Time `gorm:"column:deadline"`
		}
		var rows []row
		if err := r.db.WithContext(ctx).Table("zt_task").
			Where("id IN ?", taskIDs).
			Select("id, name, status, pri, deadline").
			Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		displayMap, _ := r.loadAccountDisplayMap(ctx)
		for _, row := range rows {
			deadline := ""
			if row.Deadline != nil {
				deadline = row.Deadline.Format("2006-01-02")
				if deadline < now {
					deadline = deadline // 已逾期，service 层标红
				}
			}
			owner := displayMap[account]
			items = append(items, TodoItem{
				Kind:         "task",
				ID:           row.ID,
				DisplayID:    fmt.Sprintf("TASK-%d", row.ID),
				Title:        row.Name,
				Type:         "任务",
				Stage:        row.Status,
				Priority:     fmt.Sprintf("P%d", row.Pri),
				Relation:     "我负责",
				Responsibility: "待我处理",
				Reason:       row.Status,
				Deadline:     deadline,
				Owner:        owner,
				URL:          "",
			})
		}
	}

	if len(bugIDs) > 0 {
		type row struct {
			ID     int64  `gorm:"column:id"`
			Title  string `gorm:"column:title"`
			Status string `gorm:"column:status"`
			Pri    int    `gorm:"column:pri"`
		}
		var rows []row
		if err := r.db.WithContext(ctx).Table("zt_bug").
			Where("id IN ?", bugIDs).
			Select("id, title, status, pri").
			Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		displayMap, _ := r.loadAccountDisplayMap(ctx)
		for _, row := range rows {
			items = append(items, TodoItem{
				Kind:         "bug",
				ID:           row.ID,
				DisplayID:    fmt.Sprintf("BUG-%d", row.ID),
				Title:        row.Title,
				Type:         "Bug",
				Stage:        row.Status,
				Priority:     fmt.Sprintf("P%d", row.Pri),
				Relation:     "我负责",
				Responsibility: "待我处理",
				Reason:       row.Status,
				Deadline:     "",
				Owner:        displayMap[account],
				URL:          "",
			})
		}
	}

	total := int64(len(items))
	return items, total, nil
}

// loadAccountDisplayMap 加载 account → 展示名映射（zhentao 兼容）。
func (r *Repo) loadAccountDisplayMap(ctx context.Context) (map[string]string, error) {
	type row struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Table("zt_user").
		Where("deleted = ?", "0").
		Select("account, realname").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Account) == "" {
			continue
		}
		if strings.TrimSpace(row.Realname) == "" {
			out[row.Account] = row.Account
		} else {
			out[row.Account] = row.Realname + "(" + row.Account + ")"
		}
	}
	return out, nil
}

// FindDoneActions 查询我的已办列表（来自 zt_action）。
// V10.1 02 节：已办形成条件 = 本人真实执行的正式业务动作；待办消失不能自动变已办。
// 本期实现：actor = account + 时间段 bound + 对象类型筛选；按 zt_action.id DESC。
