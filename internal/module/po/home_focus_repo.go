package po

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// homeFocusQuery intersects the existing stage scopes with the homepage focus.
// Focus facts currently belong to demands; independent stories have no confirmed
// deadline/blocking/hang mapping in the homepage KPI contract.
func (r *Repo) homeFocusQuery(ctx context.Context, account string, req DemandsReq) *gorm.DB {
	parts := []string{}
	args := []interface{}{}
	today := time.Now().Format("2006-01-02")
	for index, stage := range valueStreamStages {
		if stage.status == "all" || (req.Status != "all" && stage.status != req.Status) {
			continue
		}
		q := r.roleDemandScope(ctx, account, mysqlStageFilters[stage.status])
		switch req.Focus {
		case "today":
			q = q.Where("deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline <= ?", today)
		case "overdue":
			q = q.Where("deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline < ?", today)
		case "blocked":
			q = q.Where("status = ?", "refuse")
		case "suspended":
			q = q.Where("hang = ?", "1")
		}
		var rows []struct{ ID int }
		stmt := q.Select("id, ? AS stage_index", index).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
		parts = append(parts, stmt.SQL.String())
		args = append(args, stmt.Vars...)
	}
	if len(parts) == 0 {
		// 无匹配阶段时返回空候选集，避免非法 SQL。
		return r.db.WithContext(ctx).Table("(SELECT NULL AS id, 0 AS stage_index WHERE 1 = 0) AS candidates").
			Select("id, stage_index")
	}
	return r.db.WithContext(ctx).Table("("+strings.Join(parts, " UNION ALL ")+") AS candidates", args...).
		Select("id, MIN(stage_index) AS stage_index").Group("id")
}

// applyHomeFocusToolbarFilters 把 DemandsReq 的 keyword/objectType/priority/relation
// 透传到 SQL（参数化）。FindHomeFocus 当前只查业务需求，因此：
//   - objectType="story" 直接返回空候选集，避免返回与筛选语义不一致的数据；
//   - keyword 在 id / 名称 / 责任账号上模糊匹配（MySQL LOWER）；
//   - priority 映射到 zt_demand.pri 的 1/2/3/4；前端 p3 含义为 P3+P4；
//   - relation 是 base scope 已经覆盖的"我负责/我配合/我关注"，这里只补默认 all。
//
// 设计依据：本仓库查询默认走 SQL filter → sort → count → 分页
// （AGENTS.md MUST 5 / docs/engineering/database.md）。前端不得再做同语义二次过滤。
func applyHomeFocusToolbarFilters(base *gorm.DB, req DemandsReq) *gorm.DB {
	// objectType = story：FindHomeFocus 当前不查 story；返回空候选集。
	if req.ObjectType == "story" {
		return base.Where("1 = 0")
	}
	if kw := strings.ToLower(strings.TrimSpace(req.Keyword)); kw != "" {
		pattern := "%" + kw + "%"
		base = base.Where(
			"id IN (SELECT id FROM zt_demand WHERE LOWER(CAST(id AS CHAR)) LIKE ? OR LOWER(name) LIKE ?) "+
				"OR id IN (SELECT id FROM zt_demand WHERE LOWER(IFNULL(assignedTo, '')) LIKE ? OR LOWER(IFNULL(QD, '')) LIKE ? OR LOWER(IFNULL(RD, '')) LIKE ?)",
			pattern, pattern, pattern, pattern, pattern,
		)
	}
	switch req.Priority {
	case "p1":
		base = base.Where("id IN (SELECT id FROM zt_demand WHERE pri = '1')")
	case "p2":
		base = base.Where("id IN (SELECT id FROM zt_demand WHERE pri = '2')")
	case "p3":
		base = base.Where("id IN (SELECT id FROM zt_demand WHERE pri IN ('3', '4'))")
	}
	// relation: roleDemandBase 已覆盖 owner/cooperate/watch；当前只接受 all 透传。
	_ = req.Relation
	return base
}

func (r *Repo) FindHomeFocus(ctx context.Context, account string, req DemandsReq) ([]itemRef, int, error) {
	if strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	base := r.homeFocusQuery(ctx, account, req)
	base = applyHomeFocusToolbarFilters(base, req)
	var total int64
	if err := r.db.WithContext(ctx).Table("(?) AS focused", base).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID         int
		StageIndex int
	}
	err := r.db.WithContext(ctx).Table("(?) AS focused", base).
		Select("id, stage_index").Order("stage_index ASC, id DESC").
		Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	refs := make([]itemRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, itemRef{kind: "demand", id: row.ID, stageStatus: valueStreamStages[row.StageIndex].status})
	}
	return refs, int(total), nil
}
