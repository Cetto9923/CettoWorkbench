// =============================================================================
// 文件: internal/module/po/repodone_enrich.go
// 模块: PO 工作台
// 类型: repo
// 职责: 我的已办动作详情（F01）数据访问：对象上下文 / 历史 / 邻近时间线组装。
//       授权预查询方法见 repodone_authz.go（FindDoneActionActor 等）。
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

type doneActionDBRow struct {
	ID         int64     `gorm:"column:id"`
	ObjectType string    `gorm:"column:objectType"`
	ObjectID   int64     `gorm:"column:objectID"`
	Action     string    `gorm:"column:action"`
	Actor      string    `gorm:"column:actor"`
	Date       time.Time `gorm:"column:date"`
}

type doneHistoryRow struct {
	Action uint   `gorm:"column:action"`
	Field  string `gorm:"column:field"`
	Old    string `gorm:"column:old"`
	New    string `gorm:"column:new"`
}

type doneObjectContext struct {
	Title         string
	Status        string
	ProjectID     int64
	ProjectName   string
	ExecutionID   int64
	ExecutionName string
	ProductName   string
	CurrentOwner  string
}

// fetchActionHistories 批量获取当前页动作的状态变更 history。
func (r *Repo) fetchActionHistories(ctx context.Context, actionIDs []int64) map[int64][2]string {
	out := make(map[int64][2]string, len(actionIDs))
	if r == nil || r.db == nil || len(actionIDs) == 0 {
		return out
	}
	var rows []doneHistoryRow
	err := r.db.WithContext(ctx).Table("zt_history").
		Select("action, field, `old`, `new`").
		Where("action IN ? AND field IN ('status', 'stage')", actionIDs).
		Order("id ASC").
		Scan(&rows).Error
	if err != nil {
		return out
	}
	for _, h := range rows {
		out[int64(h.Action)] = [2]string{h.Old, h.New}
	}
	return out
}

// fetchObjectContexts 批量加载对象的名称、状态、所属项目及执行。
func (r *Repo) fetchObjectContexts(ctx context.Context, rows []doneActionDBRow) map[string]doneObjectContext {
	out := make(map[string]doneObjectContext, len(rows))
	if r == nil || r.db == nil || len(rows) == 0 {
		return out
	}

	demandIDs := make([]int64, 0)
	storyIDs := make([]int64, 0)
	taskIDs := make([]int64, 0)
	bugIDs := make([]int64, 0)
	todoIDs := make([]int64, 0)
	charterIDs := make([]int64, 0)
	planchangeIDs := make([]int64, 0)
	buildguidelineIDs := make([]int64, 0)
	reviewIDs := make([]int64, 0)
	caseIDs := make([]int64, 0)

	for _, row := range rows {
		switch row.ObjectType {
		case "demand":
			demandIDs = append(demandIDs, row.ObjectID)
		case "story":
			storyIDs = append(storyIDs, row.ObjectID)
		case "task":
			taskIDs = append(taskIDs, row.ObjectID)
		case "bug":
			bugIDs = append(bugIDs, row.ObjectID)
		case "todo":
			todoIDs = append(todoIDs, row.ObjectID)
		case "charter":
			charterIDs = append(charterIDs, row.ObjectID)
		case "planchange":
			planchangeIDs = append(planchangeIDs, row.ObjectID)
		case "buildguideline":
			buildguidelineIDs = append(buildguidelineIDs, row.ObjectID)
		case "review":
			reviewIDs = append(reviewIDs, row.ObjectID)
		case "case":
			caseIDs = append(caseIDs, row.ObjectID)
		}
	}

	// 1. Demand
	if len(demandIDs) > 0 {
		type dRow struct {
			ID          int64  `gorm:"column:id"`
			Name        string `gorm:"column:name"`
			Status      string `gorm:"column:status"`
			AssignedTo  string `gorm:"column:assignedTo"`
			ProductName string `gorm:"column:product_name"`
		}
		var list []dRow
		_ = r.db.WithContext(ctx).Table("zt_demand AS d").
			Select("d.id, d.name, d.status, d.assignedTo, COALESCE(p.name, '') AS product_name").
			Joins("LEFT JOIN zt_product AS p ON p.id = d.product").
			Where("d.id IN ?", demandIDs).
			Scan(&list).Error
		for _, d := range list {
			out[fmt.Sprintf("demand:%d", d.ID)] = doneObjectContext{
				Title:        d.Name,
				Status:       d.Status,
				ProductName:  d.ProductName,
				CurrentOwner: d.AssignedTo,
			}
		}
	}

	// 2. Story
	if len(storyIDs) > 0 {
		type sRow struct {
			ID          int64  `gorm:"column:id"`
			Title       string `gorm:"column:title"`
			Status      string `gorm:"column:status"`
			AssignedTo  string `gorm:"column:assignedTo"`
			ProductName string `gorm:"column:product_name"`
		}
		var list []sRow
		_ = r.db.WithContext(ctx).Table("zt_story AS s").
			Select("s.id, s.title, s.status, s.assignedTo, COALESCE(p.name, '') AS product_name").
			Joins("LEFT JOIN zt_product AS p ON p.id = s.product").
			Where("s.id IN ?", storyIDs).
			Scan(&list).Error
		for _, s := range list {
			out[fmt.Sprintf("story:%d", s.ID)] = doneObjectContext{
				Title:        s.Title,
				Status:       s.Status,
				ProductName:  s.ProductName,
				CurrentOwner: s.AssignedTo,
			}
		}
	}

	// 3. Task
	if len(taskIDs) > 0 {
		type tRow struct {
			ID            int64  `gorm:"column:id"`
			Name          string `gorm:"column:name"`
			Status        string `gorm:"column:status"`
			Project       int64  `gorm:"column:project"`
			Execution     int64  `gorm:"column:execution"`
			AssignedTo    string `gorm:"column:assignedTo"`
			ProjectName   string `gorm:"column:project_name"`
			ExecutionName string `gorm:"column:execution_name"`
		}
		var list []tRow
		_ = r.db.WithContext(ctx).Table("zt_task AS t").
			Select("t.id, t.name, t.status, t.project, t.execution, t.assignedTo, COALESCE(p.name, '') AS project_name, COALESCE(e.name, '') AS execution_name").
			Joins("LEFT JOIN zt_project AS p ON p.id = t.project").
			Joins("LEFT JOIN zt_project AS e ON e.id = t.execution").
			Where("t.id IN ?", taskIDs).
			Scan(&list).Error
		for _, t := range list {
			out[fmt.Sprintf("task:%d", t.ID)] = doneObjectContext{
				Title:         t.Name,
				Status:        t.Status,
				ProjectID:     t.Project,
				ProjectName:   t.ProjectName,
				ExecutionID:   t.Execution,
				ExecutionName: t.ExecutionName,
				CurrentOwner:  t.AssignedTo,
			}
		}
	}

	// 4. Bug
	if len(bugIDs) > 0 {
		type bRow struct {
			ID            int64  `gorm:"column:id"`
			Title         string `gorm:"column:title"`
			Status        string `gorm:"column:status"`
			Project       int64  `gorm:"column:project"`
			Execution     int64  `gorm:"column:execution"`
			AssignedTo    string `gorm:"column:assignedTo"`
			ProjectName   string `gorm:"column:project_name"`
			ExecutionName string `gorm:"column:execution_name"`
		}
		var list []bRow
		_ = r.db.WithContext(ctx).Table("zt_bug AS b").
			Select("b.id, b.title, b.status, b.project, b.execution, b.assignedTo, COALESCE(p.name, '') AS project_name, COALESCE(e.name, '') AS execution_name").
			Joins("LEFT JOIN zt_project AS p ON p.id = b.project").
			Joins("LEFT JOIN zt_project AS e ON e.id = b.execution").
			Where("b.id IN ?", bugIDs).
			Scan(&list).Error
		for _, b := range list {
			out[fmt.Sprintf("bug:%d", b.ID)] = doneObjectContext{
				Title:         b.Title,
				Status:        b.Status,
				ProjectID:     b.Project,
				ProjectName:   b.ProjectName,
				ExecutionID:   b.Execution,
				ExecutionName: b.ExecutionName,
				CurrentOwner:  b.AssignedTo,
			}
		}
	}

	// 5. Todo
	if len(todoIDs) > 0 {
		type tdRow struct {
			ID     int64  `gorm:"column:id"`
			Name   string `gorm:"column:name"`
			Status string `gorm:"column:status"`
		}
		var list []tdRow
		_ = r.db.WithContext(ctx).Table("zt_todo").
			Select("id, name, status").
			Where("id IN ?", todoIDs).
			Scan(&list).Error
		for _, td := range list {
			out[fmt.Sprintf("todo:%d", td.ID)] = doneObjectContext{
				Title:  td.Name,
				Status: td.Status,
			}
		}
	}

	// 6. 审批对象。禅道的章程和建设指引标题来自关联项目，objectID 仍保留对象自身编号。
	if len(charterIDs) > 0 {
		type cRow struct {
			ID          int64  `gorm:"column:id"`
			Project     int64  `gorm:"column:project"`
			ProjectName string `gorm:"column:project_name"`
		}
		var list []cRow
		_ = r.db.WithContext(ctx).Table("zt_charter AS c").
			Select("c.id, c.project, COALESCE(p.name, '') AS project_name").
			Joins("LEFT JOIN zt_project AS p ON p.id = c.project AND p.deleted = '0'").
			Where("c.id IN ? AND c.deleted = '0'", charterIDs).Scan(&list).Error
		for _, c := range list {
			title := "项目章程"
			if c.ProjectName != "" {
				title = c.ProjectName + " / 项目章程"
			}
			out[fmt.Sprintf("charter:%d", c.ID)] = doneObjectContext{Title: title, ProjectID: c.Project, ProjectName: c.ProjectName}
		}
	}
	if len(planchangeIDs) > 0 {
		type pRow struct {
			ID          int64  `gorm:"column:id"`
			Title       string `gorm:"column:title"`
			Project     int64  `gorm:"column:project"`
			ProjectName string `gorm:"column:project_name"`
		}
		var list []pRow
		_ = r.db.WithContext(ctx).Table("zt_planchange AS pc").Select("pc.id, pc.title, pc.project, COALESCE(p.name, '') AS project_name").Joins("LEFT JOIN zt_project AS p ON p.id = pc.project AND p.deleted = '0'").Where("pc.id IN ?", planchangeIDs).Scan(&list).Error
		for _, p := range list {
			title := p.Title
			if title == "" {
				title = "计划变更"
			}
			out[fmt.Sprintf("planchange:%d", p.ID)] = doneObjectContext{Title: title, ProjectID: p.Project, ProjectName: p.ProjectName}
		}
	}
	if len(buildguidelineIDs) > 0 {
		type bRow struct {
			ID          int64  `gorm:"column:id"`
			Project     int64  `gorm:"column:projectID"`
			ProjectName string `gorm:"column:project_name"`
		}
		var list []bRow
		_ = r.db.WithContext(ctx).Table("zt_projectbuildguide AS bg").Select("bg.id, bg.projectID, COALESCE(p.name, '') AS project_name").Joins("LEFT JOIN zt_project AS p ON p.id = bg.projectID AND p.deleted = '0'").Where("bg.id IN ? AND bg.deleted = '0'", buildguidelineIDs).Scan(&list).Error
		for _, b := range list {
			title := "项目建设指引"
			if b.ProjectName != "" {
				title = b.ProjectName + " / 项目建设指引"
			}
			out[fmt.Sprintf("buildguideline:%d", b.ID)] = doneObjectContext{Title: title, ProjectID: b.Project, ProjectName: b.ProjectName}
		}
	}
	if len(reviewIDs) > 0 {
		type rvRow struct {
			ID          int64  `gorm:"column:id"`
			Title       string `gorm:"column:title"`
			Project     int64  `gorm:"column:project"`
			ProjectName string `gorm:"column:project_name"`
		}
		var list []rvRow
		_ = r.db.WithContext(ctx).Table("zt_review AS rv").Select("rv.id, rv.title, rv.project, COALESCE(p.name, '') AS project_name").Joins("LEFT JOIN zt_project AS p ON p.id = rv.project AND p.deleted = '0'").Where("rv.id IN ? AND rv.deleted = '0'", reviewIDs).Scan(&list).Error
		for _, rv := range list {
			title := rv.Title
			if title == "" {
				title = "项目评审"
			}
			out[fmt.Sprintf("review:%d", rv.ID)] = doneObjectContext{Title: title, ProjectID: rv.Project, ProjectName: rv.ProjectName}
		}
	}
	if len(caseIDs) > 0 {
		type caRow struct {
			ID          int64  `gorm:"column:id"`
			Title       string `gorm:"column:title"`
			Project     int64  `gorm:"column:project"`
			ProjectName string `gorm:"column:project_name"`
		}
		var list []caRow
		_ = r.db.WithContext(ctx).Table("zt_case AS ca").Select("ca.id, ca.title, ca.project, COALESCE(p.name, '') AS project_name").Joins("LEFT JOIN zt_project AS p ON p.id = ca.project AND p.deleted = '0'").Where("ca.id IN ? AND ca.deleted = '0'", caseIDs).Scan(&list).Error
		for _, ca := range list {
			out[fmt.Sprintf("case:%d", ca.ID)] = doneObjectContext{Title: ca.Title, ProjectID: ca.Project, ProjectName: ca.ProjectName}
		}
	}

	return out
}

// CountDoneFacetCounts 按 11 种操作对象类型聚合已办数量。
func (r *Repo) CountDoneFacetCounts(ctx context.Context, account string) []DoneFacet {
	order := []string{"approval", "demand", "story", "task", "bug", "risk", "issue", "feedback", "release", "build", "todo"}
	labels := map[string]string{
		"approval": "审批",
		"demand":   "业务需求",
		"story":    "研发需求",
		"task":     "任务",
		"bug":      "Bug",
		"risk":     "风险",
		"issue":    "问题",
		"feedback": "反馈",
		"release":  "发布",
		"build":    "构建",
		"todo":     "待办",
	}

	counts := make(map[string]int64)
	if r != nil && r.db != nil && strings.TrimSpace(account) != "" {
		scopeSQL, scopeArgs := buildFormalDoneScopeSQL()
		type fRow struct {
			ObjectType string `gorm:"column:objectType"`
			Cnt        int64  `gorm:"column:cnt"`
		}
		var rows []fRow
		_ = r.db.WithContext(ctx).Table("zt_action AS a").
			Select("a.objectType, COUNT(*) AS cnt").
			Where("a.actor = ?", account).
			Where(scopeSQL, scopeArgs...).
			Group("a.objectType").
			Scan(&rows).Error
		for _, row := range rows {
			counts[row.ObjectType] = row.Cnt
		}
		var approvalCount int64
		_ = r.db.WithContext(ctx).Table("zt_action AS a").
			Where("a.actor = ?", account).
			Where(scopeSQL, scopeArgs...).
			Where(buildApprovalDoneScopeSQL()).
			Count(&approvalCount).Error
		counts["approval"] = approvalCount
	}

	facets := make([]DoneFacet, 0, len(order))
	for _, key := range order {
		facets = append(facets, DoneFacet{
			Key:   key,
			Label: labels[key],
			Count: counts[key],
		})
	}
	return facets
}

// FindDoneProjects 查询当前用户在已办事项中涉及的可选项目列表。
func (r *Repo) FindDoneProjects(ctx context.Context, account string) []DoneMetaOption {
	out := []DoneMetaOption{}
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return out
	}
	type pRow struct {
		ID   int64  `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var rows []pRow
	_ = r.db.WithContext(ctx).Table("zt_project AS p").
		Select("DISTINCT p.id, p.name").
		Where("p.type = 'project' AND p.deleted = '0'").
		Order("p.id DESC").
		Limit(50).
		Scan(&rows).Error
	for _, p := range rows {
		out = append(out, DoneMetaOption{
			Key:   fmt.Sprintf("%d", p.ID),
			Label: p.Name,
		})
	}
	return out
}

// FindDoneActionDetail 查询单个已办动作详情、上下文及邻近时间线。
// 已办动作按 ZenTao zt_action 真实 schema，过滤 deleted='0'。
func (r *Repo) FindDoneActionDetail(ctx context.Context, actionID int64) (*DoneDetailResp, error) {
	if r == nil || r.db == nil || actionID <= 0 {
		return nil, fmt.Errorf("invalid action id")
	}

	var row doneActionDBRow
	if err := r.db.WithContext(ctx).Table("zt_action").
		Where("id = ? AND deleted = ?", actionID, "0").
		Find(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, fmt.Errorf("done action %d not found", actionID)
	}

	ctxs := r.fetchObjectContexts(ctx, []doneActionDBRow{row})
	objCtx := ctxs[fmt.Sprintf("%s:%d", row.ObjectType, row.ObjectID)]
	meta := formalDoneActions[row.ObjectType+":"+row.Action]
	actionLabel := meta.Label
	if actionLabel == "" {
		actionLabel = row.Action
	}

	hists := r.fetchActionHistories(ctx, []int64{actionID})
	_ = hists[actionID]

	item := DoneAction{
		ID:              row.ID,
		Actor:           row.Actor,
		Action:          actionLabel,
		ObjectType:      row.ObjectType,
		ObjectTypeLabel: doneObjectTypeLabel(row.ObjectType),
		ObjectID:        row.ObjectID,
		ObjectName:      objCtx.Title,
		Date:            row.Date.Format("2006-01-02 15:04:05"),
		Result:          meta.Result,
		URL:             objectViewURL(row.ObjectType, uint(row.ObjectID)),
	}
	if row.ObjectType == "charter" || row.ObjectType == "buildguideline" {
		item.URL = zentao.URL(row.ObjectType, "view", fmt.Sprintf("projectID=%d", objCtx.ProjectID))
	}

	// 历史时间线（前后 10 条）
	type tlRow struct {
		ID     int64     `gorm:"column:id"`
		Action string    `gorm:"column:action"`
		Actor  string    `gorm:"column:actor"`
		Date   time.Time `gorm:"column:date"`
	}
	var nearby []tlRow
	_ = r.db.WithContext(ctx).Table("zt_action").
		Select("id, action, actor, date").
		Where("objectType = ? AND objectID = ? AND deleted = ?", row.ObjectType, row.ObjectID, "0").
		Order("id DESC").
		Scan(&nearby).Error

	timeline := make([]DoneDetailTimeline, 0, len(nearby))
	for _, n := range nearby {
		nMeta := formalDoneActions[row.ObjectType+":"+n.Action]
		lbl := nMeta.Label
		if lbl == "" {
			lbl = n.Action
		}
		timeline = append(timeline, DoneDetailTimeline{
			ActionName: lbl,
			ActorName:  n.Actor,
			OccurredAt: n.Date.Format("2006-01-02 15:04:05"),
			IsCurrent:  n.ID == row.ID,
		})
	}

	return &DoneDetailResp{
		Item: item,
		Context: DoneDetailContext{
			ProductName:   objCtx.ProductName,
			ProjectName:   objCtx.ProjectName,
			ExecutionName: objCtx.ExecutionName,
			CurrentStatus: objCtx.Status,
			CurrentOwner:  objCtx.CurrentOwner,
		},
		Timeline: timeline,
	}, nil
}
