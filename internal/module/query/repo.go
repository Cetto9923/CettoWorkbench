// 需求查询只读仓储：查询禅道需求与研发需求，不写本地状态。
package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"workbench/internal/pkg/demandstage"
	zturl "workbench/internal/pkg/zentao"
)

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) List(ctx context.Context, req ListReq) (ListResp, error) {
	if r == nil || r.db == nil {
		return ListResp{}, fmt.Errorf("query database is unavailable")
	}
	if req.Tab == tabRD {
		return r.listStories(ctx, req)
	}
	return r.listDemands(ctx, req)
}

// listDemands 业务需求列表：filter → sort → count → pagination 全在 SQL 完成。
// Stage 派生走 demandstage.Label。
func (r *Repo) listDemands(ctx context.Context, req ListReq) (ListResp, error) {
	base := `d.id, d.name, d.pri, d.status, d.stage,
		COALESCE(cu.realname, d.BRA) AS owner,
		COALESCE((SELECT GROUP_CONCAT(DISTINCT COALESCE(cp.name, dc.product)
			ORDER BY CASE WHEN dc.product = d.mainSystem THEN 0 ELSE 1 END, cp.name SEPARATOR '、')
			FROM zt_demandclarify dc
			LEFT JOIN zt_product cp ON cp.id = CAST(NULLIF(dc.product, '') AS UNSIGNED) AND cp.deleted = '0'
			WHERE dc.demand = d.id AND dc.product <> ''),
			COALESCE(p.name, d.mainSystem)) AS system_name,
		d.estimateLaunch, d.source`

	q := r.db.WithContext(ctx).Table("zt_demand d").
		Select(base).
		Joins("LEFT JOIN zt_user cu ON cu.account = d.BRA AND cu.deleted = '0'").
		Joins("LEFT JOIN zt_product p ON p.id = CAST(NULLIF(d.mainSystem, '') AS UNSIGNED) AND p.deleted = '0'").
		Where("d.deleted = ? AND d.parent IN (0, -1)", "0")

	// keyword 跨 id/name/owner/涉及系统模糊匹配；LOWER LIKE 大小写不敏感。
	if req.Keyword != "" {
		like := "%" + strings.ToLower(req.Keyword) + "%"
		q = q.Where(`(LOWER(d.name) LIKE ? OR LOWER(COALESCE(cu.realname, d.BRA)) LIKE ?
			OR LOWER(COALESCE(p.name, d.mainSystem)) LIKE ?
			OR EXISTS (SELECT 1 FROM zt_demandclarify kwdc
				LEFT JOIN zt_product kwcp ON kwcp.id = CAST(NULLIF(kwdc.product, '') AS UNSIGNED) AND kwcp.deleted = '0'
				WHERE kwdc.demand = d.id AND LOWER(COALESCE(kwcp.name, kwdc.product)) LIKE ?)
			OR CAST(d.id AS CHAR) = ?)`, like, like, like, like, req.Keyword)
	}
	if req.Status != "" {
		q = q.Where("d.status = ?", req.Status)
	}
	if req.Priority != "" {
		q = q.Where("d.pri = ?", req.Priority)
	}
	// owner 输入支持姓名/账号的部分匹配，并与列表展示的 COALESCE 字段保持同一口径。
	if req.Owner != "" {
		ownerLike := "%" + strings.ToLower(req.Owner) + "%"
		q = q.Where("(LOWER(COALESCE(cu.realname, d.BRA)) LIKE ? OR LOWER(d.BRA) LIKE ?)", ownerLike, ownerLike)
	}
	// system 输入支持主系统及需求澄清中的配合系统部分匹配。
	if req.System != "" {
		systemLike := "%" + strings.ToLower(req.System) + "%"
		q = q.Where(`(LOWER(COALESCE(p.name, d.mainSystem)) LIKE ? OR EXISTS (
			SELECT 1 FROM zt_demandclarify fdc
			LEFT JOIN zt_product fcp ON fcp.id = CAST(NULLIF(fdc.product, '') AS UNSIGNED) AND fcp.deleted = '0'
			WHERE fdc.demand = d.id AND LOWER(COALESCE(fcp.name, fdc.product)) LIKE ?))`, systemLike, systemLike)
	}
	if req.Stage != "" {
		q = q.Where("? IN (d.stage, d.status)", req.Stage)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return ListResp{}, err
	}

	type demandRow struct {
		ID             uint
		Name           string
		Title          string
		Pri            string
		Status         string
		Stage          string
		Owner          string
		System         string `gorm:"column:system_name"`
		EstimateLaunch *time.Time
		Source         string
	}
	var rows []demandRow
	if err := q.Order("d.id DESC").Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Find(&rows).Error; err != nil {
		return ListResp{}, err
	}
	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		title := row.Name
		if title == "" {
			title = row.Title
		}
		out = append(out, Row{
			ID:       row.ID,
			Title:    title,
			Kind:     "业务需求",
			Priority: row.Pri,
			Status:   row.Status,
			Stage:    demandstage.Label(row.Stage, row.Status),
			Owner:    dashIfEmpty(row.Owner),
			System:   dashIfEmpty(row.System),
			Deadline: formatDate(row.EstimateLaunch),
			Source:   row.Source,
			URL:      zturl.DemandViewURLWithBase("", row.ID),
		})
	}
	return ListResp{Kind: tabBiz, Rows: out, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

// listStories 研发需求列表：filter → sort → count → pagination 全在 SQL 完成。
func (r *Repo) listStories(ctx context.Context, req ListReq) (ListResp, error) {
	base := `s.id, s.title, s.pri, s.status, s.stage,
		COALESCE(su.realname, s.assignedTo) AS owner,
		COALESCE(sp.name, CAST(s.product AS CHAR)) AS system_name,
		s.estimateLaunch, s.source`

	q := r.db.WithContext(ctx).Table("zt_story s").
		Select(base).
		Joins("LEFT JOIN zt_user su ON su.account = s.assignedTo AND su.deleted = '0'").
		Joins("LEFT JOIN zt_product sp ON sp.id = s.product AND sp.deleted = '0'").
		Where("s.deleted = ? AND s.parent = ? AND s.type = ?", "0", 0, "story")

	if req.Keyword != "" {
		like := "%" + strings.ToLower(req.Keyword) + "%"
		q = q.Where("(LOWER(s.title) LIKE ? OR LOWER(COALESCE(su.realname, s.assignedTo)) LIKE ? OR LOWER(COALESCE(sp.name, CAST(s.product AS CHAR))) LIKE ? OR CAST(s.id AS CHAR) = ?)",
			like, like, like, req.Keyword)
	}
	if req.Status != "" {
		q = q.Where("s.status = ?", req.Status)
	}
	if req.Priority != "" {
		q = q.Where("s.pri = ?", req.Priority)
	}
	if req.Owner != "" {
		ownerLike := "%" + strings.ToLower(req.Owner) + "%"
		q = q.Where("(LOWER(COALESCE(su.realname, s.assignedTo)) LIKE ? OR LOWER(s.assignedTo) LIKE ?)", ownerLike, ownerLike)
	}
	if req.System != "" {
		systemLike := "%" + strings.ToLower(req.System) + "%"
		q = q.Where("LOWER(COALESCE(sp.name, CAST(s.product AS CHAR))) LIKE ?", systemLike)
	}
	if req.Stage != "" {
		q = q.Where("? IN (s.stage, s.status)", req.Stage)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return ListResp{}, err
	}

	type storyRow struct {
		ID             uint
		Title          string
		Pri            int
		Status         string
		Stage          string
		Owner          string
		System         string `gorm:"column:system_name"`
		EstimateLaunch *time.Time
		Source         string
	}
	var rows []storyRow
	if err := q.Order("s.id DESC").Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Find(&rows).Error; err != nil {
		return ListResp{}, err
	}
	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		out = append(out, Row{
			ID:       row.ID,
			Title:    row.Title,
			Kind:     "研发需求",
			Priority: fmt.Sprint(row.Pri),
			Status:   row.Status,
			Stage:    demandstage.Label(row.Stage, row.Status),
			Owner:    dashIfEmpty(row.Owner),
			System:   dashIfEmpty(row.System),
			Deadline: formatDate(row.EstimateLaunch),
			Source:   row.Source,
			URL:      zturl.StoryViewURLWithBase("", row.ID),
		})
	}
	return ListResp{Kind: tabRD, Rows: out, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

func dashIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

func formatDate(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.Format("2006-01-02")
}
