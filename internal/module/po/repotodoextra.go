// =============================================================================
// 文件: internal/module/po/repotodoextra.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办附加对象查询（测试单、问题、风险）。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/pkg/zentao"
)

// RepoFindTodoExtraReq 查询附加待办对象的数据参数。
type RepoFindTodoExtraReq struct {
	Account    string
	Keyword    string
	ObjectType string
}

func (r *Repo) FindTodoExtraItems(ctx context.Context, req RepoFindTodoExtraReq) ([]TodoItem, error) {
	items := make([]TodoItem, 0)
	if req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "approval" {
		approvals, err := r.FindCurrentApprovalTodos(ctx, req.Account)
		if err != nil {
			return nil, err
		}
		items = append(items, approvals...)
	}
	if req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "story" {
		stories, err := r.FindOwnedStories(ctx, req)
		if err != nil {
			return nil, err
		}
		items = append(items, stories...)
	}
	if req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "testtask" {
		tests, err := r.FindOwnedTesttasks(ctx, req)
		if err != nil {
			return nil, err
		}
		items = append(items, tests...)
	}
	if req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "issue" {
		issues, err := r.FindIssueTodos(ctx, req)
		if err != nil {
			return nil, err
		}
		items = append(items, issues...)
	}
	if req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "risk" {
		risks, err := r.FindRiskTodos(ctx, req)
		if err != nil {
			return nil, err
		}
		items = append(items, risks...)
	}
	if req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "todo" {
		personal, err := r.FindPersonalTodos(ctx, req)
		if err != nil {
			return nil, err
		}
		items = append(items, personal...)
	}
	return items, nil
}

func (r *Repo) FindOwnedStories(ctx context.Context, req RepoFindTodoExtraReq) ([]TodoItem, error) {
	type row struct {
		ID         int64      `gorm:"column:id"`
		Title      string     `gorm:"column:title"`
		Status     string     `gorm:"column:status"`
		Pri        int        `gorm:"column:pri"`
		Deadline   *time.Time `gorm:"column:deadline"`
		AssignedTo string     `gorm:"column:assignedTo"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("zt_story").
		Where("deleted = ? AND assignedTo = ?", "0", req.Account).
		Where("status NOT IN ?", []string{"closed", "released"}).
		Where("IFNULL(sourceType, '') <> ?", "demandpool").
		Where("(title LIKE ? OR CAST(id AS CHAR) LIKE ?)", "%"+req.Keyword+"%", "%"+req.Keyword+"%").
		// zt_story 无 deadline 列，用 deliverDate(预计交付) 作为研需截止参考。
		Select("id, title, status, pri, deliverDate AS deadline, assignedTo").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	displayMap, err := r.loadAccountDisplayMap(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]TodoItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, TodoItem{
			Kind: "story", ID: row.ID, DisplayID: fmt.Sprintf("%d", row.ID), Title: row.Title,
			Type: "研发需求", Stage: row.Status, Priority: formatTodoPriority(row.Pri), Relation: "我负责",
			Responsibility: "待我处理", Reason: row.Status, Deadline: formatTodoDeadline(row.Deadline),
			Owner: displayMap[row.AssignedTo], URL: zentao.StoryViewURL(uint(row.ID)), Action: "办理",
		})
	}
	return items, nil
}

func (r *Repo) FindPersonalTodos(ctx context.Context, req RepoFindTodoExtraReq) ([]TodoItem, error) {
	type row struct {
		ID         int64      `gorm:"column:id"`
		Name       string     `gorm:"column:name"`
		Status     string     `gorm:"column:status"`
		Pri        int        `gorm:"column:pri"`
		Date       *time.Time `gorm:"column:date"`
		AssignedTo string     `gorm:"column:assignedTo"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("zt_todo").
		Where("deleted = ? AND (account = ? OR assignedTo = ?)", "0", req.Account, req.Account).
		Where("status NOT IN ?", []string{"done", "closed"}).
		Where("(name LIKE ? OR CAST(id AS CHAR) LIKE ?)", "%"+req.Keyword+"%", "%"+req.Keyword+"%").
		Select("id, name, status, pri, date, assignedTo").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	displayMap, err := r.loadAccountDisplayMap(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]TodoItem, 0, len(rows))
	for _, row := range rows {
		owner := row.AssignedTo
		if strings.TrimSpace(owner) == "" {
			owner = req.Account
		}
		items = append(items, TodoItem{
			Kind: "todo", ID: row.ID, DisplayID: fmt.Sprintf("%d", row.ID), Title: row.Name,
			Type: "个人待办", Stage: row.Status, Priority: formatTodoPriority(row.Pri), Relation: "我负责",
			Responsibility: "待我处理", Reason: row.Status, Deadline: formatTodoDeadline(row.Date),
			Owner: displayMap[owner], URL: zentao.URL("todo", "view", fmt.Sprintf("todoID=%d", row.ID)), Action: "办理",
		})
	}
	return items, nil
}

func (r *Repo) FindOwnedTesttasks(ctx context.Context, req RepoFindTodoExtraReq) ([]TodoItem, error) {
	type row struct {
		ID     int64      `gorm:"column:id"`
		Name   string     `gorm:"column:name"`
		Status string     `gorm:"column:status"`
		Pri    int        `gorm:"column:pri"`
		End    *time.Time `gorm:"column:end"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("zt_testtask").
		Where("deleted = ? AND owner = ? AND status <> ?", "0", req.Account, "done").
		Where("(name LIKE ? OR CAST(id AS CHAR) LIKE ?)", "%"+req.Keyword+"%", "%"+req.Keyword+"%").
		Select("id, name, status, pri, end").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	displayMap, err := r.loadAccountDisplayMap(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]TodoItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, TodoItem{
			Kind: "testtask", ID: row.ID, DisplayID: fmt.Sprintf("%d", row.ID), Title: row.Name,
			Type: "测试单", Stage: row.Status, Priority: formatTodoPriority(row.Pri), Relation: "我负责",
			Responsibility: "待我处理", Reason: row.Status, Deadline: formatTodoDeadline(row.End),
			Owner: displayMap[req.Account], URL: zentao.TesttaskViewURL(uint(row.ID)), Action: "处理",
			Blocked: row.Status == "blocked",
		})
	}
	return items, nil
}

func (r *Repo) FindIssueTodos(ctx context.Context, req RepoFindTodoExtraReq) ([]TodoItem, error) {
	var rows []todoIssueRiskRow
	err := r.db.WithContext(ctx).Table("zt_issue").
		Where("deleted = ? AND (assignedTo = ? OR createdBy = ?)", "0", req.Account, req.Account).
		Where("status NOT IN ?", []string{"closed", "cancel"}).
		Where("(title LIKE ? OR CAST(id AS CHAR) LIKE ?)", "%"+req.Keyword+"%", "%"+req.Keyword+"%").
		Select("id, title, status, pri, deadline, assignedTo, createdBy").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	displayMap, err := r.loadAccountDisplayMap(ctx)
	if err != nil {
		return nil, err
	}
	return buildIssueRiskTodoItems(rows, req.Account, displayMap, todoIssueRiskMeta{
		Kind: "issue", Label: "问题", Prefix: "ISSUE",
	}), nil
}

func (r *Repo) FindRiskTodos(ctx context.Context, req RepoFindTodoExtraReq) ([]TodoItem, error) {
	var rows []todoIssueRiskRow
	err := r.db.WithContext(ctx).Table("zt_risk").
		Where("deleted = ? AND (assignedTo = ? OR createdBy = ?)", "0", req.Account, req.Account).
		Where("status NOT IN ?", []string{"closed", "cancel"}).
		Where("(name LIKE ? OR CAST(id AS CHAR) LIKE ?)", "%"+req.Keyword+"%", "%"+req.Keyword+"%").
		Select("id, name AS title, status, pri, plannedClosedDate AS deadline, assignedTo, createdBy").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	displayMap, err := r.loadAccountDisplayMap(ctx)
	if err != nil {
		return nil, err
	}
	return buildIssueRiskTodoItems(rows, req.Account, displayMap, todoIssueRiskMeta{
		Kind: "risk", Label: "风险", Prefix: "RISK",
	}), nil
}

type todoIssueRiskRow struct {
	ID         int64      `gorm:"column:id"`
	Title      string     `gorm:"column:title"`
	Status     string     `gorm:"column:status"`
	Pri        string     `gorm:"column:pri"` // zt_issue/zt_risk.pri 为 char(30)，混存 low/middle/high/urgent 与数字串
	Deadline   *time.Time `gorm:"column:deadline"`
	AssignedTo string     `gorm:"column:assignedTo"`
	CreatedBy  string     `gorm:"column:createdBy"`
}

type todoIssueRiskMeta struct {
	Kind   string
	Label  string
	Prefix string
}

func buildIssueRiskTodoItems(rows []todoIssueRiskRow, account string, displayMap map[string]string, meta todoIssueRiskMeta) []TodoItem {
	items := make([]TodoItem, 0, len(rows))
	for _, row := range rows {
		relation, responsibility := "我配合", "待我跟进"
		ownerAccount := row.AssignedTo
		if row.AssignedTo == account || (strings.TrimSpace(row.AssignedTo) == "" && row.CreatedBy == account) {
			relation, responsibility = "我负责", "待我处理"
			if strings.TrimSpace(ownerAccount) == "" {
				ownerAccount = row.CreatedBy
			}
		}
		items = append(items, TodoItem{
			Kind: meta.Kind, ID: row.ID, DisplayID: fmt.Sprintf("%d", row.ID), Title: row.Title,
			Type: meta.Label, Stage: row.Status, Priority: issueRiskPriLabel(row.Pri), Relation: relation,
			Responsibility: responsibility, Reason: row.Status, Deadline: formatTodoDeadline(row.Deadline),
			Owner: displayMap[ownerAccount], URL: zentao.URL(meta.Kind, "view", fmt.Sprintf("%sID=%d", meta.Kind, row.ID)),
			Action: "处理", Blocked: row.Status == "blocked" || row.Status == "hangup",
		})
	}
	return items
}

func formatTodoPriority(priority int) string {
	if priority <= 0 {
		return ""
	}
	return fmt.Sprintf("P%d", priority)
}

// issueRiskPriLabel 把 zt_issue/zt_risk.pri（char(30)，混存数字串与 low/middle/high/urgent）映射为 P1..P4。
// zentao 优先级语义: 1/urgent 最高 → P1，4/low 最低 → P4；无法识别返回空。
func issueRiskPriLabel(pri string) string {
	switch strings.ToLower(strings.TrimSpace(pri)) {
	case "1", "urgent", "immediate":
		return "P1"
	case "2", "high":
		return "P2"
	case "3", "middle", "medium":
		return "P3"
	case "4", "low":
		return "P4"
	}
	return ""
}
