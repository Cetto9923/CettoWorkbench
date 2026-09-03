// =============================================================================
// 文件: internal/module/po/reponotice.go
// 模块: PO 工作台
// 类型: action
// 职责: 从 zt_notify 和对应事件真源读取、分类和持久化通知已读状态。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type noticeRepoResp struct {
	Items      []NoticeItem
	Total      int64
	Filtered   int64
	Unread     int64
	Action     int64
	Abnormal   int64
	Today      int64
	Categories map[string]int64
}

type noticeRow struct {
	ID          int64     `gorm:"column:id"`
	ObjectType  string    `gorm:"column:objectType"`
	ObjectID    int64     `gorm:"column:objectID"`
	Subject     string    `gorm:"column:subject"`
	Data        string    `gorm:"column:data"`
	ActionCode  string    `gorm:"column:actionCode"`
	Actor       string    `gorm:"column:actor"`
	CreatedBy   string    `gorm:"column:createdBy"`
	CreatedDate time.Time `gorm:"column:createdDate"`
	IsRead      int       `gorm:"column:isRead"`
}

func (r *Repo) FindNotices(ctx context.Context, account string, req NoticeListReq) (noticeRepoResp, error) {
	resp := noticeRepoResp{Items: []NoticeItem{}, Categories: noticeCategoryCounts()}
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return resp, nil
	}

	allRows, err := r.findNoticeRows(ctx, account)
	if err != nil {
		return resp, err
	}
	resp.Total, resp.Unread, resp.Action, resp.Abnormal, resp.Today = noticeQuickCounts(allRows)
	categoryRows := filterNoticeRows(allRows, NoticeListReq{QuickView: req.QuickView, ObjectType: req.ObjectType, TimeRange: req.TimeRange, ReadState: req.ReadState, NeedAction: req.NeedAction, Keyword: req.Keyword})
	resp.Categories["all"] = int64(len(categoryRows))
	for _, row := range categoryRows {
		resp.Categories[classifyNotice(row.ObjectType, row.ActionCode)]++
	}
	filteredRows := filterNoticeRows(allRows, req)
	resp.Filtered = int64(len(filteredRows))
	start := (req.Page - 1) * req.PageSize
	if start >= len(filteredRows) {
		return resp, nil
	}
	end := start + req.PageSize
	if end > len(filteredRows) {
		end = len(filteredRows)
	}
	displayMap, _ := r.loadAccountDisplayMap(ctx)
	for _, row := range filteredRows[start:end] {
		resp.Items = append(resp.Items, newNoticeItem(row, displayMap))
	}
	return resp, nil
}

func (r *Repo) findNoticeRows(ctx context.Context, account string) ([]noticeRow, error) {
	q := r.db.WithContext(ctx).Table("zt_notify AS n").
		Select(`n.id, COALESCE(a.objectType, n.objectType) AS objectType, n.objectID, n.subject, n.data,
			COALESCE(a.action, '') AS actionCode, COALESCE(a.actor, '') AS actor, n.createdBy, n.createdDate,
			CASE WHEN nr.id IS NULL THEN 0 ELSE 1 END AS isRead`).
		Joins("LEFT JOIN zt_action AS a ON a.id = n.action").
		Joins("LEFT JOIN zt_workbench_notify_reads AS nr ON nr.notify = n.id AND nr.account = ?", account).
		Where("FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0", account)
	var rows []noticeRow
	if err := q.Order("n.createdDate DESC, n.id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func filterNoticeRows(rows []noticeRow, req NoticeListReq) []noticeRow {
	filtered := make([]noticeRow, 0, len(rows))
	for _, row := range rows {
		category := classifyNotice(row.ObjectType, row.ActionCode)
		needAction := noticeNeedsAction(row.ActionCode)
		if req.Keyword != "" && !noticeMatchesKeyword(row, req.Keyword) {
			continue
		}
		if req.ObjectType == "approval" && category != "approval" {
			continue
		}
		if req.ObjectType != "" && req.ObjectType != "all" && req.ObjectType != "approval" && row.ObjectType != req.ObjectType {
			continue
		}
		if !noticeMatchesTimeRange(row.CreatedDate, req.TimeRange) {
			continue
		}
		if req.ReadState == "unread" && row.IsRead != 0 {
			continue
		}
		if req.ReadState == "read" && row.IsRead == 0 {
			continue
		}
		if req.Category != "" && req.Category != "all" && category != req.Category {
			continue
		}
		if req.QuickView == "unread" && row.IsRead != 0 {
			continue
		}
		if req.QuickView == "action" && !needAction {
			continue
		}
		if req.QuickView == "abnormal" && category != "risk" {
			continue
		}
		if req.QuickView == "today" && !sameNoticeDay(row.CreatedDate, time.Now()) {
			continue
		}
		if req.NeedAction == "required" && !needAction {
			continue
		}
		if req.NeedAction == "none" && needAction {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func noticeQuickCounts(rows []noticeRow) (int64, int64, int64, int64, int64) {
	var unread, action, abnormal, today int64
	for _, row := range rows {
		if row.IsRead == 0 {
			unread++
		}
		if noticeNeedsAction(row.ActionCode) {
			action++
		}
		if classifyNotice(row.ObjectType, row.ActionCode) == "risk" {
			abnormal++
		}
		if sameNoticeDay(row.CreatedDate, time.Now()) {
			today++
		}
	}
	return int64(len(rows)), unread, action, abnormal, today
}

func noticeMatchesKeyword(row noticeRow, keyword string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return true
	}
	return strings.Contains(strings.ToLower(row.Subject), keyword) || strings.Contains(strings.ToLower(row.Data), keyword) || strings.Contains(strconv.FormatInt(row.ObjectID, 10), keyword)
}

func noticeMatchesTimeRange(date time.Time, timeRange string) bool {
	switch timeRange {
	case "today":
		return sameNoticeDay(date, time.Now())
	case "3d":
		return !date.Before(time.Now().AddDate(0, 0, -3))
	case "7d":
		return !date.Before(time.Now().AddDate(0, 0, -7))
	case "30d":
		return !date.Before(time.Now().AddDate(0, 0, -30))
	default:
		return true
	}
}

func noticeCategoryCounts() map[string]int64 {
	return map[string]int64{"all": 0, "business": 0, "approval": 0, "reminder": 0, "collaboration": 0, "risk": 0, "system": 0}
}

func newNoticeItem(row noticeRow, displayMap map[string]string) NoticeItem {
	actor := strings.TrimSpace(row.Actor)
	if actor == "" {
		actor = row.CreatedBy
	}
	if displayName := displayMap[actor]; displayName != "" {
		actor = displayName
	}
	return NoticeItem{ID: row.ID, ObjectType: row.ObjectType, ObjectID: row.ObjectID, Subject: strings.TrimSpace(stripHTMLTags(row.Subject)), Data: strings.TrimSpace(stripHTMLTags(row.Data)), Actor: actor, Action: row.ActionCode, Category: classifyNotice(row.ObjectType, row.ActionCode), NeedAction: noticeNeedsAction(row.ActionCode), Anomaly: classifyNotice(row.ObjectType, row.ActionCode) == "risk", Read: row.IsRead != 0, Date: row.CreatedDate.Format("2006-01-02 15:04:05"), URL: objectViewURL(row.ObjectType, uint(row.ObjectID))}
}

// classifyNotice 只按事件 action 和对象类型映射，浏览器不参与猜测分类。
func classifyNotice(objectType, action string) string {
	switch action {
	case "reviewed", "reviewpassed", "reviewrejected", "submitreview", "submit", "submitted", "returned", "withdraw":
		return "approval"
	case "reminded", "overdue", "due", "delay", "delayed", "soon":
		return "reminder"
	case "assigned", "assignedTo", "transfer", "cc", "commented", "remark", "mentioned":
		return "collaboration"
	case "rejected", "bugconfirmed", "paused", "suspended", "hangup", "archive", "blocked", "gatefailed":
		return "risk"
	}
	switch objectType {
	case "doc", "release", "system", "sync", "account":
		return "system"
	}
	return "business"
}

func noticeNeedsAction(action string) bool {
	switch action {
	case "reviewed", "clarify", "assigned", "assignedTo", "submitted", "submit", "returned", "reminded":
		return true
	}
	return false
}
func sameNoticeDay(left, right time.Time) bool {
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}

func (r *Repo) SaveNoticeRead(ctx context.Context, account string, notifyID int64) error {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || notifyID <= 0 {
		return nil
	}
	return r.db.WithContext(ctx).Exec(`INSERT INTO zt_workbench_notify_reads (notify, account, readAt)
		SELECT n.id, ?, NOW() FROM zt_notify n LEFT JOIN zt_workbench_notify_reads nr ON nr.notify = n.id AND nr.account = ?
		WHERE n.id = ? AND FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0 AND nr.id IS NULL`, account, account, notifyID, account).Error
}

func (r *Repo) SaveAllNoticeReads(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Exec(`INSERT INTO zt_workbench_notify_reads (notify, account, readAt)
		SELECT n.id, ?, NOW() FROM zt_notify n LEFT JOIN zt_workbench_notify_reads nr ON nr.notify = n.id AND nr.account = ?
		WHERE FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0 AND nr.id IS NULL`, account, account, account)
	return result.RowsAffected, result.Error
}

func stripHTMLTags(value string) string {
	var builder strings.Builder
	inTag := false
	for _, char := range value {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
