// =============================================================================
// 文件: internal/module/kanban/task_columns.go
// 模块: 工作看板
// 类型: readonly
// 职责: 任务行归入 wait/doing/done 三列，并计算超期（不含 pause）。
// 依赖: internal/pkg/zentao
// =============================================================================

package kanban

import (
	"fmt"
	"strings"
	"time"

	"workbench/internal/pkg/zentao"
)

// taskRow 任务看板查询行（Repo 与列归类共用）。
type taskRow struct {
	ID         int64
	Name       string
	Type       string
	Status     string
	StoryID    int64
	AssignedTo string
	FinishedBy string
	Deadline   *time.Time
	StoryTitle string
}

func taskColumnKey(status string) string {
	switch strings.TrimSpace(status) {
	case "wait":
		return "wait"
	case "doing":
		return "doing"
	case "done":
		return "done"
	default:
		return ""
	}
}

func taskIsOverdue(status string, deadline *time.Time, today time.Time) bool {
	if status == "done" {
		return false
	}
	return deadline != nil && !deadline.IsZero() && deadline.Before(today)
}

func newEmptyTaskColumns() []TaskColumn {
	return []TaskColumn{
		{Key: "wait", Name: "未开始", Items: []TaskItem{}},
		{Key: "doing", Name: "进行中", Items: []TaskItem{}},
		{Key: "done", Name: "已完成", Items: []TaskItem{}},
	}
}

// buildTaskColumns 将任务行归入三列；pause/closed 等跳过；done 卡 owner 取完成者。
func buildTaskColumns(rows []taskRow, displayMap map[string]string, today time.Time) ([]TaskColumn, TaskSummary) {
	cols := newEmptyTaskColumns()
	var summary TaskSummary
	for _, row := range rows {
		key := taskColumnKey(row.Status)
		if key == "" {
			continue
		}
		overdue := taskIsOverdue(row.Status, row.Deadline, today)
		if overdue {
			summary.Overdue++
		}
		item := toTaskItem(row, displayMap, overdue)
		for i := range cols {
			if cols[i].Key == key {
				cols[i].Items = append(cols[i].Items, item)
				break
			}
		}
	}
	return cols, summary
}

func toTaskItem(row taskRow, displayMap map[string]string, overdue bool) TaskItem {
	ownerAccount := strings.TrimSpace(row.AssignedTo)
	if row.Status == "done" {
		ownerAccount = strings.TrimSpace(row.FinishedBy)
		if ownerAccount == "" {
			ownerAccount = strings.TrimSpace(row.AssignedTo)
		}
	}
	owner := lookupDisplay(displayMap, ownerAccount)
	deadline := ""
	if row.Deadline != nil && !row.Deadline.IsZero() {
		deadline = row.Deadline.Format("2006-01-02")
	}
	return TaskItem{
		ID:           row.ID,
		DisplayID:    fmt.Sprintf("%d", row.ID),
		Title:        row.Name,
		Status:       row.Status,
		Type:         taskTypeLabel(row.Type),
		StoryID:      row.StoryID,
		StoryTitle:   row.StoryTitle,
		Owner:        owner,
		OwnerAccount: ownerAccount,
		Deadline:     deadline,
		Blocked:      false,
		Overdue:      overdue,
		URL:          zentao.URL("task", "view", fmt.Sprintf("taskID=%d", row.ID)),
	}
}

// taskTypeLabel 对齐禅道任务类型中文名
func taskTypeLabel(typ string) string {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return ""
	}
	labels := map[string]string{
		"devel":        "开发",
		"design":       "设计（UI、数据库、概设、详设）",
		"request":      "需求",
		"test":         "SIT测试",
		"review":       "评审（需求、开发、测试、代码）",
		"Online":       "生产变更",
		"OLtest":       "验证测试",
		"OLDebug":      "生产缺陷排查",
		"Train":        "学习或培训他人",
		"Meeting":      "会议",
		"meeting":      "会议", // 兼容历史小写
		"misc":         "其他",
		"smokeTesting": "冒烟测试",
		"testCase":     "详细案例编写",
	}
	if label, ok := labels[typ]; ok {
		return label
	}
	return typ
}
