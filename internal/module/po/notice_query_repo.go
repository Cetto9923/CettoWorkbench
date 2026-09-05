// =============================================================================
// 文件: internal/module/po/notice_query_repo.go
// 模块: PO 工作台
// 类型: repo
// 职责: 通知中心 SQL 过滤、分类计数与分页查询构建。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

const noticeCategorySQLExpr = `CASE
	WHEN COALESCE(a.action, '') IN ('reviewed', 'reviewpassed', 'reviewrejected', 'submitreview', 'submit', 'submitted', 'returned', 'withdraw') THEN 'approval'
	WHEN COALESCE(a.action, '') IN ('reminded', 'overdue', 'due', 'delay', 'delayed', 'soon') THEN 'reminder'
	WHEN COALESCE(a.action, '') IN ('assigned', 'assignedTo', 'transfer', 'cc', 'commented', 'remark', 'mentioned') THEN 'collaboration'
	WHEN COALESCE(a.action, '') IN ('rejected', 'bugconfirmed', 'paused', 'suspended', 'hangup', 'archive', 'blocked', 'gatefailed') THEN 'risk'
	WHEN COALESCE(a.objectType, n.objectType) IN ('doc', 'release', 'system', 'sync', 'account') THEN 'system'
	ELSE 'business'
END`

const noticeNeedsActionSQLExpr = `COALESCE(a.action, '') IN ('reviewed', 'clarify', 'assigned', 'assignedTo', 'submitted', 'submit', 'returned', 'reminded')`

type noticeQuickCountsRow struct {
	Total    int64 `gorm:"column:total"`
	Unread   int64 `gorm:"column:unread"`
	Action   int64 `gorm:"column:action_count"`
	Abnormal int64 `gorm:"column:abnormal"`
	Today    int64 `gorm:"column:today_count"`
}

type noticeCategoryCountsRow struct {
	Total         int64 `gorm:"column:total"`
	ApprovalCount int64 `gorm:"column:approval_count"`
	BusinessCount int64 `gorm:"column:business_count"`
	CollabCount   int64 `gorm:"column:collab_count"`
	ReminderCount int64 `gorm:"column:reminder_count"`
	RiskCount     int64 `gorm:"column:risk_count"`
	SystemCount   int64 `gorm:"column:system_count"`
}

type noticeCombinedCountsRow struct {
	Total         int64 `gorm:"column:total"`
	Unread        int64 `gorm:"column:unread"`
	Action        int64 `gorm:"column:action_count"`
	Abnormal      int64 `gorm:"column:abnormal"`
	Today         int64 `gorm:"column:today_count"`
	ApprovalCount int64 `gorm:"column:approval_count"`
	BusinessCount int64 `gorm:"column:business_count"`
	CollabCount   int64 `gorm:"column:collab_count"`
	ReminderCount int64 `gorm:"column:reminder_count"`
	SystemCount   int64 `gorm:"column:system_count"`
}

func escapeSQLLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func noticeBaseQuery(ctx context.Context, db *gorm.DB, account string) *gorm.DB {
	return db.WithContext(ctx).Table("zt_notify AS n").
		Joins("LEFT JOIN zt_action AS a ON a.id = n.action").
		Joins("LEFT JOIN zt_workbench_notify_reads AS nr ON nr.notify = n.id AND nr.account = ?", account).
		Where("FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0", account)
}

func isNoticeReqCategoryUnfiltered(req NoticeListReq) bool {
	return req.QuickView == "" &&
		(req.ObjectType == "" || req.ObjectType == "all") &&
		req.TimeRange == "" &&
		req.ReadState == "" &&
		req.NeedAction == "" &&
		strings.TrimSpace(req.Keyword) == ""
}

func applyNoticeFilters(query *gorm.DB, now time.Time, req NoticeListReq, includeCategory bool) *gorm.DB {
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		escaped := "%" + escapeSQLLike(strings.ToLower(kw)) + "%"
		query = query.Where("(LOWER(n.subject) LIKE ? OR LOWER(n.data) LIKE ? OR CAST(n.objectID AS CHAR) LIKE ?)",
			escaped, escaped, escaped)
	}

	if req.ObjectType == "approval" {
		query = query.Where(noticeCategorySQLExpr + " = 'approval'")
	} else if req.ObjectType != "" && req.ObjectType != "all" {
		query = query.Where("COALESCE(a.objectType, n.objectType) = ?", req.ObjectType)
	}

	switch req.TimeRange {
	case "today":
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfToday := startOfToday.AddDate(0, 0, 1)
		query = query.Where("n.createdDate >= ? AND n.createdDate < ?", startOfToday, endOfToday)
	case "3d":
		threshold := now.AddDate(0, 0, -3)
		query = query.Where("n.createdDate >= ?", threshold)
	case "7d":
		threshold := now.AddDate(0, 0, -7)
		query = query.Where("n.createdDate >= ?", threshold)
	case "30d":
		threshold := now.AddDate(0, 0, -30)
		query = query.Where("n.createdDate >= ?", threshold)
	}

	if req.ReadState == "unread" {
		query = query.Where("nr.id IS NULL")
	} else if req.ReadState == "read" {
		query = query.Where("nr.id IS NOT NULL")
	}

	switch req.QuickView {
	case "unread":
		query = query.Where("nr.id IS NULL")
	case "action":
		query = query.Where(noticeNeedsActionSQLExpr)
	case "abnormal":
		query = query.Where(noticeCategorySQLExpr + " = 'risk'")
	case "today":
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfToday := startOfToday.AddDate(0, 0, 1)
		query = query.Where("n.createdDate >= ? AND n.createdDate < ?", startOfToday, endOfToday)
	}

	if req.NeedAction == "required" {
		query = query.Where(noticeNeedsActionSQLExpr)
	} else if req.NeedAction == "none" {
		query = query.Where("NOT (" + noticeNeedsActionSQLExpr + ")")
	}

	if includeCategory && req.Category != "" && req.Category != "all" {
		query = query.Where(noticeCategorySQLExpr+" = ?", req.Category)
	}

	return query
}

func (r *Repo) queryCombinedNoticeCounts(ctx context.Context, account string, now time.Time) (noticeCombinedCountsRow, error) {
	var row noticeCombinedCountsRow
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.AddDate(0, 0, 1)

	query := noticeBaseQuery(ctx, r.db, account).Select(`
		COUNT(n.id) AS total,
		COALESCE(SUM(CASE WHEN nr.id IS NULL THEN 1 ELSE 0 END), 0) AS unread,
		COALESCE(SUM(CASE WHEN `+noticeNeedsActionSQLExpr+` THEN 1 ELSE 0 END), 0) AS action_count,
		COALESCE(SUM(CASE WHEN `+noticeCategorySQLExpr+` = 'risk' THEN 1 ELSE 0 END), 0) AS abnormal,
		COALESCE(SUM(CASE WHEN n.createdDate >= ? AND n.createdDate < ? THEN 1 ELSE 0 END), 0) AS today_count,
		COALESCE(SUM(CASE WHEN `+noticeCategorySQLExpr+` = 'approval' THEN 1 ELSE 0 END), 0) AS approval_count,
		COALESCE(SUM(CASE WHEN `+noticeCategorySQLExpr+` = 'business' THEN 1 ELSE 0 END), 0) AS business_count,
		COALESCE(SUM(CASE WHEN `+noticeCategorySQLExpr+` = 'collaboration' THEN 1 ELSE 0 END), 0) AS collab_count,
		COALESCE(SUM(CASE WHEN `+noticeCategorySQLExpr+` = 'reminder' THEN 1 ELSE 0 END), 0) AS reminder_count,
		COALESCE(SUM(CASE WHEN `+noticeCategorySQLExpr+` = 'system' THEN 1 ELSE 0 END), 0) AS system_count
	`, startOfToday, endOfToday)

	if err := query.Scan(&row).Error; err != nil {
		return row, err
	}
	return row, nil
}

func (r *Repo) queryQuickNoticeCounts(ctx context.Context, account string, now time.Time) (noticeQuickCountsRow, error) {
	var row noticeQuickCountsRow
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.AddDate(0, 0, 1)

	query := noticeBaseQuery(ctx, r.db, account).Select(`
		COUNT(n.id) AS total,
		COALESCE(SUM(CASE WHEN nr.id IS NULL THEN 1 ELSE 0 END), 0) AS unread,
		COALESCE(SUM(CASE WHEN `+noticeNeedsActionSQLExpr+` THEN 1 ELSE 0 END), 0) AS action_count,
		COALESCE(SUM(CASE WHEN `+noticeCategorySQLExpr+` = 'risk' THEN 1 ELSE 0 END), 0) AS abnormal,
		COALESCE(SUM(CASE WHEN n.createdDate >= ? AND n.createdDate < ? THEN 1 ELSE 0 END), 0) AS today_count
	`, startOfToday, endOfToday)

	err := query.Scan(&row).Error
	return row, err
}

func (r *Repo) queryCategoryNoticeCounts(ctx context.Context, account string, now time.Time, req NoticeListReq) (noticeCategoryCountsRow, error) {
	var row noticeCategoryCountsRow
	q := noticeBaseQuery(ctx, r.db, account)
	q = applyNoticeFilters(q, now, req, false) // Exclude Category filter

	query := q.Select(`
		COUNT(n.id) AS total,
		COALESCE(SUM(CASE WHEN ` + noticeCategorySQLExpr + ` = 'approval' THEN 1 ELSE 0 END), 0) AS approval_count,
		COALESCE(SUM(CASE WHEN ` + noticeCategorySQLExpr + ` = 'business' THEN 1 ELSE 0 END), 0) AS business_count,
		COALESCE(SUM(CASE WHEN ` + noticeCategorySQLExpr + ` = 'collaboration' THEN 1 ELSE 0 END), 0) AS collab_count,
		COALESCE(SUM(CASE WHEN ` + noticeCategorySQLExpr + ` = 'reminder' THEN 1 ELSE 0 END), 0) AS reminder_count,
		COALESCE(SUM(CASE WHEN ` + noticeCategorySQLExpr + ` = 'risk' THEN 1 ELSE 0 END), 0) AS risk_count,
		COALESCE(SUM(CASE WHEN ` + noticeCategorySQLExpr + ` = 'system' THEN 1 ELSE 0 END), 0) AS system_count
	`)

	err := query.Scan(&row).Error
	return row, err
}

func (r *Repo) queryPagedNoticeRows(ctx context.Context, account string, now time.Time, req NoticeListReq, limit, offset int) ([]noticeRow, error) {
	q := noticeBaseQuery(ctx, r.db, account)
	q = applyNoticeFilters(q, now, req, true) // Include Category filter

	var rows []noticeRow
	err := q.Select(`n.id, COALESCE(a.objectType, n.objectType) AS objectType, n.objectID, n.subject, n.data,
		COALESCE(a.action, '') AS actionCode, COALESCE(a.actor, '') AS actor, n.createdBy, n.createdDate,
		CASE WHEN nr.id IS NULL THEN 0 ELSE 1 END AS isRead`).
		Order("n.createdDate DESC, n.id DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	return rows, err
}
