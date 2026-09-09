package po

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func homeFocusStoryStageSQL() string {
	var sql strings.Builder
	sql.WriteString("CASE LOWER(TRIM(status))")
	for _, status := range strings.Fields("draft wait active clarified planned projected designed designing developing developed testing tested verified reviewing waitacceptance acceptanced delivering delivered waitdeliver released releasing") {
		fmt.Fprintf(&sql, " WHEN '%s' THEN %d", status, homeFocusStageIndex(homeStoryStage(status)))
	}
	sql.WriteString(" ELSE 99 END")
	return sql.String()
}

// Keep both object kinds in one ordered SQL relation so LIMIT applies after
// merging, including the existing demand-first tie break on equal IDs.
func (r *Repo) focusPageQuery(ctx context.Context, account string, req DemandsReq, reviewIDs []int) *gorm.DB {
	var parts []string
	var args []interface{}
	if req.ObjectType != "story" {
		base := r.applyHomeFocusToolbarFiltersWithReviews(r.homeFocusQueryWithReviews(ctx, account, req, reviewIDs), account, req, reviewIDs)
		rank := "CASE WHEN status = 'wait' AND id IN (SELECT demand FROM zt_demandreview WHERE reviewer = ? AND (result IS NULL OR result = '')) THEN 0 WHEN status IN ('draft', 'refuse') AND (assignedTo = ? OR createdBy = ?) THEN 1 ELSE 2 END"
		stmt := r.db.WithContext(ctx).Table("(?) AS focused", base).
			Select("id, 'demand' AS kind, stage_index, "+rank+" AS action_rank", account, account, account).
			Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
		parts = append(parts, stmt.SQL.String())
		args = append(args, stmt.Vars...)
	}
	if req.ObjectType != "demand" {
		stmt := r.homeFocusStoryQuery(ctx, account, req).Select("id, 'story' AS kind, stage_index, 2 AS action_rank").
			Session(&gorm.Session{DryRun: true}).Find(&[]struct{ ID int }{}).Statement
		parts = append(parts, stmt.SQL.String())
		args = append(args, stmt.Vars...)
	}
	return r.db.WithContext(ctx).Table("("+strings.Join(parts, " UNION ALL ")+") AS focus_objects", args...)
}

func (r *Repo) FindHomeFocus(ctx context.Context, account string, req DemandsReq) ([]itemRef, int, error) {
	if strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	reviewIDs, err := r.FindAccountPendingReviewDemandIDs(ctx, account)
	if err != nil {
		return nil, 0, err
	}
	base := r.focusPageQuery(ctx, account, req, reviewIDs)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID         int
		Kind       string
		StageIndex int
		ActionRank int
	}
	if err := base.Select("id, kind, stage_index, action_rank").
		Order("action_rank ASC, stage_index ASC, id DESC, kind ASC").
		Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	refs := make([]itemRef, 0, len(rows))
	for _, row := range rows {
		stage := ""
		if row.StageIndex > 0 && row.StageIndex < len(valueStreamStages) {
			stage = valueStreamStages[row.StageIndex].status
		}
		refs = append(refs, itemRef{kind: row.Kind, id: row.ID, stageStatus: stage, actionRank: row.ActionRank})
	}
	return refs, int(total), nil
}
