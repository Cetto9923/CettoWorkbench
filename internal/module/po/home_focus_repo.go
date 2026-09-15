package po

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

// FindAccountPendingReviewDemandIDs 快速查询当前账号待评审的 demand ID 列表（~9ms 单次全表扫描），
// 替代逐行在子查询中扫描 zt_demandreview。
func (r *Repo) FindAccountPendingReviewDemandIDs(ctx context.Context, account string) ([]int, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var ids []int
	err := r.db.WithContext(ctx).Table("zt_demandreview").
		Where("reviewer = ? AND (result IS NULL OR result = '')", account).
		Pluck("demand", &ids).Error
	return ids, err
}

// homeFocusQuery intersects the existing stage scopes with the homepage focus.
func (r *Repo) homeFocusQuery(ctx context.Context, account string, req DemandsReq) *gorm.DB {
	var reviewIDs []int
	if req.Focus == "my_action" {
		reviewIDs, _ = r.FindAccountPendingReviewDemandIDs(ctx, account)
	}
	return r.homeFocusQueryWithReviews(ctx, account, req, reviewIDs)
}

func (r *Repo) homeFocusQueryWithReviews(ctx context.Context, account string, req DemandsReq, reviewIDs []int) *gorm.DB {
	parts := []string{}
	args := []interface{}{}
	today := time.Now().Format("2006-01-02")
	for index, stage := range valueStreamStages {
		if stage.status == "all" || (req.Status != "all" && stage.status != req.Status) {
			continue
		}
		q := r.roleDemandScope(ctx, account, mysqlStageFilters[stage.status])
		if req.Status == "all" {
			q = r.roleDemandBase(ctx, account)
		}
		if isParticipateScheduleRequest(req) {
			if req.Status == "schedule" {
				q = q.Where("1 = 0")
			} else if req.Status == "all" {
				q = q.Where("NOT (status = ? AND "+scheduleIncompleteDemandSQL()+")", "clarified")
			}
		}
		switch req.Focus {
		case "my_action":
			// 只把当前价值流步骤的正式办理人视为“待我处理”；创建人、评审人和关注人不是当前负责人。
			where, whereArgs := currentHandlerDemandWhereWithReviews(account, reviewIDs)
			q = q.Where(where, whereArgs...)
		case "today":
			q = q.Where("deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline <= ?", today)
		case "overdue":
			q = q.Where("deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline < ?", today)
		case "blocked":
			q = q.Where(`status = ? OR (
				 developFinish IS NOT NULL AND developFinish != '0000-00-00' AND developFinish <= ?
				AND (managerReviewers IS NOT NULL AND managerReviewers <> '' OR EXISTS (
					SELECT 1 FROM zt_demandmanagerreview mr
					WHERE mr.demand = zt_demand.id
				))
				AND COALESCE(isManagerReview, '') NOT IN ('pass', 'passed')
			)`, "refuse", today)
		case "suspended":
			q = q.Where("hang = ?", "1")
		}
		var rows []struct{ ID int }
		stmt := q.Select("id, status, assignedTo, createdBy, ? AS stage_index, "+demandDurationDaysSQLExpr("zt_demand.createdDate")+" AS duration_days", index).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
		if req.Status == "all" {
			stageSQL, stageArgs := r.demandStageCase(ctx, account)
			stmt = q.Select("id, status, assignedTo, createdBy, "+stageSQL+" AS stage_index, "+demandDurationDaysSQLExpr("zt_demand.createdDate")+" AS duration_days", stageArgs...).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
			parts = append(parts, "SELECT id, status, assignedTo, createdBy, stage_index, duration_days FROM ("+stmt.SQL.String()+") AS demand_stages WHERE stage_index IS NOT NULL")
			args = append(args, stmt.Vars...)
			break
		}
		parts = append(parts, stmt.SQL.String())
		args = append(args, stmt.Vars...)
	}
	if len(parts) == 0 {
		// 无匹配阶段时返回空候选集，避免非法 SQL。
		return r.db.WithContext(ctx).Table("(SELECT NULL AS id, NULL AS status, NULL AS assignedTo, NULL AS createdBy, 0 AS stage_index, NULL AS duration_days WHERE 1 = 0) AS candidates").
			Select("id, status, assignedTo, createdBy, stage_index, duration_days")
	}
	return r.db.WithContext(ctx).Table("("+strings.Join(parts, " UNION ALL ")+") AS candidates", args...).
		Select("id, MAX(status) AS status, MAX(assignedTo) AS assignedTo, MAX(createdBy) AS createdBy, MIN(stage_index) AS stage_index, MAX(duration_days) AS duration_days").Group("id")
}

// applyHomeFocusToolbarFilters 把 DemandsReq 的 keyword/objectType/priority/relation
// 透传到业务需求 SQL（研发需求由 findHomeFocusStoryRefs 单独使用同等过滤）。
func applyHomeFocusToolbarFilters(base *gorm.DB, account string, req DemandsReq) *gorm.DB {
	where, args := currentHandlerDemandWhere(account)
	return applyHomeFocusToolbarFiltersWithClause(base, account, req, where, args)
}

func (r *Repo) applyHomeFocusToolbarFiltersWithReviews(base *gorm.DB, account string, req DemandsReq, reviewIDs []int) *gorm.DB {
	where, args := currentHandlerDemandWhereWithReviews(account, reviewIDs)
	return applyHomeFocusToolbarFiltersWithClause(base, account, req, where, args)
}

func applyHomeFocusToolbarFiltersWithClause(base *gorm.DB, account string, req DemandsReq, where string, args []interface{}) *gorm.DB {
	if kw := strings.ToLower(strings.TrimSpace(req.Keyword)); kw != "" {
		pattern := "%" + kw + "%"
		base = base.Where(
			"id IN (SELECT id FROM zt_demand WHERE LOWER(CAST(id AS CHAR)) LIKE ? OR LOWER(name) LIKE ? OR ("+currentHandlerDemandKeywordWhere()+"))",
			pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern,
		)
	}
	switch strings.ToLower(strings.TrimSpace(req.Priority)) {
	case "p1":
		base = base.Where("id IN (SELECT id FROM zt_demand WHERE pri = '1')")
	case "p2":
		base = base.Where("id IN (SELECT id FROM zt_demand WHERE pri = '2')")
	case "p3":
		base = base.Where("id IN (SELECT id FROM zt_demand WHERE pri IN ('3', '4'))")
	}
	switch req.Relation {
	case "lead":
		if account == "" {
			base = base.Where("1 = 0")
		} else {
			base = base.Where("id IN (SELECT id FROM zt_demand WHERE BRA = ?)", account)
		}
	case "participate":
		if account == "" {
			base = base.Where("1 = 0")
		} else {
			base = base.Where(`id IN (
				SELECT d.id
				FROM zt_demand d
				WHERE d.pool IS NOT NULL AND d.pool <> 0
				  AND EXISTS (
					SELECT 1 FROM zt_demandpool dp
					WHERE dp.id = d.pool AND dp.deleted = '0'
				  )
				  AND (d.BRA <> ? OR d.BRA IS NULL OR d.BRA = '')
				  AND EXISTS (
					SELECT 1 FROM zt_demandclarify dc
					WHERE dc.demand = d.id
					  AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0
				  )
			)`, account, account)
		}
	case "handling":
		base = base.Where("id IN (SELECT id FROM zt_demand WHERE "+where+")", args...)
	case "following":
		base = base.Where("id NOT IN (SELECT id FROM zt_demand WHERE "+where+")", args...)
	}
	return base
}

// CountHomeFocus 只计算符合 focus 范围的条目总数，供首页 KPI 或分页计数使用。
func (r *Repo) CountHomeFocus(ctx context.Context, account string, req DemandsReq) (int64, error) {
	if strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var total int64
	reviewIDs, _ := r.FindAccountPendingReviewDemandIDs(ctx, account)
	if req.ObjectType != "story" {
		base := r.homeFocusQueryWithReviews(ctx, account, req, reviewIDs)
		base = r.applyHomeFocusToolbarFiltersWithReviews(base, account, req, reviewIDs)
		if err := r.db.WithContext(ctx).Table("(?) AS focused", base).Count(&total).Error; err != nil {
			return 0, err
		}
	}
	if req.ObjectType != "demand" {
		storyCount, storyErr := r.countHomeFocusStories(ctx, account, req)
		if storyErr != nil {
			return 0, storyErr
		}
		total += storyCount
	}
	return total, nil
}

// findHomeFocusStoryRefs 复用首页正式焦点的日期/挂起规则，为独立研发需求提供焦点列表。
// 研发需求目前只在排期、交付及待我处理等有明确工作台责任的场景进入首页。
func (r *Repo) homeFocusStoryQuery(ctx context.Context, account string, req DemandsReq) *gorm.DB {
	participate := isParticipateScheduleRequest(req)
	q := r.db.WithContext(ctx).Table("zt_story").Where("deleted = ? AND status <> ? AND IFNULL(sourceType, '') <> ?", "0", "closed", "demandpool")
	if participate {
		q = r.participateScheduleStoryScope(ctx, account)
	}
	if req.ObjectType == "story" || req.Status == "all" || req.Status == "schedule" {
		q = q.Where("((status IN ? AND (developFinish IS NULL OR developFinish = '0000-00-00' OR testFinish IS NULL OR testFinish = '0000-00-00')) OR (status IN ? AND deliverDate IS NOT NULL AND deliverDate != '0000-00-00'))", []string{"draft", "wait", "active", "clarified", "planned", "developing"}, []string{"acceptanced", "waitdeliver", "released"})
	}
	today := time.Now().Format("2006-01-02")
	switch req.Focus {
	case "today":
		q = q.Where("COALESCE(NULLIF(developFinish, '0000-00-00'), NULLIF(testFinish, '0000-00-00'), NULLIF(deliverDate, '0000-00-00')) <= ?", today)
	case "overdue":
		q = q.Where("COALESCE(NULLIF(developFinish, '0000-00-00'), NULLIF(testFinish, '0000-00-00'), NULLIF(deliverDate, '0000-00-00')) < ?", today)
	case "blocked":
		q = q.Where("status = ?", "refuse")
	case "suspended":
		// 禅道研发需求（zt_story）无 hang 字段，挂起事实仅适用于业务需求。
		q = q.Where("1 = 0")
	case "my_action":
		// assignedTo 是研发需求的正式办理责任字段；不把 openedBy/watch 视为待办。
		if !participate {
			q = q.Where("assignedTo = ?", account)
		}
	}
	if participate {
		q = applyParticipateStoryToolbarFilters(q, account, req)
	} else {
		q = applyStoryToolbarFilters(q, account, req)
	}
	stageSQL := homeFocusStoryStageSQL()
	base := q.Select("id, " + stageSQL + " AS stage_index")
	result := r.db.WithContext(ctx).Table("(?) AS focused_stories", base)
	if req.Status != "all" && req.Status != "" {
		result = result.Where("stage_index = ?", homeFocusStageIndex(req.Status))
	}
	return result
}

// homeStoryStage maps zt_story.status onto homepage value-stream stages.
// Independent stories have no clarify step; early statuses belong in schedule
// (aligned with deriveStoryStageKey: active/wait/planned → StageSchedule).
func homeStoryStage(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "draft", "wait", "active", "clarified", "planned", "projected", "designed", "designing":
		return "schedule"
	case "developing", "developed":
		return "developing"
	case "testing", "tested", "verified", "reviewing":
		return "testing"
	case "waitacceptance":
		return "waitacceptance"
	case "acceptanced", "delivering", "delivered", "waitdeliver":
		return "acceptanced"
	case "released", "releasing":
		return "publish"
	default:
		return ""
	}
}

func homeFocusStageIndex(status string) int {
	for i, stage := range valueStreamStages {
		if stage.status == status {
			return i
		}
	}
	return 99
}

func (r *Repo) countHomeFocusStories(ctx context.Context, account string, req DemandsReq) (int64, error) {
	var total int64
	err := r.homeFocusStoryQuery(ctx, account, req).Count(&total).Error
	return total, err
}

// HomeFocusStageSummary partitions the exact same candidate set used by the
// homepage list. Every object is already assigned its first matching stage by
// homeFocusQuery, so the stage counts add up to the all-card count.
func (r *Repo) HomeFocusStageSummary(ctx context.Context, account string, req DemandsReq) ([]ValueStreamStage, error) {
	stages := make([]ValueStreamStage, len(valueStreamStages))
	for i, stage := range valueStreamStages {
		stages[i] = ValueStreamStage{
			Label:           stage.label,
			Status:          stage.status,
			Valid:           true,
			AvgDurationText: "—",
		}
	}
	if strings.TrimSpace(account) == "" {
		return stages, nil
	}
	req.Status = "all"
	var rows []struct {
		StageIndex    int   `gorm:"column:stage_index"`
		Count         int64 `gorm:"column:count"`
		TotalDuration int64 `gorm:"column:total_duration"`
		DurationCount int64 `gorm:"column:duration_count"`
	}
	if req.ObjectType != "story" {
		reviewIDs, _ := r.FindAccountPendingReviewDemandIDs(ctx, account)
		base := r.applyHomeFocusToolbarFiltersWithReviews(r.homeFocusQueryWithReviews(ctx, account, req, reviewIDs), account, req, reviewIDs)
		if err := r.db.WithContext(ctx).Table("(?) AS focused", base).
			Select("stage_index, COUNT(*) AS count, IFNULL(SUM(duration_days), 0) AS total_duration, COUNT(duration_days) AS duration_count").
			Group("stage_index").Scan(&rows).Error; err != nil {
			return nil, err
		}
	}
	var all int64
	var allDemand int64
	var allDemandDuration int64
	var allDemandDurationCount int64
	for _, row := range rows {
		if row.StageIndex <= 0 || row.StageIndex >= len(stages) {
			continue
		}
		stages[row.StageIndex].Count = row.Count
		stages[row.StageIndex].DemandCount = row.Count
		if row.DurationCount > 0 {
			avg := int(math.Round(float64(row.TotalDuration) / float64(row.DurationCount)))
			if avg < 1 {
				avg = 1
			}
			stages[row.StageIndex].AvgDurationDays = avg
			stages[row.StageIndex].AvgDurationText = fmt.Sprintf("均%d天", avg)
		} else {
			stages[row.StageIndex].AvgDurationDays = 0
			stages[row.StageIndex].AvgDurationText = "—"
		}
		all += row.Count
		allDemand += row.Count
		allDemandDuration += row.TotalDuration
		allDemandDurationCount += row.DurationCount
	}
	if req.ObjectType != "demand" {
		var storyRows []struct {
			StageIndex int
			Count      int64
		}
		if err := r.homeFocusStoryQuery(ctx, account, req).Select("stage_index, COUNT(*) AS count").Group("stage_index").Scan(&storyRows).Error; err != nil {
			return nil, err
		}
		for _, row := range storyRows {
			idx := row.StageIndex
			if idx < 0 || idx >= len(stages) {
				continue
			}
			stages[idx].Count += row.Count
			stages[idx].StoryCount += row.Count
			all += row.Count
		}
	}
	stages[0].Count = all
	stages[0].DemandCount = allDemand
	stages[0].StoryCount = all - allDemand
	if allDemandDurationCount > 0 {
		avg := int(math.Round(float64(allDemandDuration) / float64(allDemandDurationCount)))
		if avg < 1 {
			avg = 1
		}
		stages[0].AvgDurationDays = avg
		stages[0].AvgDurationText = fmt.Sprintf("均%d天", avg)
	} else {
		stages[0].AvgDurationDays = 0
		stages[0].AvgDurationText = "—"
	}
	return stages, nil
}
