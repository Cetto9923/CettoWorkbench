// =============================================================================
// 文件: internal/module/po/todo_query_repo.go
// 模块: PO 工作台
// 类型: repo
// 职责: 我的待办数据库统一聚合、过滤、全局排序与有界分页查询 (P2)。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/pkg/zentao"
)

type TodoPagedResult struct {
	Items   []TodoItem
	Total   int64
	Summary TodoSummary
	Groups  TodoGroupCounts
	Facets  []TodoFacet
}

type todoUnifiedRow struct {
	Kind           string `gorm:"column:kind"`
	ID             int64  `gorm:"column:id"`
	DisplayID      string `gorm:"column:display_id"`
	Title          string `gorm:"column:title"`
	Status         string `gorm:"column:status"`
	PriStr         string `gorm:"column:pri_str"`
	PriorityRank   int    `gorm:"column:priority_rank"`
	DeadlineStr    string `gorm:"column:deadline_str"`
	OwnerAccount   string `gorm:"column:owner_account"`
	Relation       string `gorm:"column:relation"`
	Responsibility string `gorm:"column:responsibility"`
	Blocked        int    `gorm:"column:blocked"`
	TypeOrder      int    `gorm:"column:type_order"`
}

type todoCountsRow struct {
	TotalPending int64 `gorm:"column:total_pending"`
	TodayCount   int64 `gorm:"column:today_count"`
	OverdueCount int64 `gorm:"column:overdue_count"`
	BlockedCount int64 `gorm:"column:blocked_count"`
	P1Count      int64 `gorm:"column:p1_count"`
	DemandCount  int64 `gorm:"column:demand_count"`
	TaskCount    int64 `gorm:"column:task_count"`
	BugCount     int64 `gorm:"column:bug_count"`
}

func (r *Repo) QueryTodoUnified(ctx context.Context, account string, req TodoListReq) (*TodoPagedResult, error) {
	result := &TodoPagedResult{Items: []TodoItem{}, Summary: TodoSummary{}, Groups: TodoGroupCounts{}, Facets: buildTodoFacets(nil)}
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return result, nil
	}

	todayStr := time.Now().Format("2006-01-02")
	displayMap, _ := r.loadAccountDisplayMap(ctx)

	unionSQL, unionArgs := buildTodoUnionSQL(account, req)
	if unionSQL == "" {
		return result, nil
	}

	// 1. 统计 Summary 与 Groups（Focus 之前）
	preTabWhere, preTabArgs := buildTodoOuterWhere(req, displayMap, false, todayStr)
	summarySQL := fmt.Sprintf(`SELECT
		COUNT(*) AS total_pending,
		COALESCE(SUM(CASE WHEN t.deadline_str = ? THEN 1 ELSE 0 END), 0) AS today_count,
		COALESCE(SUM(CASE WHEN t.deadline_str != '9999-12-31' AND t.deadline_str < ? THEN 1 ELSE 0 END), 0) AS overdue_count,
		COALESCE(SUM(CASE WHEN t.blocked = 1 THEN 1 ELSE 0 END), 0) AS blocked_count,
		COALESCE(SUM(CASE WHEN t.priority_rank = 1 THEN 1 ELSE 0 END), 0) AS p1_count,
		COALESCE(SUM(CASE WHEN t.kind = 'demand' THEN 1 ELSE 0 END), 0) AS demand_count,
		COALESCE(SUM(CASE WHEN t.kind = 'task' THEN 1 ELSE 0 END), 0) AS task_count,
		COALESCE(SUM(CASE WHEN t.kind = 'bug' THEN 1 ELSE 0 END), 0) AS bug_count
	FROM (%s) AS t %s`, unionSQL, preTabWhere)

	summaryArgs := append([]interface{}{todayStr, todayStr}, unionArgs...)
	summaryArgs = append(summaryArgs, preTabArgs...)

	var countRow todoCountsRow
	if err := r.db.WithContext(ctx).Raw(summarySQL, summaryArgs...).Scan(&countRow).Error; err != nil {
		return nil, err
	}
	result.Summary = TodoSummary{Pending: int(countRow.TotalPending), Today: int(countRow.TodayCount), Overdue: int(countRow.OverdueCount), Blocked: int(countRow.BlockedCount), P1: int(countRow.P1Count)}
	result.Groups = TodoGroupCounts{All: int(countRow.TotalPending), Demand: int(countRow.DemandCount), Execution: int(countRow.TaskCount), Testing: int(countRow.BugCount)}

	facets, err := r.countTodoFacets(ctx, account, req, displayMap, todayStr)
	if err != nil {
		return nil, err
	}
	result.Facets = facets

	// 2. 统计应用 Tab 与 Focus 后的 Total
	postTabWhere, postTabArgs := buildTodoOuterWhere(req, displayMap, true, todayStr)
	totalArgs := append(append([]interface{}{}, unionArgs...), postTabArgs...)

	if req.Focus == "" || req.Focus == "pending" {
		result.Total = int64(result.Summary.Pending)
	} else {
		totalSQL := fmt.Sprintf(`SELECT COUNT(*) FROM (%s) AS t %s`, unionSQL, postTabWhere)
		if err := r.db.WithContext(ctx).Raw(totalSQL, totalArgs...).Scan(&result.Total).Error; err != nil {
			return nil, err
		}
	}
	if result.Total == 0 {
		return result, nil
	}

	// 3. 有界全局稳定排序分页
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if int64(offset) >= result.Total {
		return result, nil
	}

	pagedSQL := fmt.Sprintf(`SELECT t.kind, t.id, t.display_id, t.title, t.status, t.pri_str, t.priority_rank,
		t.deadline_str, t.owner_account, t.relation, t.responsibility, t.blocked, t.type_order
	FROM (%s) AS t %s
	ORDER BY t.priority_rank ASC, t.deadline_str ASC, t.id DESC, t.type_order ASC
	LIMIT ? OFFSET ?`, unionSQL, postTabWhere)

	pagedArgs := append(append([]interface{}{}, totalArgs...), pageSize, offset)
	var rows []todoUnifiedRow
	if err := r.db.WithContext(ctx).Raw(pagedSQL, pagedArgs...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result.Items = append(result.Items, formatTodoUnifiedItem(row, displayMap))
	}
	return result, nil
}

func buildTodoUnionSQL(account string, req TodoListReq) (string, []interface{}) {
	includeDemand := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "demand"
	includeTask := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "task"
	includeBug := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "bug"
	includeApproval := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "approval"
	includeStory := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "story"
	includeRisk := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "risk"
	includeIssue := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "issue"
	includePersonal := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "todo"
	includeTesttask := req.ObjectType == "" || req.ObjectType == "all" || req.ObjectType == "testtask"

	if (req.Stage != "" && req.Stage != "all") || (req.Action != "" && req.Action != "all") {
		includeTask, includeBug = false, false
		includeApproval, includeStory = false, false
		includeRisk, includeIssue, includePersonal, includeTesttask = false, false, false, false
	}

	var parts []string
	var args []interface{}
	if includeDemand {
		demandSQL := `SELECT 'demand' AS kind, d.id, CONCAT('US', d.id) AS display_id, d.name AS title, d.status,
			CASE WHEN d.pri = '1' THEN '1' WHEN d.pri = '2' THEN '2' WHEN d.pri = '3' THEN '3' WHEN d.pri = '4' THEN '4' ELSE '0' END AS pri_str,
			CASE WHEN d.pri = '1' THEN 1 WHEN d.pri = '2' THEN 2 WHEN d.pri = '3' THEN 3 ELSE 4 END AS priority_rank,
			CASE WHEN d.deadline IS NULL OR d.deadline = '0000-00-00' OR d.deadline = '0001-01-01' THEN '9999-12-31' ELSE DATE_FORMAT(d.deadline, '%Y-%m-%d') END AS deadline_str,
			CASE WHEN TRIM(d.assignedTo) != '' THEN d.assignedTo WHEN TRIM(d.QD) != '' THEN d.QD WHEN TRIM(d.RD) != '' THEN d.RD ELSE '' END AS owner_account,
			CASE WHEN d.assignedTo = ? THEN '我负责' ELSE '我配合' END AS relation,
			CASE WHEN d.assignedTo = ? THEN '待我处理' ELSE '待我跟进' END AS responsibility,
			CASE WHEN d.status = 'refuse' THEN 1 ELSE 0 END AS blocked, 1 AS type_order
		FROM zt_demand AS d
		WHERE d.deleted = '0' AND d.status IN ('draft','wait','refuse','active','clarified','developing','testing','waitacceptance','waitdeliver','acceptanced')
		  AND NOT EXISTS (SELECT 1 FROM zt_demand child WHERE child.deleted = '0' AND child.parent = d.id)
		  AND (d.assignedTo = ? OR d.distributedBy = ? OR d.QD = ? OR d.RD = ? OR d.accepter = ? OR d.id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?))`
		args = append(args, account, account, account, account, account, account, account, account)
		demandSQL += getDemandFilterClauses(req)
		parts = append(parts, demandSQL)
	}
	if includeTask {
		taskSQL := `SELECT 'task' AS kind, t.id, CAST(t.id AS CHAR) AS display_id, t.name AS title, t.status,
			CASE WHEN t.pri = 1 THEN '1' WHEN t.pri = 2 THEN '2' WHEN t.pri = 3 THEN '3' WHEN t.pri = 4 THEN '4' ELSE '0' END AS pri_str,
			CASE WHEN t.pri = 1 THEN 1 WHEN t.pri = 2 THEN 2 WHEN t.pri = 3 THEN 3 ELSE 4 END AS priority_rank,
			CASE WHEN t.deadline IS NULL OR t.deadline = '0000-00-00' OR t.deadline = '0001-01-01' THEN '9999-12-31' ELSE DATE_FORMAT(t.deadline, '%Y-%m-%d') END AS deadline_str,
			t.assignedTo AS owner_account, '我负责' AS relation, '待我处理' AS responsibility, 0 AS blocked, 2 AS type_order
		FROM zt_task AS t WHERE t.deleted = '0' AND t.assignedTo = ? AND t.status NOT IN ('done', 'closed', 'cancel')`
		args = append(args, account)
		parts = append(parts, taskSQL)
	}
	if includeBug {
		bugSQL := `SELECT 'bug' AS kind, b.id, CAST(b.id AS CHAR) AS display_id, b.title AS title, b.status,
			CASE WHEN b.pri = 1 THEN '1' WHEN b.pri = 2 THEN '2' WHEN b.pri = 3 THEN '3' WHEN b.pri = 4 THEN '4' ELSE '0' END AS pri_str,
			CASE WHEN b.pri = 1 THEN 1 WHEN b.pri = 2 THEN 2 WHEN b.pri = 3 THEN 3 ELSE 4 END AS priority_rank,
			'9999-12-31' AS deadline_str, b.assignedTo AS owner_account, '我负责' AS relation, '待我处理' AS responsibility, 0 AS blocked, 3 AS type_order
		FROM zt_bug AS b WHERE b.deleted = '0' AND b.assignedTo = ? AND b.status NOT IN ('resolved', 'closed')`
		args = append(args, account)
		parts = append(parts, bugSQL)
	}
	if includeApproval {
		sql, a := buildTodoApprovalSQL(account)
		parts = append(parts, sql)
		args = append(args, a...)
	}
	if includeStory {
		sql, a := buildTodoStorySQL(account)
		parts = append(parts, sql)
		args = append(args, a...)
	}
	if includeRisk {
		sql, a := buildTodoRiskSQL(account)
		parts = append(parts, sql)
		args = append(args, a...)
	}
	if includeIssue {
		sql, a := buildTodoIssueSQL(account)
		parts = append(parts, sql)
		args = append(args, a...)
	}
	if includePersonal {
		sql, a := buildTodoPersonalSQL(account)
		parts = append(parts, sql)
		args = append(args, a...)
	}
	if includeTesttask {
		sql, a := buildTodoTesttaskSQL(account)
		parts = append(parts, sql)
		args = append(args, a...)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " UNION ALL "), args
}

// todoFacetOrder / todoFacetLabels 覆盖全部接入对象类型（汇总所有审批）。
var todoFacetOrder = []string{"approval", "demand", "story", "task", "bug", "risk", "issue", "todo", "testtask"}

var todoFacetLabels = map[string]string{
	"approval": "审批",
	"demand":   "业务需求",
	"story":    "研发需求",
	"task":     "任务",
	"bug":      "Bug",
	"risk":     "风险",
	"issue":    "问题",
	"todo":     "待办",
	"testtask": "测试单",
}

// buildTodoFacetSQL 构建按对象类型聚合的分面计数 SQL。
// 分面维度自身（ObjectType）与对象域 Tab 不参与过滤，否则选中某个类型后其余芯片计数会归零、
// 无法再切回去；其它维度（关系/责任/关键词/阶段/场景/焦点）保持与列表同一口径。
func buildTodoFacetSQL(account string, req TodoListReq, displayMap map[string]string, todayStr string) (string, []interface{}) {
	facetReq := req
	facetReq.ObjectType = "all"
	unionSQL, unionArgs := buildTodoUnionSQL(account, facetReq)
	if unionSQL == "" {
		return "", nil
	}
	where, whereArgs := buildTodoOuterWhere(facetReq, displayMap, true, todayStr)
	args := append(append([]interface{}{}, unionArgs...), whereArgs...)
	return fmt.Sprintf("SELECT t.kind AS kind, COUNT(*) AS cnt FROM (%s) AS t %s GROUP BY t.kind", unionSQL, where), args
}

func buildTodoFacets(counts map[string]int64) []TodoFacet {
	facets := make([]TodoFacet, 0, len(todoFacetOrder))
	for _, key := range todoFacetOrder {
		facets = append(facets, TodoFacet{Key: key, Label: todoFacetLabels[key], Count: counts[key]})
	}
	return facets
}

// countTodoFacets 用一次 GROUP BY 聚合得到全部对象类型芯片计数，不做全量加载后 Go 内计数。
func (r *Repo) countTodoFacets(ctx context.Context, account string, req TodoListReq, displayMap map[string]string, todayStr string) ([]TodoFacet, error) {
	facetSQL, facetArgs := buildTodoFacetSQL(account, req, displayMap, todayStr)
	if facetSQL == "" {
		return buildTodoFacets(nil), nil
	}
	type facetRow struct {
		Kind string `gorm:"column:kind"`
		Cnt  int64  `gorm:"column:cnt"`
	}
	var rows []facetRow
	if err := r.db.WithContext(ctx).Raw(facetSQL, facetArgs...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.Kind] = row.Cnt
	}
	return buildTodoFacets(counts), nil
}

func getDemandFilterClauses(req TodoListReq) string {
	var sql string
	if req.Stage != "" && req.Stage != "all" {
		switch req.Stage {
		case "accept":
			sql += " AND d.status IN ('draft', 'wait', 'refuse')"
		case "clarify":
			sql += " AND d.status = 'active' AND NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = d.id)"
		case "schedule":
			sql += " AND d.status = 'clarified' AND (d.developFinish IS NULL OR d.developFinish = '0000-00-00' OR d.testFinish IS NULL OR d.testFinish = '0000-00-00' OR d.verifyFinish IS NULL OR d.verifyFinish = '0000-00-00' OR d.estimateLaunch IS NULL OR d.estimateLaunch = '0000-00-00' OR d.QD = '' OR d.mainDevelopers = '')"
		case "developing":
			sql += " AND d.status = 'developing'"
		case "testing":
			sql += " AND d.status = 'testing'"
		case "waitacceptance":
			sql += " AND (d.status = 'testing' OR d.status = 'waitacceptance')"
		case "acceptanced":
			sql += " AND d.status = 'acceptanced'"
		case "publish":
			sql += " AND (d.status = 'waitdeliver' OR (d.status = 'released' AND NOT EXISTS (SELECT 1 FROM zt_demandappraise da WHERE da.demand = d.id AND da.appraiseBy <> '' AND da.appraiseBy IS NOT NULL AND da.appraiseTime IS NOT NULL)))"
		case "released":
			sql += " AND d.status = 'released' AND d.overall = '0' AND d.parent != '-1'"
		}
	}
	if req.Action != "" && req.Action != "all" {
		switch req.Action {
		case TodoActionReview:
			sql += " AND d.status IN ('draft', 'wait', 'active', 'refuse')"
		case TodoActionSchedule:
			sql += " AND d.status = 'clarified' AND (d.developFinish IS NULL OR d.developFinish = '0000-00-00' OR d.testFinish IS NULL OR d.testFinish = '0000-00-00' OR d.verifyFinish IS NULL OR d.verifyFinish = '0000-00-00' OR d.estimateLaunch IS NULL OR d.estimateLaunch = '0000-00-00' OR d.QD = '' OR d.mainDevelopers = '')"
		case TodoActionVerify:
			sql += " AND d.status IN ('testing', 'waitacceptance')"
		case TodoActionDeliver:
			sql += " AND d.status = 'acceptanced'"
		}
	}
	return sql
}

func buildTodoOuterWhere(req TodoListReq, displayMap map[string]string, includeTabAndFocus bool, todayStr string) (string, []interface{}) {
	var conds []string
	var args []interface{}
	if req.Relation != RelationAll && req.Relation != "" {
		switch req.Relation {
		case RelationInCharge:
			conds = append(conds, "t.relation = '我负责'")
		case RelationCooperate:
			conds = append(conds, "t.relation = '我配合'")
		case RelationFollow:
			conds = append(conds, "t.relation = '我关注'")
		}
	}
	if req.Responsibility != ResponsibilityAll && req.Responsibility != "" {
		switch req.Responsibility {
		case ResponsibilityMyAction:
			conds = append(conds, "t.responsibility = '待我处理'")
		case ResponsibilityMyFollowUp:
			conds = append(conds, "t.responsibility = '待我跟进'")
		}
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		escaped := "%" + escapeSQLLike(strings.ToLower(kw)) + "%"
		var matchingAccounts []string
		kwLower := strings.ToLower(kw)
		for acc, displayName := range displayMap {
			if strings.Contains(strings.ToLower(displayName), kwLower) || strings.Contains(strings.ToLower(acc), kwLower) {
				matchingAccounts = append(matchingAccounts, acc)
			}
		}
		if len(matchingAccounts) > 0 {
			conds = append(conds, "(LOWER(t.display_id) LIKE ? OR LOWER(t.title) LIKE ? OR t.owner_account IN (?))")
			args = append(args, escaped, escaped, matchingAccounts)
		} else {
			conds = append(conds, "(LOWER(t.display_id) LIKE ? OR LOWER(t.title) LIKE ?)")
			args = append(args, escaped, escaped)
		}
	}
	if includeTabAndFocus && req.Focus != "" && req.Focus != "pending" {
		switch req.Focus {
		case "today":
			conds = append(conds, "t.deadline_str = ?")
			args = append(args, todayStr)
		case "overdue":
			conds = append(conds, "t.deadline_str != '9999-12-31' AND t.deadline_str < ?")
			args = append(args, todayStr)
		case "blocked":
			conds = append(conds, "t.blocked = 1")
		case "p1":
			conds = append(conds, "t.priority_rank = 1")
		}
	}
	if len(conds) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func formatTodoUnifiedItem(row todoUnifiedRow, displayMap map[string]string) TodoItem {
	owner := displayMap[row.OwnerAccount]
	if owner == "" {
		owner = row.OwnerAccount
	}
	deadline := ""
	if row.DeadlineStr != "" && row.DeadlineStr != "9999-12-31" {
		deadline = row.DeadlineStr
	}
	priority := ""
	if row.PriStr != "" {
		priority = "P" + row.PriStr
	}
	item := TodoItem{
		Kind: row.Kind, ID: row.ID, DisplayID: row.DisplayID, Title: row.Title, Stage: row.Status,
		Priority: priority, Relation: row.Relation, Responsibility: row.Responsibility,
		Reason: row.Status, Deadline: deadline, Owner: owner,
	}
	switch row.Kind {
	case "demand":
		item.Type = "业务需求"
		item.URL = zentao.DemandViewURL(uint(row.ID))
		item.Action = todoActionLabel(row.Status)
		item.Blocked = row.Blocked == 1
	case "task":
		item.Type = "任务"
		item.URL = zentao.TaskViewURL(uint(row.ID))
		item.Action = "办理"
	case "bug":
		item.Type = "Bug"
		item.URL = zentao.BugViewURL(uint(row.ID))
		item.Action = "处理"
		item.Deadline = ""
	case "approval":
		item.Type = "审批"
		item.URL = zentao.URL("approval", "view", fmt.Sprintf("approvalID=%d", row.ID))
		item.Action = "审批"
	case "story":
		item.Type = "研发需求"
		item.URL = zentao.StoryViewURL(uint(row.ID))
		item.Action = "办理"
	case "testtask":
		item.Type = "测试单"
		item.URL = zentao.TesttaskViewURL(uint(row.ID))
		item.Action = "处理"
		item.Blocked = row.Blocked == 1
	case "issue":
		item.Type = "问题"
		item.URL = zentao.IssueViewURL(uint(row.ID))
		item.Action = "处理"
	case "risk":
		item.Type = "风险"
		item.URL = zentao.RiskViewURL(uint(row.ID))
		item.Action = "跟进"
	case "todo":
		item.Type = "待办"
		item.URL = zentao.URL("todo", "view", fmt.Sprintf("todoID=%d", row.ID))
		item.Action = "办理"
	}
	return item
}
