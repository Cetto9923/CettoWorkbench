// =============================================================================
// 文件: internal/module/po/repofollow.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的关注数据访问。V10.1 04 节：只有 2 个对象视图（业务需求默认 / 项目报告），
//       不再有"全部"对象 Tab。真源 zt_starinfo (followed='1')。
//       业务需求列表补充生命周期统计、进度摘要与关键时间（developFinish/testFinish/deadline）。
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
	Account   string
	Scope     FollowScope
	Lifecycle FollowLifecycle
	Keyword   string
	Page      int
	PageSize  int
}

// RepoSaveDemandFollowReq 保存业务需求关注关系的数据参数。
type RepoSaveDemandFollowReq struct {
	Account  string
	DemandID int64
	Followed bool
}

// RepoRemoveProjectReportFollowReq 解除项目周报关注关系的数据参数。
type RepoRemoveProjectReportFollowReq struct {
	Account   string
	ProjectID int64
}

// RepoFindFollowedProjectReportsReq 查询关注项目最新周报的数据参数。
type RepoFindFollowedProjectReportsReq struct {
	Account  string
	Keyword  string
	Page     int
	PageSize int
}

func followDemandWatchWhere(account string) (string, []any) {
	sql := `(
		EXISTS (
			SELECT 1 FROM zt_starinfo s
			WHERE s.objectType = 'demand' AND s.objectID = d.id AND s.account = ? AND s.followed = '1'
		)
		OR (
			FIND_IN_SET(?, REPLACE(COALESCE(d.mailto, ''), ' ', '')) > 0
			AND NOT EXISTS (
				SELECT 1 FROM zt_starinfo s2
				WHERE s2.objectType = 'demand' AND s2.objectID = d.id AND s2.account = ? AND s2.followed = '0'
			)
		)
	)`
	return sql, []any{account, account, account}
}

// FindFollowedDemands 查询当前账号关注的业务需求（V10.1 04 节默认对象视图）。
// 数据真源为 zt_starinfo(objectType='demand', account=?, followed='1')，
// 并兼容禅道历史上通过需求 mailto 字段形成的关注关系；显式取消关注优先。
func (r *Repo) FindFollowedDemands(ctx context.Context, req RepoFindFollowedDemandsReq) ([]FollowItem, int64, *FollowDemandStats, error) {
	if r == nil || r.db == nil || strings.TrimSpace(req.Account) == "" {
		return nil, 0, &FollowDemandStats{}, nil
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	watchSQL, watchArgs := followDemandWatchWhere(req.Account)
	base := r.db.WithContext(ctx).Table("zt_demand d").
		Where("d.deleted = ?", "0").
		Where(watchSQL, watchArgs...)

	if req.Keyword != "" {
		base = base.Where(`(d.name LIKE ? OR CAST(d.id AS CHAR) = ?)`,
			"%"+req.Keyword+"%", req.Keyword)
	}

	stats, err := r.countFollowDemandStats(ctx, req.Account, req.Keyword)
	if err != nil {
		return nil, 0, nil, err
	}

	filtered := base
	switch req.Scope {
	case FollowScopeOpen, "":
		filtered = filtered.Where("d.status <> ?", "closed")
	case FollowScopeKey:
		filtered = filtered.Where("d.isNeedFocus = ?", "1")
	case FollowScopeKeyOpen:
		filtered = filtered.Where("d.isNeedFocus = ? AND d.status <> ?", "1", "closed")
	case FollowScopeClosed:
		filtered = filtered.Where("d.status = ?", "closed")
	case FollowScopeOpenClean:
		filtered = filtered.Where("(d.isNeedFocus IS NULL OR d.isNeedFocus <> ?) AND d.status <> ?", "1", "closed")
	case FollowScopeAll:
		// 全量（含关闭）
	}
	switch req.Lifecycle {
	case FollowLifecycleClarifying:
		filtered = filtered.Where("d.status IN ?", []string{"draft", "wait", "refuse", "active"})
	case FollowLifecycleImplementing:
		filtered = filtered.Where("d.status IN ?", []string{"clarified", "developing", "testing", "waitacceptance", "acceptanced", "waitdeliver", "delivered"})
	case FollowLifecycleReleased:
		filtered = filtered.Where("d.status = ?", "released")
	case FollowLifecycleClosed:
		filtered = filtered.Where("d.status = ?", "closed")
	}

	var total int64
	if err := filtered.Count(&total).Error; err != nil {
		return nil, 0, nil, err
	}

	type row struct {
		ID            int64      `gorm:"column:id"`
		Name          string     `gorm:"column:name"`
		Status        string     `gorm:"column:status"`
		Pri           string     `gorm:"column:pri"`
		BRA           string     `gorm:"column:BRA"`
		QD            string     `gorm:"column:QD"`
		RD            string     `gorm:"column:RD"`
		NeedFocus     string     `gorm:"column:need_focus"`
		SystemName    string     `gorm:"column:system_name"`
		WatchSource   string     `gorm:"column:watch_source"`
		Deadline      *time.Time `gorm:"column:deadline"`
		DevelopFinish *time.Time `gorm:"column:developFinish"`
		TestFinish    *time.Time `gorm:"column:testFinish"`
	}
	var rows []row
	if err := filtered.
		Select(`d.id, d.name, d.status, d.pri, d.BRA, d.QD, d.RD, d.deadline, d.developFinish, d.testFinish,
			d.isNeedFocus AS need_focus, COALESCE(p.name, d.mainSystem, '') AS system_name,
			CASE WHEN EXISTS (
				SELECT 1 FROM zt_starinfo s3
				WHERE s3.objectType = 'demand' AND s3.objectID = d.id AND s3.account = ? AND s3.followed = '1'
			) THEN 'star' ELSE 'mailto' END AS watch_source`, req.Account).
		Joins("LEFT JOIN zt_product p ON p.id = CAST(NULLIF(d.mainSystem, '') AS UNSIGNED) AND p.deleted = '0'").
		Order("d.id DESC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, nil, err
	}

	today := time.Now().In(time.Local).Format("2006-01-02")
	displayMap, _ := r.loadAccountDisplayMap(ctx)
	items := make([]FollowItem, 0, len(rows))
	for _, row := range rows {
		owner := ""
		if strings.TrimSpace(row.QD) != "" {
			owner = displayMap[row.QD]
			if owner == "" {
				owner = row.QD
			}
		} else if strings.TrimSpace(row.RD) != "" {
			owner = displayMap[row.RD]
			if owner == "" {
				owner = row.RD
			}
		} else if strings.TrimSpace(row.BRA) != "" {
			owner = displayMap[row.BRA]
			if owner == "" {
				owner = row.BRA
			}
		}
		priority := ""
		if row.Pri != "" {
			priority = "P" + row.Pri
		}
		isClosed := row.Status == "closed"
		_, stage := mapValueStage("", row.Status)
		risk := "无"
		if row.NeedFocus == "1" {
			risk = "重点关注"
		}
		reason := "抄送关注"
		if row.WatchSource == "star" {
			reason = "主动关注"
		}
		devFinish := formatFollowDate(row.DevelopFinish)
		testFinish := formatFollowDate(row.TestFinish)
		deadline := formatFollowDate(row.Deadline)
		progressStatus, progressLabel := followProgress(row.Status, deadline, today)
		items = append(items, FollowItem{
			ID:              row.ID,
			Title:           row.Name,
			Status:          row.Status,
			Stage:           stage,
			Role:            "我关注",
			SystemName:      strings.TrimSpace(row.SystemName),
			SupportSystems:  "",
			Risk:            risk,
			Reason:          reason,
			Priority:        priority,
			Owner:           owner,
			LatestNote:      "",
			Date:            "",
			IsKey:           row.NeedFocus == "1",
			IsClosed:        isClosed,
			URL:             zentao.DemandViewURL(uint(row.ID)),
			LifecycleBucket: demandFollowLifecycleBucket(row.Status),
			DevelopFinish:   devFinish,
			TestFinish:      testFinish,
			Deadline:        deadline,
			ProgressStatus:  progressStatus,
			ProgressLabel:   progressLabel,
			ScheduleSummary: followScheduleSummary(devFinish, testFinish, deadline),
		})
	}
	return items, total, stats, nil
}

func (r *Repo) countFollowDemandStats(ctx context.Context, account, keyword string) (*FollowDemandStats, error) {
	stats := &FollowDemandStats{}
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return stats, nil
	}
	watchSQL, watchArgs := followDemandWatchWhere(account)
	args := append([]any{}, watchArgs...)
	keywordSQL := ""
	if strings.TrimSpace(keyword) != "" {
		keywordSQL = " AND (d.name LIKE ? OR CAST(d.id AS CHAR) = ?)"
		args = append(args, "%"+keyword+"%", keyword)
	}
	type agg struct {
		Open         int64 `gorm:"column:c_open"`
		All          int64 `gorm:"column:c_all"`
		Clarifying   int64 `gorm:"column:c_clarifying"`
		Implementing int64 `gorm:"column:c_implementing"`
		Released     int64 `gorm:"column:c_released"`
		Closed       int64 `gorm:"column:c_closed"`
		Key          int64 `gorm:"column:c_key"`
		KeyOpen      int64 `gorm:"column:c_key_open"`
		OpenClean    int64 `gorm:"column:c_open_clean"`
	}
	var row agg
	sql := `
SELECT
  SUM(CASE WHEN d.status <> 'closed' THEN 1 ELSE 0 END) AS c_open,
  COUNT(*) AS c_all,
  SUM(CASE WHEN d.status IN ('draft','wait','refuse','active') THEN 1 ELSE 0 END) AS c_clarifying,
  SUM(CASE WHEN d.status IN ('clarified','developing','testing','waitacceptance','acceptanced','waitdeliver','delivered') THEN 1 ELSE 0 END) AS c_implementing,
  SUM(CASE WHEN d.status = 'released' THEN 1 ELSE 0 END) AS c_released,
  SUM(CASE WHEN d.status = 'closed' THEN 1 ELSE 0 END) AS c_closed,
  SUM(CASE WHEN d.isNeedFocus = '1' THEN 1 ELSE 0 END) AS c_key,
  SUM(CASE WHEN d.isNeedFocus = '1' AND d.status <> 'closed' THEN 1 ELSE 0 END) AS c_key_open,
  SUM(CASE WHEN (d.isNeedFocus IS NULL OR d.isNeedFocus <> '1') AND d.status <> 'closed' THEN 1 ELSE 0 END) AS c_open_clean
FROM zt_demand d
WHERE d.deleted = '0' AND ` + watchSQL + keywordSQL
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&row).Error; err != nil {
		return nil, err
	}
	stats.Open = row.Open
	stats.All = row.All
	stats.Clarifying = row.Clarifying
	stats.Implementing = row.Implementing
	stats.Released = row.Released
	stats.Closed = row.Closed
	stats.Key = row.Key
	stats.KeyOpen = row.KeyOpen
	stats.OpenClean = row.OpenClean
	return stats, nil
}

func demandFollowLifecycleBucket(status string) string {
	switch strings.TrimSpace(status) {
	case "draft", "wait", "refuse", "active":
		return string(FollowLifecycleClarifying)
	case "clarified", "developing", "testing", "waitacceptance", "acceptanced", "waitdeliver", "delivered":
		return string(FollowLifecycleImplementing)
	case "released":
		return string(FollowLifecycleReleased)
	case "closed":
		return string(FollowLifecycleClosed)
	default:
		return string(FollowLifecycleClarifying)
	}
}

func formatFollowDate(t *time.Time) string {
	if t == nil || t.Year() <= 1 {
		return ""
	}
	return t.Format("2006-01-02")
}

func followProgress(status, deadline, today string) (string, string) {
	st := strings.TrimSpace(status)
	if st == "closed" || st == "released" {
		return "done", "已完成"
	}
	if deadline == "" {
		return "unknown", "—"
	}
	if deadline < today {
		return "delayed", "延期"
	}
	return "normal", "正常"
}

func followScheduleSummary(devFinish, testFinish, deadline string) string {
	parts := make([]string, 0, 3)
	if devFinish != "" {
		parts = append(parts, "开发完成 "+devFinish)
	}
	if testFinish != "" {
		parts = append(parts, "测试完成 "+testFinish)
	}
	if deadline != "" {
		parts = append(parts, "截止 "+deadline)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " · ")
}

// RemoveProjectReportFollow 按禅道项目关注字段解除当前账号对项目周报的关注。
func (r *Repo) RemoveProjectReportFollow(ctx context.Context, req RepoRemoveProjectReportFollowReq) error {
	if r == nil || r.writeDB == nil || strings.TrimSpace(req.Account) == "" || req.ProjectID <= 0 {
		return nil
	}
	return r.writeDB.WithContext(ctx).Exec(`
UPDATE zt_project AS p
INNER JOIN zt_user AS u ON u.account = ? AND u.deleted = '0'
SET p.follow = CASE
  WHEN REPLACE(COALESCE(p.follow, ''), CONCAT(',', u.id, ','), ',') = ',' THEN ''
  ELSE REPLACE(COALESCE(p.follow, ''), CONCAT(',', u.id, ','), ',')
END
WHERE p.id = ? AND p.deleted = '0' AND p.type = 'project'
  AND p.follow LIKE CONCAT('%,', u.id, ',%')`, req.Account, req.ProjectID).Error
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

// EnsureDemandUnfollowed 确保 starinfo 存在 followed=0 行（压制 mailto 历史关注）。
// 禅道 unfollowObject 在无行时 no-op，工作台列表依赖显式 followed=0 才能排除抄送。
func (r *Repo) EnsureDemandUnfollowed(ctx context.Context, req RepoSaveDemandFollowReq) error {
	req.Followed = false
	if r == nil || r.writeDB == nil || strings.TrimSpace(req.Account) == "" || req.DemandID <= 0 {
		return nil
	}
	var count int64
	if err := r.writeDB.WithContext(ctx).Table("zt_starinfo").
		Where("objectType = ? AND objectID = ? AND account = ?", "demand", req.DemandID, req.Account).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return r.writeDB.WithContext(ctx).Table("zt_starinfo").
			Where("objectType = ? AND objectID = ? AND account = ?", "demand", req.DemandID, req.Account).
			Update("followed", "0").Error
	}
	return r.writeDB.WithContext(ctx).Table("zt_starinfo").Create(map[string]any{
		"objectType": "demand", "objectID": req.DemandID, "account": req.Account, "followed": "0",
	}).Error
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
		return nil
	}
	return r.writeDB.WithContext(ctx).Table("zt_starinfo").Create(map[string]any{
		"objectType": "demand", "objectID": req.DemandID, "account": req.Account, "followed": "1",
	}).Error
}
