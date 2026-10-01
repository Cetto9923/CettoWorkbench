package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// issueRiskAlias 按 Kind 返回表名、别名及 issue/risk 各自的列名。
func issueRiskAlias(kind string) (a, table, title, plan string) {
	if kind == "risk" {
		return "k", "zt_risk AS k", "k.name", "k.plannedClosedDate"
	}
	return "i", "zt_issue AS i", "i.title", "i.deadline"
}

// issueRiskEmptyScope 判断请求是否应直接返回空结果：团队视角但没有可用账号。
func issueRiskEmptyScope(req IssueRiskListReq) bool {
	return req.Scope != "" && len(req.teamAccounts) == 0
}

// applyIssueRiskFilters 追加 FindIssueRiskList 与 CountIssueRiskProjects 共用的
// 七段过滤条件：可见范围 / 关联 / 状态 / 循环 / 项目 / 关键字 / 逾期。
// 条件顺序与两处调用方逐条一致，合并后 SQL 不变。
func applyIssueRiskFilters(q *gorm.DB, account string, req IssueRiskListReq, a, title, plan string) *gorm.DB {
	if req.Scope != "" {
		q = q.Where(fmt.Sprintf("(%s.createdBy IN ? OR %s.assignedTo IN ?)", a, a), req.teamAccounts, req.teamAccounts)
	} else {
		q = q.Where(fmt.Sprintf("(%s.createdBy = ? OR %s.assignedTo = ?)", a, a), account, account)
	}
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
		q = q.Where(dateSetExpr(plan)+" AND "+plan+" < ?", today)
	}
	return q
}

func (r *Repo) FindIssueRiskList(ctx context.Context, account string, req IssueRiskListReq) ([]issueRiskRow, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return []issueRiskRow{}, 0, nil
	}
	a, table, title, plan := issueRiskAlias(req.Kind)
	severity := "i.severity"
	if req.Kind == "risk" {
		severity = "k.impact"
	}
	if issueRiskEmptyScope(req) {
		return []issueRiskRow{}, 0, nil
	}
	q := r.db.WithContext(ctx).Table(table).
		Joins(fmt.Sprintf("LEFT JOIN zt_user cu ON cu.account = %s.createdBy AND cu.deleted = '0'", a)).
		Joins(fmt.Sprintf("LEFT JOIN zt_user au ON au.account = %s.assignedTo AND au.deleted = '0'", a)).
		Joins(fmt.Sprintf("LEFT JOIN zt_user ru ON ru.account = %s.resolvedBy AND ru.deleted = '0'", a)).
		Joins(fmt.Sprintf("LEFT JOIN zt_user clu ON clu.account = %s.closedBy AND clu.deleted = '0'", a)).
		Joins(fmt.Sprintf("LEFT JOIN zt_project p ON p.id = CAST(NULLIF(%s.project, '') AS UNSIGNED) AND p.deleted = '0'", a)).
		Where(fmt.Sprintf("%s.deleted = '0'", a))
	q = applyIssueRiskFilters(q, account, req, a, title, plan)
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
	countReq := req
	countReq.Page = 1
	countReq.PageSize = 1
	_, total, err := r.FindIssueRiskList(ctx, account, countReq)
	return total, err
}

// CountIssueRiskProjects 返回当前查询条件下出现的项目 ID 与名称，去重并按名称升序，最多 100 条。
// 复用 FindIssueRiskList 的过滤维度（kind/relation/status/keyword/loop/overdue），不读取 pageSize。
func (r *Repo) CountIssueRiskProjects(ctx context.Context, account string, req IssueRiskListReq) ([]IssueRiskProject, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return []IssueRiskProject{}, nil
	}
	a, table, title, plan := issueRiskAlias(req.Kind)
	if issueRiskEmptyScope(req) {
		return []IssueRiskProject{}, nil
	}
	q := r.db.WithContext(ctx).Table(table).
		Joins(fmt.Sprintf("LEFT JOIN zt_project p ON p.id = CAST(NULLIF(%s.project, '') AS UNSIGNED) AND p.deleted = '0'", a)).
		Where(fmt.Sprintf("%s.deleted = '0'", a))
	q = applyIssueRiskFilters(q, account, req, a, title, plan)

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
