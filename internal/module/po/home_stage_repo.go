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

// allStageRefQuery keeps demand and story identities separate during deduplication.
func (r *Repo) allStageRefQuery(ctx context.Context, account string, req DemandsReq) *gorm.DB {
	includeDemand := req.ObjectType != "story"
	includeStory := req.ObjectType != "demand"
	parts := make([]string, 0, len(valueStreamStages)+2)
	args := make([]interface{}, 0)
	if includeDemand {
		stageSQL, stageArgs := r.demandStageCase(ctx, account)
		stmt := r.roleDemandBase(ctx, account).Select("id, 'demand' AS kind, "+stageSQL+" AS stage_index, 0 AS kind_rank", stageArgs...).Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
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
			stmt := r.scheduleStoryScope(ctx, account).
				Select("id, 'story' AS kind, ? AS stage_index, 1 AS kind_rank", index).
				Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
			parts = append(parts, stmt.SQL.String())
			args = append(args, stmt.Vars...)
		}
		if includeStory && filter.deliverStories {
			stmt := r.deliverStoryScope(ctx, account).
				Select("id, 'story' AS kind, ? AS stage_index, 1 AS kind_rank", index).
				Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
			parts = append(parts, stmt.SQL.String())
			args = append(args, stmt.Vars...)
		}
	}
	return r.db.WithContext(ctx).Table("("+strings.Join(parts, " UNION ALL ")+") AS stage_candidates", args...).
		Select("kind, id, MIN(stage_index) AS stage_index, MIN(kind_rank) AS kind_rank").
		Group("kind, id")
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
