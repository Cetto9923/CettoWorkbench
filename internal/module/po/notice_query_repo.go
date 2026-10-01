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
	WHEN COALESCE(a.action, '') IN ('reviewed', 'reviewpassed', 'reviewrejected', 'submitreview', 'submit', 'submitted', 'returned', 'withdraw', 'approvalreview') THEN 'approval'
	WHEN COALESCE(a.action, '') IN ('reminded', 'overdue', 'due', 'delay', 'delayed', 'soon') THEN 'reminder'
	WHEN COALESCE(a.action, '') IN ('assigned', 'assignedTo', 'transfer', 'cc', 'commented', 'remark', 'mentioned') THEN 'collaboration'
	WHEN COALESCE(a.action, '') IN ('rejected', 'bugconfirmed', 'paused', 'suspended', 'hangup', 'archive', 'archived', 'blocked', 'gatefailed') THEN 'risk'
	WHEN COALESCE(a.objectType, n.objectType) IN ('approval', 'charter', 'guideline', 'buildguideline', 'review', 'planchange') THEN 'approval'
	WHEN COALESCE(a.objectType, n.objectType) IN ('doc', 'release', 'system', 'sync', 'account') THEN 'system'
	WHEN n.subject LIKE 'CHARTER %' OR n.subject LIKE 'GUIDELINE %' OR n.subject LIKE 'REVIEW %' OR n.subject LIKE 'APPROVAL %' OR n.subject LIKE '%审批%' OR n.subject LIKE '%评审%' OR n.data LIKE '%当前需要您进行审批%' OR n.data LIKE '%/charter-view-%' OR n.data LIKE '%/guideline-view-%' OR n.data LIKE '%/review-view-%' THEN 'approval'
	WHEN n.subject LIKE '提醒：您有 %' OR n.subject LIKE '%催办%' OR n.subject LIKE '%到期%' OR n.subject LIKE '%逾期%' OR n.subject LIKE '%延期%' OR n.data LIKE '%催办%' OR n.data LIKE '%到期%' OR n.data LIKE '%逾期%' OR n.data LIKE '%延期%' THEN 'reminder'
	WHEN n.subject LIKE '%指派%' OR n.subject LIKE '%转交%' OR n.subject LIKE '%抄送%' OR n.data LIKE '%指派给%' OR n.data LIKE '%转交%' OR n.data LIKE '%抄送%' OR n.data LIKE '%评论%' OR n.data LIKE '%备注%' THEN 'collaboration'
	WHEN n.subject LIKE '%挂起%' OR n.subject LIKE '%阻塞%' OR n.subject LIKE '%异常%' OR n.subject LIKE '%驳回%' OR n.subject LIKE '%拒绝%' OR n.subject LIKE '%终止%' OR n.data LIKE '%挂起%' OR n.data LIKE '%阻塞%' OR n.data LIKE '%异常%' OR n.data LIKE '%驳回%' OR n.data LIKE '%拒绝%' OR n.data LIKE '%终止%' THEN 'risk'
	WHEN n.subject LIKE '%系统通知%' OR n.subject LIKE '%系统公告%' OR n.data LIKE '%系统通知%' OR n.data LIKE '%系统公告%' THEN 'system'
	ELSE 'business'
END`

const noticeNeedsActionSQLExpr = `(
	COALESCE(a.action, '') IN ('reviewed', 'clarify', 'assigned', 'assignedTo', 'submitted', 'submit', 'returned', 'reminded')
	OR (COALESCE(a.action, '') = '' AND (
		n.subject LIKE 'CHARTER %' OR n.subject LIKE 'GUIDELINE %' OR n.subject LIKE 'REVIEW %' OR n.subject LIKE 'APPROVAL %'
		OR n.subject LIKE '提醒：您有 %' OR n.subject LIKE '%催办%'
		OR n.data LIKE '%当前需要您进行审批%' OR n.data LIKE '%指派给%'
	))
)`

type noticeQuickCountsRow struct {
	Total    int64 `gorm:"column:total"`
	Unread   int64 `gorm:"column:unread"`
	Action   int64 `gorm:"column:action_count"`
	Inform   int64 `gorm:"column:inform_count"`
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
	Inform        int64 `gorm:"column:inform_count"`
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

// noticeObjectTypes 通知对象类型的主体匹配条件、标题前缀正则与无编号兜底。
// 各类型的差异只有这三段，抽出为表驱动，避免 applyNoticeFilters 里堆叠近似分支。
type noticeObjectType struct {
	objectTypeSQL string // 直接匹配 objectType 的条件
	objectTypeArg any    // 非空时以绑定参数形式追加到 objectTypeSQL 之后
	subjectPrefix string // 标题前缀正则（大小写与中文别名）
	fallbackSQL   string // 标题无编号时的兜底条件
}

var noticeObjectTypes = map[string]noticeObjectType{
	"demand": {
		objectTypeSQL: "COALESCE(a.objectType, n.objectType) IN ('demand', 'sub_demand', 'business')",
		subjectPrefix: "DEMAND|demand|业务需求|需求",
		fallbackSQL:   "(n.subject LIKE '%需求%' OR n.data LIKE '%/demand-view-%')",
	},
	"story": {
		objectTypeSQL: "COALESCE(a.objectType, n.objectType) = 'story'",
		subjectPrefix: "STORY|story|研发需求|研需",
		fallbackSQL:   "n.data LIKE '%/story-view-%'",
	},
	"task": {
		objectTypeSQL: "COALESCE(a.objectType, n.objectType) = ?",
		objectTypeArg: "task",
		subjectPrefix: "TASK|task|任务",
		fallbackSQL:   "(n.subject LIKE '提醒：您有 任务%' OR n.data LIKE '%/task-view-%')",
	},
	"bug": {
		objectTypeSQL: "COALESCE(a.objectType, n.objectType) = 'bug'",
		subjectPrefix: "BUG|bug|缺陷",
		fallbackSQL:   "(n.subject LIKE '提醒：您有 Bug%' OR n.data LIKE '%/bug-view-%')",
	},
	"project": {
		objectTypeSQL: "COALESCE(a.objectType, n.objectType) = 'project'",
		subjectPrefix: "PROJECT|project|项目",
		fallbackSQL:   "n.data LIKE '%/project-view-%'",
	},
	"issue": {
		objectTypeSQL: "COALESCE(a.objectType, n.objectType) = 'issue'",
		subjectPrefix: "ISSUE|issue|问题",
		fallbackSQL:   "n.data LIKE '%/issue-view-%'",
	},
	"risk": {
		objectTypeSQL: "COALESCE(a.objectType, n.objectType) = 'risk'",
		subjectPrefix: "RISK|risk|风险",
		fallbackSQL:   "n.data LIKE '%/risk-view-%'",
	},
	"testtask": {
		objectTypeSQL: "COALESCE(a.objectType, n.objectType) IN ('testtask', 'testcase', 'case')",
		subjectPrefix: "TESTTASK|testtask|TESTCASE|testcase|测试单|测试",
		fallbackSQL:   "(n.subject LIKE '%测试单%' OR n.data LIKE '%/testtask-view-%' OR n.data LIKE '%/testcase-view-%')",
	},
}

// applyNoticeObjectTypeFilter 按对象类型过滤：命中结构化 objectType，
// 或命中邮件/消息类标题前缀，或落到该类型的兜底条件。
func applyNoticeObjectTypeFilter(query *gorm.DB, objectType string) *gorm.DB {
	if objectType == "" || objectType == "all" {
		return query
	}
	spec, ok := noticeObjectTypes[objectType]
	if !ok {
		return query.Where("COALESCE(a.objectType, n.objectType) = ?", objectType)
	}
	cond := `(
			` + spec.objectTypeSQL + `
			OR ((COALESCE(a.objectType, n.objectType, '') IN ('mail', '', 'message')) AND (
				n.subject REGEXP '^(` + spec.subjectPrefix + `)[[:space:]]*#[[:space:]]*[0-9]+'
				OR (n.subject NOT REGEXP '#[[:space:]]*[0-9]+' AND ` + spec.fallbackSQL + `)
			))
		)`
	if spec.objectTypeArg != nil {
		return query.Where(cond, spec.objectTypeArg)
	}
	return query.Where(cond)
}

// applyNoticeCreatedDateFilter 追加创建时间范围条件。
func applyNoticeCreatedDateFilter(query *gorm.DB, now time.Time, value string) *gorm.DB {
	switch value {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return query.Where("n.createdDate >= ? AND n.createdDate < ?", start, start.AddDate(0, 0, 1))
	case "3d":
		return query.Where("n.createdDate >= ?", now.AddDate(0, 0, -3))
	case "7d":
		return query.Where("n.createdDate >= ?", now.AddDate(0, 0, -7))
	case "30d":
		return query.Where("n.createdDate >= ?", now.AddDate(0, 0, -30))
	}
	return query
}

// applyNoticeQuickViewFilter 追加快捷视图条件。
func applyNoticeQuickViewFilter(query *gorm.DB, now time.Time, quickView string) *gorm.DB {
	switch quickView {
	case "unread":
		return query.Where("nr.id IS NULL")
	case "action":
		return query.Where(noticeNeedsActionSQLExpr)
	case "inform":
		return query.Where("NOT (" + noticeNeedsActionSQLExpr + ")")
	case "abnormal":
		return query.Where(noticeCategorySQLExpr + " = 'risk'")
	case "today":
		return applyNoticeCreatedDateFilter(query, now, "today")
	}
	return query
}

func applyNoticeFilters(query *gorm.DB, now time.Time, req NoticeListReq, includeCategory bool) *gorm.DB {
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		escaped := "%" + escapeSQLLike(strings.ToLower(kw)) + "%"
		query = query.Where("(LOWER(n.subject) LIKE ? OR LOWER(n.data) LIKE ? OR CAST(n.objectID AS CHAR) LIKE ?)",
			escaped, escaped, escaped)
	}

	switch req.ObjectType {
	case "approval":
		query = query.Where(noticeCategorySQLExpr + " = 'approval'")
	case "feedback":
		query = query.Where("(COALESCE(a.objectType, n.objectType) = ? OR (COALESCE(a.objectType, n.objectType, '') IN ('mail', '') AND n.subject REGEXP ?))",
			"feedback", `^(反馈|FEEDBACK|Feedback)[[:space:]]*#[[:space:]]*[0-9]+`)
	case "mail":
		query = query.Where("n.objectType = 'mail'")
	default:
		query = applyNoticeObjectTypeFilter(query, req.ObjectType)
	}

	query = applyNoticeCreatedDateFilter(query, now, req.TimeRange)

	if req.ReadState == "unread" {
		query = query.Where("nr.id IS NULL")
	} else if req.ReadState == "read" {
		query = query.Where("nr.id IS NOT NULL")
	}

	query = applyNoticeQuickViewFilter(query, now, req.QuickView)

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
		COALESCE(SUM(CASE WHEN NOT (`+noticeNeedsActionSQLExpr+`) THEN 1 ELSE 0 END), 0) AS inform_count,
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
		COALESCE(SUM(CASE WHEN NOT (`+noticeNeedsActionSQLExpr+`) THEN 1 ELSE 0 END), 0) AS inform_count,
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
