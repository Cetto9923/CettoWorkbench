// =============================================================================
// 文件: internal/module/po/repofollow.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的关注数据访问。V10.1 04 节：只有 2 个对象视图（业务需求默认 / 项目报告），
//       不再有"全部"对象 Tab。真源 zt_starinfo (followed='1')。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/pkg/zentao"
)

// RepoFindFollowedDemandsReq 查询关注业务需求的数据参数。
type RepoFindFollowedDemandsReq struct {
	Account  string
	Scope    FollowScope
	Keyword  string
	Page     int
	PageSize int
}

// RepoSaveDemandFollowReq 保存业务需求关注关系的数据参数。
type RepoSaveDemandFollowReq struct {
	Account  string
	DemandID int64
	Followed bool
}

// RepoFindFollowedProjectReportsReq 查询关注项目最新周报的数据参数。
type RepoFindFollowedProjectReportsReq struct {
	Account  string
	Keyword  string
	Page     int
	PageSize int
}

// FindFollowedDemands 查询当前账号关注的业务需求（V10.1 04 节默认对象视图）。
// 数据真源 zt_starinfo(objectType='demand', account=?, followed='1')。
// 二级筛选 (scope=key/closed) 走 zt_demand 自身字段过滤。
func (r *Repo) FindFollowedDemands(ctx context.Context, req RepoFindFollowedDemandsReq) ([]FollowItem, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(req.Account) == "" {
		return nil, 0, nil
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 基础 query: zt_starinfo JOIN zt_demand
	q := r.db.WithContext(ctx).Table("zt_starinfo si").
		Joins("INNER JOIN zt_demand d ON d.id = si.objectID AND d.deleted = '0'").
		Where("si.account = ?", req.Account).
		Where("si.objectType = ?", "demand").
		Where("si.followed = ?", "1")

	// 二级筛选
	switch req.Scope {
	case FollowScopeKey:
		q = q.Where("d.status NOT IN ?", []string{"closed", "released"})
	case FollowScopeClosed:
		q = q.Where("d.status IN ?", []string{"closed", "released"})
	}
	if req.Keyword != "" {
		q = q.Where(`(d.name LIKE ? OR CAST(d.id AS CHAR) = ?)`,
			"%"+req.Keyword+"%", req.Keyword)
	}

	// 总数
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type row struct {
		ID       int64      `gorm:"column:id"`
		Name     string     `gorm:"column:name"`
		Status   string     `gorm:"column:status"`
		Pri      string     `gorm:"column:pri"`
		BRA      string     `gorm:"column:BRA"`
		QD       string     `gorm:"column:QD"`
		RD       string     `gorm:"column:RD"`
		Deadline *time.Time `gorm:"column:deadline"`
	}
	var rows []row
	if err := q.
		Select("d.id, d.name, d.status, d.pri, d.BRA, d.QD, d.RD, d.deadline").
		Order("d.id DESC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	displayMap, _ := r.loadAccountDisplayMap(ctx)
	items := make([]FollowItem, 0, len(rows))
	for _, row := range rows {
		owner := ""
		if strings.TrimSpace(row.QD) != "" {
			owner = displayMap[row.QD]
		} else if strings.TrimSpace(row.RD) != "" {
			owner = displayMap[row.RD]
		} else {
			owner = displayMap[row.BRA]
		}
		priority := ""
		if row.Pri != "" {
			priority = "P" + row.Pri
		}
		isClosed := row.Status == "closed" || row.Status == "released"
		items = append(items, FollowItem{
			ID:         row.ID,
			Title:      row.Name,
			Status:     row.Status,
			Priority:   priority,
			Owner:      owner,
			LatestNote: "",
			Date:       "",
			IsKey:      !isClosed,
			IsClosed:   isClosed,
			URL:        zentao.DemandViewURL(uint(row.ID)),
		})
	}
	return items, total, nil
}

// FindFollowedProjectReports 查询当前账号关注项目及每个项目的最新周报。
func (r *Repo) FindFollowedProjectReports(ctx context.Context, req RepoFindFollowedProjectReportsReq) ([]FollowItem, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(req.Account) == "" {
		return nil, 0, nil
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	base := r.db.WithContext(ctx).Table("zt_project AS p").
		Joins("INNER JOIN zt_user AS u ON u.account = ? AND u.deleted = ?", req.Account, "0").
		Where("p.deleted = ?", "0").
		Where("FIND_IN_SET(u.id, TRIM(BOTH ',' FROM p.follow)) > 0")
	if req.Keyword != "" {
		base = base.Where("p.name LIKE ? OR CAST(p.id AS CHAR) LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type row struct {
		ID               int64      `gorm:"column:id"`
		Name             string     `gorm:"column:name"`
		Status           string     `gorm:"column:status"`
		PM               string     `gorm:"column:PM"`
		ImportantProject string     `gorm:"column:importantProject"`
		Cycle            string     `gorm:"column:cycle"`
		Sunday           *time.Time `gorm:"column:sunday"`
	}
	var rows []row
	if err := base.
		Joins(`LEFT JOIN zt_projectweekly AS pw ON pw.id = (
			SELECT MAX(pw2.id) FROM zt_projectweekly AS pw2 WHERE pw2.project = p.id
		)`).
		Select("p.id, p.name, p.status, p.PM, p.importantProject, pw.cycle, pw.sunday").
		Order("p.id DESC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	displayMap, _ := r.loadAccountDisplayMap(ctx)
	items := make([]FollowItem, 0, len(rows))
	for _, row := range rows {
		date := ""
		if row.Sunday != nil && row.Sunday.Year() > 1 {
			date = row.Sunday.Format("2006-01-02")
		}
		items = append(items, FollowItem{
			ID: row.ID, Title: row.Name, Status: row.Status, Owner: displayMap[row.PM],
			LatestNote: strings.TrimSpace(row.Cycle), Date: date,
			IsKey: strings.TrimSpace(row.ImportantProject) != "", IsClosed: row.Status == "closed",
			URL: zentao.URL("project", "view", fmt.Sprintf("projectID=%d", row.ID)),
		})
	}
	return items, total, nil
}

// SaveDemandFollow 切换对业务需求的关注状态。
// V10.1 04 节：取消关注只解除关注关系，不关闭业务对象。
// 写入策略：upsert zt_starinfo(objectType='demand', objectID=?, account=?, followed='1'/'0')。
func (r *Repo) SaveDemandFollow(ctx context.Context, req RepoSaveDemandFollowReq) error {
	if r == nil || r.writeDB == nil || strings.TrimSpace(req.Account) == "" || req.DemandID <= 0 {
		return nil
	}
	followedStr := "0"
	if req.Followed {
		followedStr = "1"
	}
	// upsert 全程走主库写连接，避免只读副本滞后导致误判。
	var count int64
	if err := r.writeDB.WithContext(ctx).Table("zt_starinfo").
		Where("objectType = ? AND objectID = ? AND account = ?", "demand", req.DemandID, req.Account).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return r.writeDB.WithContext(ctx).Table("zt_starinfo").
			Where("objectType = ? AND objectID = ? AND account = ?", "demand", req.DemandID, req.Account).
			Update("followed", followedStr).Error
	}
	if !req.Followed {
		// 之前未关注、现在要取消关注 — 无需操作
		return nil
	}
	return r.writeDB.WithContext(ctx).Table("zt_starinfo").Create(map[string]any{
		"objectType": "demand", "objectID": req.DemandID, "account": req.Account, "followed": "1",
	}).Error
}
