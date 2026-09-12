package po

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (r *Repo) FindIssueRiskList(ctx context.Context, account string, req IssueRiskListReq) ([]issueRiskRow, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return []issueRiskRow{}, 0, nil
	}
	a, table, title, plan, severity := "i", "zt_issue AS i", "i.title", "i.deadline", "i.severity"
	if req.Kind == "risk" {
		a, table, title, plan, severity = "k", "zt_risk AS k", "k.name", "k.plannedClosedDate", "k.impact"
	}
	q := r.db.WithContext(ctx).Table(table).
		Joins(fmt.Sprintf("LEFT JOIN zt_user cu ON cu.account = %s.createdBy AND cu.deleted = '0'", a)).
		Joins(fmt.Sprintf("LEFT JOIN zt_user au ON au.account = %s.assignedTo AND au.deleted = '0'", a)).
		Joins(fmt.Sprintf("LEFT JOIN zt_user ru ON ru.account = %s.resolvedBy AND ru.deleted = '0'", a)).
		Joins(fmt.Sprintf("LEFT JOIN zt_user clu ON clu.account = %s.closedBy AND clu.deleted = '0'", a)).
		Joins(fmt.Sprintf("LEFT JOIN zt_project p ON p.id = CAST(NULLIF(%s.project, '') AS UNSIGNED) AND p.deleted = '0'", a)).
		Where(fmt.Sprintf("%s.deleted = '0'", a)).
		Where(fmt.Sprintf("(%s.createdBy = ? OR %s.assignedTo = ?)", a, a), account, account)
	if req.Relation == "myAction" {
		q = q.Where(fmt.Sprintf("%s.assignedTo = ?", a), account)
	}
	if req.Relation == "mySubmit" {
		q = q.Where(fmt.Sprintf("%s.createdBy = ?", a), account)
	}
	if req.Status != "" {
		q = q.Where(fmt.Sprintf("%s.status = ?", a), req.Status)
	}
	if req.Loop == "open" {
		q = q.Where(fmt.Sprintf("%s.status IN ?", a), irOpenStatuses)
	}
	if req.Loop == "closed" {
		q = q.Where(fmt.Sprintf("%s.status IN ?", a), irClosedStatuses)
	}
	if req.Project > 0 {
		q = q.Where("p.id = ?", req.Project)
	}
	if req.Keyword != "" {
		like := "%" + req.Keyword + "%"
		q = q.Where(fmt.Sprintf("(CAST(%s.id AS CHAR) LIKE ? OR %s LIKE ? OR COALESCE(p.name,'') LIKE ?)", a, title), like, like, like)
	}
	if req.Overdue {
		today := time.Now().Format("2006-01-02")
		q = q.Where(fmt.Sprintf("%s IS NOT NULL AND %s != '0000-00-00' AND %s < ?", plan, plan, plan), today)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []issueRiskRow{}
	sel := fmt.Sprintf("%s.id AS id, %s AS title, %s.pri AS pri, %s AS severity, %s.status AS status, %s.createdDate AS created_date, %s AS plan_date, %s.createdBy AS created_by, %s.assignedTo AS assigned_to, COALESCE(cu.realname,%s.createdBy) AS creator_name, COALESCE(au.realname,%s.assignedTo) AS handler_name, COALESCE(ru.realname,%s.resolvedBy,'') AS resolved_name, COALESCE(clu.realname,%s.closedBy,'') AS closed_name, COALESCE(p.name,'') AS project_name", a, title, a, severity, a, a, plan, a, a, a, a, a, a)
	err := q.Select(sel).Order(fmt.Sprintf("%s.id DESC", a)).Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Scan(&rows).Error
	return rows, total, err
}

func (r *Repo) CountIssueRisk(ctx context.Context, account string, req IssueRiskListReq) (int64, error) {
	_, total, err := r.FindIssueRiskList(ctx, account, IssueRiskListReq{Kind: req.Kind, Relation: req.Relation, Status: req.Status, Keyword: req.Keyword, Loop: req.Loop, Overdue: req.Overdue, Project: req.Project, Page: 1, PageSize: 1})
	return total, err
}

// CountIssueRiskProjects 返回当前查询条件下出现的项目 ID 与名称，去重并按名称升序，最多 100 条。
// 复用 FindIssueRiskList 的过滤维度（kind/relation/status/keyword/loop/overdue），不读取 pageSize。
func (r *Repo) CountIssueRiskProjects(ctx context.Context, account string, req IssueRiskListReq) ([]IssueRiskProject, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return []IssueRiskProject{}, nil
	}
	a, table, title, plan := "i", "zt_issue AS i", "i.title", "i.deadline"
	if req.Kind == "risk" {
		a, table, title, plan = "k", "zt_risk AS k", "k.name", "k.plannedClosedDate"
	}
	q := r.db.WithContext(ctx).Table(table).Joins(fmt.Sprintf("LEFT JOIN zt_project p ON p.id = CAST(NULLIF(%s.project, '') AS UNSIGNED) AND p.deleted = '0'", a)).Where(fmt.Sprintf("%s.deleted = '0'", a)).Where(fmt.Sprintf("(%s.createdBy = ? OR %s.assignedTo = ?)", a, a), account, account)
	if req.Relation == "myAction" {
		q = q.Where(fmt.Sprintf("%s.assignedTo = ?", a), account)
	}
	if req.Relation == "mySubmit" {
		q = q.Where(fmt.Sprintf("%s.createdBy = ?", a), account)
	}
	if req.Status != "" {
		q = q.Where(fmt.Sprintf("%s.status = ?", a), req.Status)
	}
	if req.Loop == "open" {
		q = q.Where(fmt.Sprintf("%s.status IN ?", a), irOpenStatuses)
	}
	if req.Loop == "closed" {
		q = q.Where(fmt.Sprintf("%s.status IN ?", a), irClosedStatuses)
	}
	if req.Project > 0 {
		q = q.Where("p.id = ?", req.Project)
	}
	if req.Keyword != "" {
		like := "%" + req.Keyword + "%"
		q = q.Where(fmt.Sprintf("(CAST(%s.id AS CHAR) LIKE ? OR %s LIKE ? OR COALESCE(p.name,'') LIKE ?)", a, title), like, like, like)
	}
	if req.Overdue {
		today := time.Now().Format("2006-01-02")
		q = q.Where(fmt.Sprintf("%s IS NOT NULL AND %s != '0000-00-00' AND %s < ?", plan, plan, plan), today)
	}
	type projectRow struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	rows := []projectRow{}
	err := q.Distinct("p.id AS id, p.name AS name").Order("p.name ASC").Limit(100).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]IssueRiskProject, 0, len(rows))
	for _, row := range rows {
		if row.ID == 0 || strings.TrimSpace(row.Name) == "" {
			continue
		}
		out = append(out, IssueRiskProject{ID: row.ID, Name: row.Name})
	}
	return out, nil
}
