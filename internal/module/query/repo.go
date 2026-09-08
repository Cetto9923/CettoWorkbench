// 需求查询只读仓储：查询禅道需求与研发需求，不写本地状态。
package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
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
// Stage 派生走本地 queryStageLabel，与 po.service_detail.mapValueStage 同步维护。
func (r *Repo) listDemands(ctx context.Context, req ListReq) (ListResp, error) {
	base := `d.id, d.name, d.pri, d.status, d.stage,
		COALESCE(cu.realname, d.BRA) AS owner,
		COALESCE(p.name, d.mainSystem) AS system_name,
		d.estimateLaunch, d.source`

	q := r.db.WithContext(ctx).Table("zt_demand d").
		Select(base).
		Joins("LEFT JOIN zt_user cu ON cu.account = d.BRA AND cu.deleted = '0'").
		Joins("LEFT JOIN zt_product p ON p.id = CAST(NULLIF(d.mainSystem, '') AS UNSIGNED) AND p.deleted = '0'").
		Where("d.deleted = ? AND d.parent IN (0, -1)", "0")

	// keyword 跨 id/name/owner/system 模糊匹配；LOWER LIKE 大小写不敏感。
	if req.Keyword != "" {
		like := "%" + strings.ToLower(req.Keyword) + "%"
		q = q.Where("(LOWER(d.name) LIKE ? OR LOWER(COALESCE(cu.realname, d.BRA)) LIKE ? OR LOWER(COALESCE(p.name, d.mainSystem)) LIKE ? OR CAST(d.id AS CHAR) = ?)",
			like, like, like, req.Keyword)
	}
	if req.Status != "" {
		q = q.Where("d.status = ?", req.Status)
	}
	if req.Priority != "" {
		q = q.Where("d.pri = ?", req.Priority)
	}
	// owner 输入可能为账号或姓名；两个字段都命中。
	if req.Owner != "" {
		q = q.Where("(cu.realname = ? OR d.BRA = ?)", req.Owner, req.Owner)
	}
	// system 输入可能为产品名或主系统原始值。
	if req.System != "" {
		q = q.Where("(p.name = ? OR d.mainSystem = ?)", req.System, req.System)
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
			Stage:    queryStageLabel(row.Stage, row.Status),
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
		s.estimateLaunch, s.source`

	q := r.db.WithContext(ctx).Table("zt_story s").
		Select(base).
		Joins("LEFT JOIN zt_user su ON su.account = s.assignedTo AND su.deleted = '0'").
		Where("s.deleted = ? AND s.parent = ? AND s.type = ?", "0", 0, "story")

	if req.Keyword != "" {
		like := "%" + strings.ToLower(req.Keyword) + "%"
		q = q.Where("(LOWER(s.title) LIKE ? OR LOWER(COALESCE(su.realname, s.assignedTo)) LIKE ? OR CAST(s.id AS CHAR) = ?)",
			like, like, req.Keyword)
	}
	if req.Status != "" {
		q = q.Where("s.status = ?", req.Status)
	}
	if req.Priority != "" {
		q = q.Where("s.pri = ?", req.Priority)
	}
	if req.Owner != "" {
		q = q.Where("(su.realname = ? OR s.assignedTo = ?)", req.Owner, req.Owner)
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
			Stage:    queryStageLabel(row.Stage, row.Status),
			Owner:    dashIfEmpty(row.Owner),
			System:   "",
			Deadline: formatDate(row.EstimateLaunch),
			Source:   row.Source,
			URL:      zturl.StoryViewURLWithBase("", row.ID),
		})
	}
	return ListResp{Kind: tabRD, Rows: out, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

// queryStageLabel 把 stage/status 投到 9 阶段中文 label。
// 与 po.service_detail.mapValueStage 同源；单独复制以避免跨包 import 形成反向依赖。
func queryStageLabel(stage, status string) string {
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "closed":
		return "已关闭"
	case "released":
		return "生产验证"
	case "waitdeliver", "delivered":
		return "发布"
	case "acceptanced":
		return "已验收"
	case "waitacceptance":
		return "待验收"
	case "testing":
		return "测试中"
	case "developing":
		return "研发中"
	case "clarified":
		return "已排期"
	case "active", "clarify":
		return "澄清中"
	case "draft", "refuse", "wait":
		return "已受理"
	}
	sg := strings.ToLower(strings.TrimSpace(stage))
	switch sg {
	case "wait":
		return "已受理"
	case "inroadmap", "clarify":
		return "澄清中"
	case "incharter", "schedule":
		return "排期中"
	case "developing":
		return "研发中"
	case "delivering", "testing":
		return "测试中"
	case "delivered":
		return "发布"
	case "closed":
		return "已关闭"
	}
	return "未知"
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
