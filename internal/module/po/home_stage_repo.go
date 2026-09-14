package po

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type stageCount struct {
	StageIndex int
	Kind       string
	Count      int64
}

func (r *Repo) acceptRefsPaged(ctx context.Context, account string, req DemandsReq) ([]itemRef, int, error) {
	base := r.roleDemandScope(ctx, account, mysqlStageFilters["accept"])
	var reviewIDs []int
	if req.Relation == "handling" || req.Relation == "following" {
		reviewIDs, _ = r.FindAccountPendingReviewDemandIDs(ctx, account)
	}
	base = r.applyHomeFocusToolbarFiltersWithReviews(base, account, req, reviewIDs)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct{ ID int }
	err := base.Select("id, CASE WHEN id IN (SELECT demand FROM zt_demandreview WHERE reviewer = ? AND (result = '' OR result IS NULL)) THEN 0 ELSE 1 END AS review_rank", account).
		Order("review_rank ASC, id DESC").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	refs := make([]itemRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, itemRef{kind: "demand", id: row.ID, stageStatus: "accept"})
	}
	return refs, int(total), nil
}

func (r *Repo) countStageRefs(ctx context.Context, account string) ([]stageCount, error) {
	var rows []stageCount
	base := r.allStageRefQuery(ctx, account, DemandsReq{})
	err := r.db.WithContext(ctx).Table("(?) AS all_stages", base).
		Select("stage_index, kind, COUNT(*) AS count").Group("stage_index, kind").Scan(&rows).Error
	return rows, err
}

// applyStoryToolbarFilters 把 DemandsReq 的 keyword/priority/relation 应用到研发需求查询。
func applyStoryToolbarFilters(q *gorm.DB, account string, req DemandsReq) *gorm.DB {
	if kw := strings.ToLower(strings.TrimSpace(req.Keyword)); kw != "" {
		pattern := "%" + kw + "%"
		q = q.Where("(LOWER(CAST(id AS CHAR)) LIKE ? OR LOWER(title) LIKE ? OR LOWER(IFNULL(assignedTo, '')) LIKE ?)", pattern, pattern, pattern)
	}
	switch strings.ToLower(strings.TrimSpace(req.Priority)) {
	case "p1":
		q = q.Where("pri = ?", 1)
	case "p2":
		q = q.Where("pri = ?", 2)
	case "p3":
		q = q.Where("pri IN ?", []int{3, 4})
	}
	switch req.Relation {
	case "lead", "handling":
		if account == "" {
			q = q.Where("1 = 0")
		} else {
			q = q.Where("assignedTo = ?", account)
		}
	case "participate":
		// 研发需求没有业务需求维度的“配合”关系；无业务需求且指派给
		// 当前用户的研发需求仍由 lead/handling 归入“我负责”。
		q = q.Where("1 = 0")
	case "following":
		if account == "" {
			q = q.Where("1 = 0")
		} else {
			q = q.Where("assignedTo <> ? OR assignedTo IS NULL OR assignedTo = ''", account)
		}
	}
	return q
}

// allStageRefQuery keeps demand and story identities separate during deduplication.
func (r *Repo) allStageRefQuery(ctx context.Context, account string, req DemandsReq) *gorm.DB {
	includeDemand := req.ObjectType != "story"
	includeStory := req.ObjectType != "demand"
	parts := make([]string, 0, len(valueStreamStages)+2)
	args := make([]interface{}, 0)
	if includeDemand {
		stageSQL, stageArgs := r.demandStageCase(ctx, account)
		demandBase := r.roleDemandBase(ctx, account)
		var reviewIDs []int
		if req.Relation == "handling" || req.Relation == "following" {
			reviewIDs, _ = r.FindAccountPendingReviewDemandIDs(ctx, account)
		}
		demandBase = r.applyHomeFocusToolbarFiltersWithReviews(demandBase, account, req, reviewIDs)
		stmt := demandBase.Select("id, 'demand' AS kind, "+stageSQL+" AS stage_index, 0 AS kind_rank", stageArgs...).Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
		parts = append(parts, "SELECT id, kind, stage_index, kind_rank FROM ("+stmt.SQL.String()+") AS demand_stages WHERE stage_index IS NOT NULL")
		args = append(args, stmt.Vars...)
	}
	for index, stage := range valueStreamStages {
		if stage.status == "all" {
			continue
		}
		filter, ok := mysqlStageFilters[stage.status]
		if !ok {
			continue
		}
		if includeStory && filter.scheduleIncomplete {
			stmt := applyStoryToolbarFilters(r.scheduleStoryScope(ctx, account), account, req).
				Select("id, 'story' AS kind, ? AS stage_index, 1 AS kind_rank", index).
				Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
			parts = append(parts, stmt.SQL.String())
			args = append(args, stmt.Vars...)
		}
		if includeStory && filter.deliverStories {
			stmt := applyStoryToolbarFilters(r.deliverStoryScope(ctx, account), account, req).
				Select("id, 'story' AS kind, ? AS stage_index, 1 AS kind_rank", index).
				Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
			parts = append(parts, stmt.SQL.String())
			args = append(args, stmt.Vars...)
		}
	}
	if len(parts) == 0 {
		return r.db.WithContext(ctx).Table("(SELECT NULL AS kind, NULL AS id, 0 AS stage_index, 0 AS kind_rank WHERE 1 = 0) AS stage_candidates").
			Select("kind, id, stage_index, kind_rank")
	}
	return r.db.WithContext(ctx).Table("("+strings.Join(parts, " UNION ALL ")+") AS stage_candidates", args...).
		Select("kind, id, MIN(stage_index) AS stage_index, MIN(kind_rank) AS kind_rank").
		Group("kind, id")
}

// roleDemandScopeWithFilters 为业需查询施加阶段过滤与工具栏过滤（priority/keyword/relation）
func (r *Repo) roleDemandScopeWithFilters(ctx context.Context, account string, filter mysqlStageFilter, req DemandsReq) *gorm.DB {
	base := r.roleDemandScope(ctx, account, filter)
	var reviewIDs []int
	if req.Relation == "handling" || req.Relation == "following" {
		reviewIDs, _ = r.FindAccountPendingReviewDemandIDs(ctx, account)
	}
	return r.applyHomeFocusToolbarFiltersWithReviews(base, account, req, reviewIDs)
}

// CountRoleDemandsWithFilters 按阶段过滤条件与工具栏筛选统计业需数量。
func (r *Repo) CountRoleDemandsWithFilters(ctx context.Context, account string, filter mysqlStageFilter, req DemandsReq) (int64, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return 0, nil
	}
	var total int64
	err := r.roleDemandScopeWithFilters(ctx, account, filter, req).Count(&total).Error
	return total, err
}

// FindRoleDemandsPagedWithFilters 按阶段过滤条件与工具栏筛选分页查询业需列表。
func (r *Repo) FindRoleDemandsPagedWithFilters(ctx context.Context, account string, filter mysqlStageFilter, req DemandsReq, offset, limit int) ([]DemandRow, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	var rows []DemandRow
	q := r.roleDemandScopeWithFilters(ctx, account, filter, req).
		Select(`zt_demand.id, zt_demand.name, zt_demand.pri, zt_demand.status, zt_demand.hang,
			zt_demand.assignedTo, zt_demand.QD, zt_demand.RD, zt_demand.BRA,
			clarify_pm.PM AS pm`).
		Joins(`LEFT JOIN (
			SELECT demand, GROUP_CONCAT(PM) AS PM
			FROM zt_demandclarify
			WHERE PM IS NOT NULL AND PM <> ''
			GROUP BY demand
		) AS clarify_pm ON clarify_pm.demand = zt_demand.id`).
		Order("zt_demand.id DESC")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

// FindRoleDemandIDsWithFilters 按阶段过滤条件与工具栏筛选只查业需 ID。
func (r *Repo) FindRoleDemandIDsWithFilters(ctx context.Context, account string, filter mysqlStageFilter, req DemandsReq) ([]int, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	var ids []int
	err := r.roleDemandScopeWithFilters(ctx, account, filter, req).
		Order("zt_demand.id DESC").
		Pluck("zt_demand.id", &ids).Error
	return ids, err
}

// FindScheduleStoryIDsWithFilters 查询排期阶段符合工具栏筛选的独立研发需求 ID。
func (r *Repo) FindScheduleStoryIDsWithFilters(ctx context.Context, account string, req DemandsReq) ([]int, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var ids []int
	err := applyStoryToolbarFilters(r.scheduleStoryScope(ctx, account), account, req).
		Order("id DESC").
		Pluck("id", &ids).Error
	return ids, err
}

// FindDeliverStoryIDsWithFilters 查询交付阶段符合工具栏筛选的独立研发需求 ID。
func (r *Repo) FindDeliverStoryIDsWithFilters(ctx context.Context, account string, req DemandsReq) ([]int, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var ids []int
	err := applyStoryToolbarFilters(r.deliverStoryScope(ctx, account), account, req).
		Order("id DESC").
		Pluck("id", &ids).Error
	return ids, err
}

// FindStageMixedRefsPaged SQL-paginates the schedule/deliver mix of demand + story
// IDs (demands first, then stories; each by id DESC) without materializing the
// full ID list in process memory.
func (r *Repo) FindStageMixedRefsPaged(ctx context.Context, account, stageStatus string, filter mysqlStageFilter, req DemandsReq) ([]itemRef, int, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	includeDemand := req.ObjectType != "story"
	includeStory := req.ObjectType != "demand"
	parts := make([]string, 0, 2)
	args := make([]interface{}, 0)

	if includeDemand && filterReady(account, filter) {
		stmt := r.roleDemandScopeWithFilters(ctx, account, filter, req).
			Select("zt_demand.id AS id, 'demand' AS kind, 0 AS kind_rank").
			Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
		parts = append(parts, stmt.SQL.String())
		args = append(args, stmt.Vars...)
	}
	if includeStory && filter.scheduleIncomplete {
		stmt := applyStoryToolbarFilters(r.scheduleStoryScope(ctx, account), account, req).
			Select("id AS id, 'story' AS kind, 1 AS kind_rank").
			Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
		parts = append(parts, stmt.SQL.String())
		args = append(args, stmt.Vars...)
	}
	if includeStory && filter.deliverStories {
		stmt := applyStoryToolbarFilters(r.deliverStoryScope(ctx, account), account, req).
			Select("id AS id, 'story' AS kind, 1 AS kind_rank").
			Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
		parts = append(parts, stmt.SQL.String())
		args = append(args, stmt.Vars...)
	}
	if len(parts) == 0 {
		return nil, 0, nil
	}

	unionSQL := strings.Join(parts, " UNION ALL ")
	var total int64
	if err := r.db.WithContext(ctx).
		Raw("SELECT COUNT(*) FROM ("+unionSQL+") AS stage_mixed", args...).
		Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (req.Page - 1) * req.PageSize
	if total == 0 || int64(offset) >= total {
		return nil, int(total), nil
	}
	var rows []struct {
		ID   int    `gorm:"column:id"`
		Kind string `gorm:"column:kind"`
	}
	pageArgs := append(append([]interface{}{}, args...), req.PageSize, offset)
	if err := r.db.WithContext(ctx).
		Raw("SELECT id, kind FROM ("+unionSQL+") AS stage_mixed ORDER BY kind_rank ASC, id DESC LIMIT ? OFFSET ?", pageArgs...).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	refs := make([]itemRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, itemRef{kind: row.Kind, id: row.ID, stageStatus: stageStatus})
	}
	return refs, int(total), nil
}

// demandStageCase reuses the single-stage predicates in first-match order.
func (r *Repo) demandStageCase(ctx context.Context, account string) (string, []interface{}) {
	var sql strings.Builder
	sql.WriteString("CASE")
	var args []interface{}
	for index, stage := range valueStreamStages {
		filter, ok := mysqlStageFilters[stage.status]
		if !ok {
			continue
		}
		stmt := applyDemandStage(r.db.WithContext(ctx).Table("zt_demand"), account, filter).
			Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
		stmt.SQL.Reset()
		stmt.Vars = nil
		stmt.Clauses["WHERE"].Expression.Build(stmt)
		sql.WriteString(" WHEN (")
		sql.WriteString(stmt.SQL.String())
		fmt.Fprintf(&sql, ") THEN %d", index)
		args = append(args, stmt.Vars...)
	}
	sql.WriteString(" ELSE NULL END")
	return sql.String(), args
}
