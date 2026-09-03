// =============================================================================
// 文件: internal/module/po/repo_follow.go
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
)

// FindFollowedDemands 查询当前账号关注的业务需求（V10.1 04 节默认对象视图）。
// 数据真源 zt_starinfo(objectType='demand', account=?, followed='1')。
// 二级筛选 (scope=key/closed) 走 zt_demand 自身字段过滤。
func (r *Repo) FindFollowedDemands(ctx context.Context, account string, scope FollowScope, keyword string, page, pageSize int) ([]FollowItem, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 基础 query: zt_starinfo JOIN zt_demand
	q := r.db.WithContext(ctx).Table("zt_starinfo si").
		Joins("INNER JOIN zt_demand d ON d.id = si.objectID AND d.deleted = '0'").
		Where("si.account = ?", account).
		Where("si.objectType = ?", "demand").
		Where("si.followed = ?", "1")

	// 二级筛选
	switch scope {
	case FollowScopeKey:
		q = q.Where("d.status NOT IN ?", []string{"closed", "released"})
	case FollowScopeClosed:
		q = q.Where("d.status IN ?", []string{"closed", "released"})
	}
	if keyword != "" {
		q = q.Where(`(d.name LIKE ? OR CAST(d.id AS CHAR) = ?)`,
			"%"+keyword+"%", keyword)
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
		Limit(pageSize).
		Offset((page - 1) * pageSize).
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
			URL:        "",
		})
	}
	return items, total, nil
}

// SetDemandFollow 切换对业务需求的关注状态。
// V10.1 04 节：取消关注只解除关注关系，不关闭业务对象。
// 写入策略：upsert zt_starinfo(objectType='demand', objectID=?, account=?, followed='1'/'0')。
func (r *Repo) SetDemandFollow(ctx context.Context, account string, demandID int64, followed bool) error {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || demandID <= 0 {
		return nil
	}
	followedStr := "0"
	if followed {
		followedStr = "1"
	}
	// 检查是否已存在
	var count int64
	if err := r.db.WithContext(ctx).Table("zt_starinfo").
		Where("objectType = ? AND objectID = ? AND account = ?", "demand", demandID, account).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return r.db.WithContext(ctx).Table("zt_starinfo").
			Where("objectType = ? AND objectID = ? AND account = ?", "demand", demandID, account).
			Update("followed", followedStr).Error
	}
	if !followed {
		// 之前未关注、现在要取消关注 — 无需操作
		return nil
	}
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO zt_starinfo (objectType, objectID, account, followed) VALUES (?, ?, ?, '1')`,
		"demand", demandID, account,
	).Error
}

// _ 防 fmt 报 unused (sql 占位在 insert/update 中)
var _ = fmt.Sprintf
