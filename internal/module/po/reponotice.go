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
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// noticeDataURLRe 解析 zt_notify.data 里 ZenTao 直链的视图类型与 ID，作为
// 「您有 Bug(N)」类系统级模板提醒的 objType/objID 兜底来源。仅捕获视图路径段
// 与数字 ID，避免匹配正文里的 bug 关键词。
var noticeDataURLRe = regexp.MustCompile(`/(bug|story|task|product|project|demand|testcase|testtask)-view-(\d+)\.html`)

// noticeObjectTypeFromURLKind 把 ZenTao 视图路径段归一到内部 objType。
// testcase 与 testtask 共享 ZenTao 的「测试单」概念，统一归到 testtask。
func noticeObjectTypeFromURLKind(kind string) string {
	switch kind {
	case "bug":
		return "bug"
	case "story":
		return "story"
	case "task":
		return "task"
	case "product":
		return "product"
	case "project":
		return "project"
	case "demand":
		return "demand"
	case "testcase", "testtask":
		return "testtask"
	}
	return ""
}

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

	now := time.Now()

	// 1. 统计计算（未筛选时合并为单条 SQL 聚合；筛选时分为全量 Quick 统计与过滤 Category 统计）
	if isNoticeReqCategoryUnfiltered(req) {
		comb, err := r.queryCombinedNoticeCounts(ctx, account, now)
		if err != nil {
			return resp, err
		}
		resp.Total = comb.Total
		resp.Unread = comb.Unread
		resp.Action = comb.Action
		resp.Abnormal = comb.Abnormal
		resp.Today = comb.Today

		resp.Categories["all"] = comb.Total
		resp.Categories["approval"] = comb.ApprovalCount
		resp.Categories["business"] = comb.BusinessCount
		resp.Categories["collaboration"] = comb.CollabCount
		resp.Categories["reminder"] = comb.ReminderCount
		resp.Categories["risk"] = comb.Abnormal
		resp.Categories["system"] = comb.SystemCount
	} else {
		quick, err := r.queryQuickNoticeCounts(ctx, account, now)
		if err != nil {
			return resp, err
		}
		resp.Total = quick.Total
		resp.Unread = quick.Unread
		resp.Action = quick.Action
		resp.Abnormal = quick.Abnormal
		resp.Today = quick.Today

		cats, err := r.queryCategoryNoticeCounts(ctx, account, now, req)
		if err != nil {
			return resp, err
		}
		resp.Categories["all"] = cats.Total
		resp.Categories["approval"] = cats.ApprovalCount
		resp.Categories["business"] = cats.BusinessCount
		resp.Categories["collaboration"] = cats.CollabCount
		resp.Categories["reminder"] = cats.ReminderCount
		resp.Categories["risk"] = cats.RiskCount
		resp.Categories["system"] = cats.SystemCount
	}

	// 2. 计算 Filtered 总数
	if req.Category != "" && req.Category != "all" {
		resp.Filtered = resp.Categories[req.Category]
	} else {
		resp.Filtered = resp.Categories["all"]
	}

	// 3. 有界 SQL 分页加载明细
	if resp.Filtered == 0 {
		return resp, nil
	}
	start := (req.Page - 1) * req.PageSize
	if start >= int(resp.Filtered) {
		return resp, nil
	}
	limit := req.PageSize
	pagedRows, err := r.queryPagedNoticeRows(ctx, account, now, req, limit, start)
	if err != nil {
		return resp, err
	}

	displayMap, _ := r.loadAccountDisplayMap(ctx)
	for _, row := range pagedRows {
		resp.Items = append(resp.Items, newNoticeItem(row, displayMap))
	}
	return resp, nil
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
	title := cleanNoticeText(row.Subject, 80)
	summary := cleanNoticeSummary(title, row.Data, 100)
	content := cleanNoticeText(row.Data, 0)
	if summary == title {
		summary = ""
	}

	objType := row.ObjectType
	objID := row.ObjectID
	if objType == "mail" || objType == "" || objID == 0 {
		if parsedType, parsedID, _ := parseNoticeSubject(row.Subject); parsedType != "" && parsedID > 0 {
			objType = parsedType
			objID = parsedID
		}
	}

	// 兜底：「您有 Bug(N)」类系统级模板提醒的 subject 不符合 TYPE #ID 形态，
	// parseNoticeSubject 无法归类；此时从 row.Data 中的 ZenTao 直链反推对象。
	if objType == "" || objID == 0 {
		if m := noticeDataURLRe.FindStringSubmatch(row.Data); m != nil {
			viewKind := m[1]
			parsedID, err := strconv.ParseInt(m[2], 10, 64)
			if err == nil && parsedID > 0 {
				if objType == "" {
					objType = noticeObjectTypeFromURLKind(viewKind)
				}
				if objID == 0 {
					objID = parsedID
				}
			}
		}
	}

	var url string
	if objID > 0 {
		url = objectViewURL(objType, uint(objID))
	}
	return NoticeItem{
		ID:         row.ID,
		ObjectType: objType,
		ObjectID:   objID,
		Title:      title,
		Summary:    summary,
		Content:    content,
		Subject:    title,
		Data:       summary,
		Actor:      actor,
		Action:     row.ActionCode,
		Category:   classifyNotice(objType, row.ActionCode),
		NeedAction: noticeNeedsAction(row.ActionCode),
		Anomaly:    classifyNotice(objType, row.ActionCode) == "risk",
		Read:       row.IsRead != 0,
		Date:       row.CreatedDate.Format("2006-01-02 15:04:05"),
		URL:        url,
	}
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

// CheckNoticeAccess 校验用户对通知项的归属与存在性（Service 对象级授权）。
func (r *Repo) CheckNoticeAccess(ctx context.Context, account string, notifyID int64) (exists bool, authorized bool, err error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || notifyID <= 0 {
		return false, false, nil
	}
	var row struct {
		ID     int64  `gorm:"column:id"`
		ToList string `gorm:"column:toList"`
	}
	err = r.db.WithContext(ctx).Table("zt_notify").
		Select("id, toList").
		Where("id = ?", notifyID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, nil
		}
		return false, false, err
	}
	exists = true
	for _, recipient := range strings.Split(row.ToList, ",") {
		if strings.TrimSpace(recipient) == account {
			authorized = true
			break
		}
	}
	return exists, authorized, nil
}

func (r *Repo) SaveNoticeRead(ctx context.Context, account string, notifyID int64) error {
	if r == nil || r.writeDB == nil || strings.TrimSpace(account) == "" || notifyID <= 0 {
		return nil
	}
	return r.writeDB.WithContext(ctx).Exec(`INSERT INTO zt_workbench_notify_reads (notify, account, readAt)
		SELECT n.id, ?, NOW() FROM zt_notify n LEFT JOIN zt_workbench_notify_reads nr ON nr.notify = n.id AND nr.account = ?
		WHERE n.id = ? AND FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0 AND nr.id IS NULL`, account, account, notifyID, account).Error
}

// SaveAllNoticeReads 使用列表相同谓词，但不分页；仅插入当前账号尚未读的匹配项。
func (r *Repo) SaveAllNoticeReads(ctx context.Context, account string, req NoticeListReq) (int64, error) {
	if r == nil || r.writeDB == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	now := time.Now()
	query := applyNoticeFilters(noticeBaseQuery(ctx, r.writeDB, account), now, req, true).
		Where("nr.id IS NULL").Select("n.id, ? AS account, ? AS readAt", account, now)
	var rows []struct{ ID int64 }
	stmt := query.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if stmt.Error != nil {
		return 0, stmt.Error
	}
	// SQL 来自 GORM 构建器，筛选值全部保留绑定参数；并发重复标记不覆盖既有 readAt。
	result := r.writeDB.WithContext(ctx).Exec("INSERT INTO zt_workbench_notify_reads (notify, account, readAt) "+
		stmt.SQL.String()+" ON DUPLICATE KEY UPDATE readAt = zt_workbench_notify_reads.readAt", stmt.Vars...)
	return result.RowsAffected, result.Error
}

// CountUnreadNotices 仅返回当前账号未读通知总数（侧栏角标用）。
// 数据真源：zt_notify LEFT JOIN zt_workbench_notify_reads WHERE nr.id IS NULL。
// 与 FindNotices 中 unread 计数的基准相同；不应用 QuickView / 类别等过滤器，
// 保持角标与底部未读提醒一致。
func (r *Repo) CountUnreadNotices(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var n int64
	err := noticeBaseQuery(ctx, r.db, account).
		Where("nr.id IS NULL").
		Count(&n).Error
	if err != nil {
		return 0, err
	}
	return n, nil
}
