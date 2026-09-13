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
	"workbench/internal/pkg/personlabel"
)

type doneActionDBRow struct {
	ID         int64     `gorm:"column:id"`
	ObjectType string    `gorm:"column:objectType"`
	ObjectID   int64     `gorm:"column:objectID"`
	Action     string    `gorm:"column:action"`
	Actor      string    `gorm:"column:actor"`
	ActorName  string    `gorm:"column:actor_name"`
	Date       time.Time `gorm:"column:date"`
	Extra      string    `gorm:"column:extra"`
	Comment    string    `gorm:"column:comment"`
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
	PoolName      string
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
// 审批对象上下文查询失败向上返回；其余对象类型保持静默降级。
func (r *Repo) fetchObjectContexts(ctx context.Context, rows []doneActionDBRow) (map[string]doneObjectContext, error) {
	out := make(map[string]doneObjectContext, len(rows))
	if r == nil || r.db == nil || len(rows) == 0 {
		return out, nil
	}

	var demandIDs, storyIDs, taskIDs, bugIDs, todoIDs []int64
	var charterIDs, planchangeIDs, buildguidelineIDs, reviewIDs, caseIDs []int64

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

	// 审批对象上下文见 loadApprovalObjectContexts。
	if err := r.loadApprovalObjectContexts(ctx, out,
		charterIDs, planchangeIDs, buildguidelineIDs, reviewIDs, caseIDs); err != nil {
		return nil, err
	}

	// 1. Demand
	if len(demandIDs) > 0 {
		type dRow struct {
			ID          int64  `gorm:"column:id"`
			Name        string `gorm:"column:name"`
			Status      string `gorm:"column:status"`
			AssignedTo  string `gorm:"column:assignedTo"`
			ProductName string `gorm:"column:product_name"`
			PoolName    string `gorm:"column:pool_name"`
		}
		var list []dRow
		_ = r.db.WithContext(ctx).Table("zt_demand AS d").
			Select("d.id, d.name, d.status, d.assignedTo, COALESCE(p.name, '') AS product_name, COALESCE(dp.name, '') AS pool_name").
			Joins("LEFT JOIN zt_product AS p ON p.id = d.product").
			Joins("LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0'").
			Where("d.id IN ?", demandIDs).
			Scan(&list).Error
		for _, d := range list {
			out[fmt.Sprintf("demand:%d", d.ID)] = doneObjectContext{
				Title:        d.Name,
				Status:       d.Status,
				ProductName:  d.ProductName,
				PoolName:     d.PoolName,
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
			PoolName    string `gorm:"column:pool_name"`
		}
		var list []sRow
		_ = r.db.WithContext(ctx).Table("zt_story AS s").
			Select("s.id, s.title, s.status, s.assignedTo, COALESCE(p.name, '') AS product_name, COALESCE(dp.name, '') AS pool_name").
			Joins("LEFT JOIN zt_product AS p ON p.id = s.product").
			Joins("LEFT JOIN zt_demand AS d ON d.id = s.fromDemand AND d.deleted = '0'").
			Joins("LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0'").
			Where("s.id IN ?", storyIDs).
			Scan(&list).Error
		for _, s := range list {
			out[fmt.Sprintf("story:%d", s.ID)] = doneObjectContext{
				Title:        s.Title,
				Status:       s.Status,
				ProductName:  s.ProductName,
				PoolName:     s.PoolName,
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
			PoolName      string `gorm:"column:pool_name"`
		}
		var list []tRow
		_ = r.db.WithContext(ctx).Table("zt_task AS t").
			Select("t.id, t.name, t.status, t.project, t.execution, t.assignedTo, COALESCE(p.name, '') AS project_name, COALESCE(e.name, '') AS execution_name, COALESCE(dp.name, '') AS pool_name").
			Joins("LEFT JOIN zt_project AS p ON p.id = t.project").
			Joins("LEFT JOIN zt_project AS e ON e.id = t.execution").
			Joins("LEFT JOIN zt_story AS s ON s.id = t.story AND s.deleted = '0'").
			Joins("LEFT JOIN zt_demand AS d ON d.id = s.fromDemand AND d.deleted = '0'").
			Joins("LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0'").
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
				PoolName:      t.PoolName,
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

	// 6. 审批对象已迁移到 repodone_enrich_approval.go，调用 loadApprovalObjectContexts 完成。

	return out, nil
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
	if err := r.db.WithContext(ctx).Table("zt_action AS a").
		Select("a.id, a.objectType, a.objectID, a.action, a.actor, a.date, a.extra, a.comment, COALESCE(NULLIF(u.realname, ''), a.actor) AS actor_name").
		Joins("LEFT JOIN zt_user u ON u.account = a.actor").
		Where("a.id = ?", actionID).
		Find(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, fmt.Errorf("done action %d not found", actionID)
	}

	ctxs, err := r.fetchObjectContexts(ctx, []doneActionDBRow{row})
	if err != nil {
		return nil, err
	}
	objCtx := ctxs[fmt.Sprintf("%s:%d", row.ObjectType, row.ObjectID)]
	meta := formalDoneActions[row.ObjectType+":"+row.Action]
	actionLabel := doneHistoryActionLabel(row.ObjectType, row.Action)
	var histories []doneHistoryRow
	if err := r.db.WithContext(ctx).Table("zt_history").Select("action, field, `old`, `new`").Where("action IN ? AND field IN ('status', 'stage')", []int64{actionID}).Order("id ASC").Scan(&histories).Error; err != nil {
		return nil, err
	}
	var before, after string
	for _, h := range histories {
		if before == "" || h.Field == "status" {
			before, after = h.Old, h.New
		}
	}
	resultCode, resultText := resolveDoneActionResult(row.Action, row.ObjectType, row.Extra, meta.Result)

	item := DoneAction{
		ID:             row.ID,
		SourceActionId: row.ID, SourceSystem: "zentao", ActionName: actionLabel, ActionKey: row.Action,
		ActorName: personlabel.Format(row.Actor, row.ActorName), ObjectCode: doneObjectCode(row.ObjectType, row.ObjectID), ObjectTitle: objCtx.Title,
		BeforeStatus: before, AfterStatus: after, CurrentStatus: objCtx.Status, NextOwnerName: objCtx.CurrentOwner,
		Actor:           row.Actor,
		Action:          actionLabel,
		ObjectType:      row.ObjectType,
		ObjectTypeLabel: doneObjectTypeLabel(row.ObjectType),
		ObjectID:        row.ObjectID,
		ObjectName:      objCtx.Title,
		Date:            row.Date.Format("2006-01-02 15:04:05"),
		Result:          resultCode,
		ResultCode:      resultCode,
		ResultText:      resultText,
		URL:             objectViewURLWithProject(row.ObjectType, uint(row.ObjectID), uint(objCtx.ProjectID)),
	}

	// 历史时间线（前后 10 条）
	type tlRow struct {
		ID        int64     `gorm:"column:id"`
		Action    string    `gorm:"column:action"`
		Actor     string    `gorm:"column:actor"`
		ActorName string    `gorm:"column:actor_name"`
		Date      time.Time `gorm:"column:date"`
	}
	var nearby []tlRow
	err = r.db.WithContext(ctx).Table("zt_action AS a").
		Select("a.id, a.action, a.actor, a.date, COALESCE(NULLIF(u.realname, ''), a.actor) AS actor_name").
		Joins("LEFT JOIN zt_user u ON u.account = a.actor").
		Where("a.objectType = ? AND a.objectID = ? AND a.id <= ?", row.ObjectType, row.ObjectID, actionID).
		Order("a.id DESC").Limit(20).Scan(&nearby).Error
	if err != nil {
		return nil, err
	}

	timeline := make([]DoneDetailTimeline, 0, len(nearby))
	for _, n := range nearby {
		lbl := doneHistoryActionLabel(row.ObjectType, n.Action)
		timeline = append(timeline, DoneDetailTimeline{
			ActionName: lbl,
			ActorName:  personlabel.Format(n.Actor, n.ActorName),
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
			PoolName:      objCtx.PoolName,
			CurrentStatus: objCtx.Status,
			CurrentOwner:  objCtx.CurrentOwner,
		},
		Timeline: timeline,
	}, nil
}
