// =============================================================================
// 文件: internal/module/po/repotodo.go
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

	"gorm.io/gorm"

	"workbench/internal/pkg/zentao"
)

// FindTodoItems 查询我的待办列表（V10.1 02 节 7 维 AND 公式）。
// 数据真源:
//   - 业务需求 zt_demand (个人责任 scope: assignedTo/distributedBy/QD/RD/accepter/澄清 PM)
//   - 任务 zt_task (assignedTo=account)
//   - Bug zt_bug (assignedTo=account)
//
// 排序: 优先级 P1>P2>P3, deadline ASC NULLS LAST (zentao 用 '0000-00-00' 表示无 deadline,
//
//	用 '9999-12-31' 占位确保排最后), id DESC。
func (r *Repo) FindTodoItems(ctx context.Context, account string, req TodoListReq) ([]TodoItem, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	keyword := strings.TrimSpace(req.Keyword)

	// 业务需求: actor scope + 关键词 + 阶段 + 办理场景
	demandBase := r.db.WithContext(ctx).Table("zt_demand").
		Where("deleted = ?", "0").
		Where("status IN ?", []string{"draft", "wait", "refuse", "active", "clarified", "developing", "testing", "waitacceptance", "waitdeliver", "acceptanced"}).
		Where("NOT EXISTS (SELECT 1 FROM zt_demand child WHERE child.deleted = ? AND child.parent = zt_demand.id)", "0").
		Where(`(
			assignedTo = ? OR distributedBy = ? OR QD = ? OR RD = ? OR accepter = ?
			OR id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
		)`, account, account, account, account, account, account)

	// 阶段过滤 (V10.1 价值流 10 阶段)
	if req.Stage != "" && req.Stage != "all" {
		demandBase = applyDemandStageFilter(demandBase, req.Stage)
	}

	// 办理场景过滤
	if req.Action != "" && req.Action != "all" {
		demandBase = applyTodoActionFilter(demandBase, req.Action)
	}

	// 排序: 优先级 P1>P2>P3, deadline ASC NULLS LAST (用 '9999-12-31' 占位确保无 deadline 排最后), id DESC
	// 单行 Order 避免 GORM backtick 字符串中嵌入换行符的解析问题
	demandBase = demandBase.Order("CASE WHEN zt_demand.pri = '1' THEN 1 WHEN zt_demand.pri = '2' THEN 2 WHEN zt_demand.pri = '3' THEN 3 ELSE 4 END, CASE WHEN zt_demand.deadline IS NULL OR zt_demand.deadline = '0000-00-00' OR zt_demand.deadline = '0001-01-01' THEN '9999-12-31' ELSE zt_demand.deadline END ASC, zt_demand.id DESC")

	var demandIDs []int64
	// 先取有序 ID，再按该顺序重建详情，避免第二次 IN 查询丢失优先级排序。
	if err := demandBase.
		Where(`(name LIKE ? OR CAST(id AS CHAR) LIKE ?)`, "%"+keyword+"%", "%"+keyword+"%").
		Pluck("zt_demand.id", &demandIDs).Error; err != nil {
		return nil, 0, err
	}

	// 任务: assignedTo + status + 关键词
	var taskIDs []int64
	taskQ := r.db.WithContext(ctx).Table("zt_task").
		Where("deleted = ?", "0").
		Where("assignedTo = ?", account).
		Where("status NOT IN ?", []string{"done", "closed", "cancel"}).
		Where(`(name LIKE ? OR CAST(id AS CHAR) LIKE ?)`, "%"+keyword+"%", "%"+keyword+"%").
		Order("CASE WHEN (deadline IS NULL OR deadline = '0000-00-00' OR deadline = '0001-01-01') THEN 1 ELSE 0 END, deadline ASC, id DESC")
	if err := taskQ.Pluck("zt_task.id", &taskIDs).Error; err != nil {
		return nil, 0, err
	}

	// Bug: assignedTo + status + 关键词
	var bugIDs []int64
	bugQ := r.db.WithContext(ctx).Table("zt_bug").
		Where("deleted = ?", "0").
		Where("assignedTo = ?", account).
		Where("status NOT IN ?", []string{"resolved", "closed"}).
		Where(`(title LIKE ? OR CAST(id AS CHAR) LIKE ?)`, "%"+keyword+"%", "%"+keyword+"%").
		Order("CASE WHEN (deadline IS NULL OR deadline = '0000-00-00' OR deadline = '0001-01-01') THEN 1 ELSE 0 END, deadline ASC, id DESC")
	if err := bugQ.Pluck("zt_bug.id", &bugIDs).Error; err != nil {
		return nil, 0, err
	}

	// 对象类型过滤
	includeDemand := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "demand"
	includeTask := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "task"
	includeBug := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "bug"

	items := make([]TodoItem, 0, len(demandIDs)+len(taskIDs)+len(bugIDs))

	if includeDemand && len(demandIDs) > 0 {
		type row struct {
			ID         int64      `gorm:"column:id"`
			Name       string     `gorm:"column:name"`
			Status     string     `gorm:"column:status"`
			Pri        string     `gorm:"column:pri"`
			AssignedTo string     `gorm:"column:assignedTo"`
			QD         string     `gorm:"column:QD"`
			RD         string     `gorm:"column:RD"`
			Deadline   *time.Time `gorm:"column:deadline"`
		}
		var rows []row
		if err := r.db.WithContext(ctx).Table("zt_demand").
			Where("id IN ?", demandIDs).
			Select("id, name, status, pri, assignedTo, QD, RD, deadline").
			Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		rowsByID := make(map[int64]row, len(rows))
		for _, row := range rows {
			rowsByID[row.ID] = row
		}
		displayMap, _ := r.loadAccountDisplayMap(ctx)
		for _, id := range demandIDs {
			row, ok := rowsByID[id]
			if !ok {
				continue
			}
			deadline := formatTodoDeadline(row.Deadline)
			owner := ""
			if strings.TrimSpace(row.AssignedTo) != "" {
				owner = displayMap[row.AssignedTo]
			} else if strings.TrimSpace(row.QD) != "" {
				owner = displayMap[row.QD]
			} else if strings.TrimSpace(row.RD) != "" {
				owner = displayMap[row.RD]
			}
			priority := ""
			if row.Pri != "" {
				priority = "P" + row.Pri
			}
			relation, responsibility := demandTodoRelation(row.AssignedTo, account)
			items = append(items, TodoItem{
				Kind:           "demand",
				ID:             row.ID,
				DisplayID:      fmt.Sprintf("US%d", row.ID),
				Title:          row.Name,
				Type:           "业务需求",
				Stage:          row.Status,
				Priority:       priority,
				Relation:       relation,
				Responsibility: responsibility,
				Reason:         row.Status,
				Deadline:       deadline,
				Owner:          owner,
				URL:            zentao.DemandViewURL(uint(row.ID)),
				Action:         todoActionLabel(row.Status),
				Blocked:        row.Status == "refuse",
			})
		}
	}

	if includeTask && len(taskIDs) > 0 {
		type row struct {
			ID       int64      `gorm:"column:id"`
			Name     string     `gorm:"column:name"`
			Status   string     `gorm:"column:status"`
			Pri      int        `gorm:"column:pri"`
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
			deadline := formatTodoDeadline(row.Deadline)
			owner := displayMap[account]
			items = append(items, TodoItem{
				Kind:           "task",
				ID:             row.ID,
				DisplayID:      fmt.Sprintf("TASK-%d", row.ID),
				Title:          row.Name,
				Type:           "任务",
				Stage:          row.Status,
				Priority:       fmt.Sprintf("P%d", row.Pri),
				Relation:       "我负责",
				Responsibility: "待我处理",
				Reason:         row.Status,
				Deadline:       deadline,
				Owner:          owner,
				URL:            zentao.TaskViewURL(uint(row.ID)),
				Action:         "办理",
			})
		}
	}

	if includeBug && len(bugIDs) > 0 {
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
				Kind:           "bug",
				ID:             row.ID,
				DisplayID:      fmt.Sprintf("BUG-%d", row.ID),
				Title:          row.Title,
				Type:           "Bug",
				Stage:          row.Status,
				Priority:       fmt.Sprintf("P%d", row.Pri),
				Relation:       "我负责",
				Responsibility: "待我处理",
				Reason:         row.Status,
				Deadline:       "",
				Owner:          displayMap[account],
				URL:            zentao.BugViewURL(uint(row.ID)),
				Action:         "处理",
			})
		}
	}

	extraItems, err := r.FindTodoExtraItems(ctx, RepoFindTodoExtraReq{
		Account: account, Keyword: keyword, ObjectType: req.ObjectType,
	})
	if err != nil {
		return nil, 0, err
	}
	items = append(items, extraItems...)

	total := int64(len(items))
	return items, total, nil
}

func demandTodoRelation(assignedTo, account string) (string, string) {
	if strings.TrimSpace(assignedTo) == strings.TrimSpace(account) {
		return "我负责", "待我处理"
	}
	return "我配合", "待我跟进"
}

func todoActionLabel(status string) string {
	switch status {
	case "draft", "wait", "refuse":
		return "受理"
	case "active":
		return "澄清"
	case "clarified":
		return "排期"
	case "developing", "testing":
		return "跟进"
	case "waitacceptance":
		return "验收"
	case "acceptanced":
		return "发起交付"
	case "waitdeliver":
		return "跟进发布"
	default:
		return "查看"
	}
}

// formatTodoDeadline 将禅道的零日期归为空，避免前端展示不存在的截止日。
func formatTodoDeadline(deadline *time.Time) string {
	if deadline == nil || deadline.Year() <= 1 {
		return ""
	}
	return deadline.Format("2006-01-02")
}

// applyDemandStageFilter 把 V10.1 价值流 10 阶段映射到 zt_demand.status SQL。
func applyDemandStageFilter(q *gorm.DB, stage string) *gorm.DB {
	switch stage {
	case "accept":
		return q.Where("status IN ?", []string{"draft", "wait", "refuse"})
	case "clarify":
		return q.Where("status = ? AND NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id)", "active")
	case "schedule":
		return q.Where(`status = 'clarified' AND (
			developFinish IS NULL OR developFinish = '0000-00-00'
			OR testFinish IS NULL OR testFinish = '0000-00-00'
			OR verifyFinish IS NULL OR verifyFinish = '0000-00-00'
			OR estimateLaunch IS NULL OR estimateLaunch = '0000-00-00'
			OR QD = '' OR mainDevelopers = ''
		)`)
	case "developing":
		return q.Where("status = ?", "developing")
	case "testing":
		return q.Where("status = ?", "testing")
	case "waitacceptance":
		return q.Where(`(
			(status = 'testing')
			OR (status = 'waitacceptance')
		)`)
	case "acceptanced":
		return q.Where("status = ?", "acceptanced")
	case "released":
		return q.Where("status = ? AND overall = '0' AND parent != '-1'", "released")
	}
	return q
}

// applyTodoActionFilter 把 V10.1 办理场景映射到 zt_demand SQL。
func applyTodoActionFilter(q *gorm.DB, action TodoAction) *gorm.DB {
	switch action {
	case TodoActionReview:
		return q.Where("status IN ?", []string{"draft", "wait", "active", "refuse"})
	case TodoActionSchedule:
		return q.Where(`status = 'clarified' AND (
			developFinish IS NULL OR developFinish = '0000-00-00'
			OR testFinish IS NULL OR testFinish = '0000-00-00'
			OR verifyFinish IS NULL OR verifyFinish = '0000-00-00'
			OR estimateLaunch IS NULL OR estimateLaunch = '0000-00-00'
			OR QD = '' OR mainDevelopers = ''
		)`)
	case TodoActionVerify:
		return q.Where("status IN ?", []string{"testing", "waitacceptance"})
	case TodoActionDeliver:
		return q.Where("status = ?", "acceptanced")
	}
	return q
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
			out[row.Account] = FormatAccountName(row.Account, row.Realname)
		}
	}
	return out, nil
}
