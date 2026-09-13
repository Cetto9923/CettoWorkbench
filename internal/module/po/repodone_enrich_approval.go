// =============================================================================
// 文件: internal/module/po/repodone_enrich_approval.go
// 模块: PO 工作台
// 类型: repo
// 职责: 我的已办对象上下文——审批对象（charter / planchange / buildguideline /
//       review / case）加载。章程和建设指引标题来自关联项目，objectID 仍为对象自身编号。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"fmt"
)

// loadApprovalObjectContexts 加载审批对象上下文。
// 查询失败必须返回 error：审批对象缺上下文时标题会退化成"项目建设指引 28"这类占位、
// URL 会丢掉 projectID 回退，把 DB 故障当成"对象不存在"会让页面静默展示错误事实。
// 注意"查询成功但对象行不存在"与"查询失败"是两种状态：前者留空由调用方回退，后者报错。
func (r *Repo) loadApprovalObjectContexts(
	ctx context.Context,
	out map[string]doneObjectContext,
	charterIDs, planchangeIDs, buildguidelineIDs, reviewIDs, caseIDs []int64,
) error {
	if r == nil || r.db == nil {
		return nil
	}

	if len(charterIDs) > 0 {
		type cRow struct {
			ID          int64  `gorm:"column:id"`
			Project     int64  `gorm:"column:project"`
			ProjectName string `gorm:"column:project_name"`
		}
		var list []cRow
		if err := r.db.WithContext(ctx).Table("zt_charter AS c").
			Select("c.id, c.project, COALESCE(p.name, '') AS project_name").
			Joins("LEFT JOIN zt_project AS p ON p.id = c.project AND p.deleted = '0'").
			Where("c.id IN ? AND c.deleted = '0'", charterIDs).Scan(&list).Error; err != nil {
			return fmt.Errorf("load charter approval contexts: %w", err)
		}
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
		if err := r.db.WithContext(ctx).Table("zt_planchange AS pc").Select("pc.id, pc.title, pc.project, COALESCE(p.name, '') AS project_name").Joins("LEFT JOIN zt_project AS p ON p.id = pc.project AND p.deleted = '0'").Where("pc.id IN ?", planchangeIDs).Scan(&list).Error; err != nil {
			return fmt.Errorf("load planchange approval contexts: %w", err)
		}
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
		if err := r.db.WithContext(ctx).Table("zt_projectbuildguide AS bg").Select("bg.id, bg.projectID, COALESCE(p.name, '') AS project_name").Joins("LEFT JOIN zt_project AS p ON p.id = bg.projectID AND p.deleted = '0'").Where("bg.id IN ? AND bg.deleted = '0'", buildguidelineIDs).Scan(&list).Error; err != nil {
			return fmt.Errorf("load buildguideline approval contexts: %w", err)
		}
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
		if err := r.db.WithContext(ctx).Table("zt_review AS rv").Select("rv.id, rv.title, rv.project, COALESCE(p.name, '') AS project_name").Joins("LEFT JOIN zt_project AS p ON p.id = rv.project AND p.deleted = '0'").Where("rv.id IN ? AND rv.deleted = '0'", reviewIDs).Scan(&list).Error; err != nil {
			return fmt.Errorf("load review approval contexts: %w", err)
		}
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
		if err := r.db.WithContext(ctx).Table("zt_case AS ca").Select("ca.id, ca.title, ca.project, COALESCE(p.name, '') AS project_name").Joins("LEFT JOIN zt_project AS p ON p.id = ca.project AND p.deleted = '0'").Where("ca.id IN ? AND ca.deleted = '0'", caseIDs).Scan(&list).Error; err != nil {
			return fmt.Errorf("load case approval contexts: %w", err)
		}
		for _, ca := range list {
			out[fmt.Sprintf("case:%d", ca.ID)] = doneObjectContext{Title: ca.Title, ProjectID: ca.Project, ProjectName: ca.ProjectName}
		}
	}
	return nil
}
