// =============================================================================
// 文件: internal/module/po/reponotice.go
// 模块: PO 工作台
// 类型: action
// 职责: 通知中心数据访问。通知内容真源 zt_notify（toList 含账号），已读 zt_workbench_notify_reads。
//       workbench 为主: 分类由 zt_action.action + objectType 推断, 不强套 V10.1 6 类。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// FindNotices 通知中心分页列表（join 触发 action 元数据 + 触发人）。
// 去重键: 每对象最新一条（按每对象的 zt_notify.id MAX）。本期先按 createdDate DESC 简化。
// 注意: zt_notify.action 是 zt_action.id (FK)，不是字符串；zt_action.action 才是字符串。
func (r *Repo) FindNotices(ctx context.Context, account string, req NoticeListReq) ([]NoticeItem, int64, int64, int64, int64, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, 0, 0, 0, 0, 0, nil
	}

	// 基础 query: 通知 toList 包含本账号
	base := r.db.WithContext(ctx).Table("zt_notify AS n").
		Where("FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0", account)

	// 关键词: subject / data / objectID
	if req.Keyword != "" {
		base = base.Where(`(n.subject LIKE ? OR n.data LIKE ? OR CAST(n.objectID AS CHAR) = ?)`,
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", req.Keyword)
	}

	// 读取标记：左连接已读表
	base = base.Select(`n.id, n.objectType, n.objectID, n.subject, n.data,
		n.action, n.createdBy, n.createdDate, n.toList,
		CASE WHEN nr.id IS NULL THEN 0 ELSE 1 END AS is_read`).
		Joins(`LEFT JOIN zt_workbench_notify_reads nr ON nr.notify = n.id AND nr.account = ?`, account)

	// 排序 + 分页
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	offset := (req.Page - 1) * req.PageSize
	if offset < 0 {
		offset = 0
	}

	type row struct {
		ID          int64     `gorm:"column:id"`
		ObjectType  string    `gorm:"column:objectType"`
		ObjectID    int64     `gorm:"column:objectID"`
		Subject     string    `gorm:"column:subject"`
		Data        string    `gorm:"column:data"`
		Action      int64     `gorm:"column:action"`
		CreatedBy   string    `gorm:"column:createdBy"`
		CreatedDate time.Time `gorm:"column:createdDate"`
		ToList      string    `gorm:"column:toList"`
		IsRead      int       `gorm:"column:is_read"`
	}

	// 各种 quick view 计数（基础集，与读取标志 join）
	// 总数: 基础集 count
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}
	// 未读: 左连接未匹配
	var unread int64
	if err := base.Session(&gorm.Session{}).Where("nr.id IS NULL").Count(&unread).Error; err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}
	// 今日: DATE(createdDate) = CURDATE()
	var today int64
	if err := base.Session(&gorm.Session{}).Where("DATE(n.createdDate) = CURDATE()").Count(&today).Error; err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}

	// 需处理 + 异常: JOIN zt_action，按 zt_action.action 字符串过滤
	// 需处理 = 需要用户介入（评审/澄清/指派/催办/退回）
	var actionCount int64
	if err := r.db.WithContext(ctx).Table("zt_notify AS n").
		Where("FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0", account).
		Joins("INNER JOIN zt_action a ON a.id = n.action").
		Where("a.action IN ?", []string{"reviewed", "clarify", "assigned", "assignedTo", "submitted", "submit", "returned", "reminded"}).
		Count(&actionCount).Error; err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}
	// 异常: 拒绝/关闭/bug确认
	var abnormal int64
	if err := r.db.WithContext(ctx).Table("zt_notify AS n").
		Where("FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0", account).
		Joins("INNER JOIN zt_action a ON a.id = n.action").
		Where("a.action IN ?", []string{"rejected", "bugconfirmed", "paused", "suspended", "hangup", "archive"}).
		Count(&abnormal).Error; err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}

	// 列表查询 + quick view 过滤
	filtered := base.Session(&gorm.Session{})
	switch req.QuickView {
	case "unread":
		filtered = filtered.Where("nr.id IS NULL")
	case "action":
		filtered = filtered.
			Joins("INNER JOIN zt_action a ON a.id = n.action").
			Where("a.action IN ?", []string{"reviewed", "clarify", "assigned", "assignedTo", "submitted", "submit", "returned", "reminded"})
	case "abnormal":
		filtered = filtered.
			Joins("INNER JOIN zt_action a ON a.id = n.action").
			Where("a.action IN ?", []string{"rejected", "bugconfirmed", "paused", "suspended", "hangup", "archive"})
	case "today":
		filtered = filtered.Where("DATE(n.createdDate) = CURDATE()")
	}

	var rows []row
	if err := filtered.
		Order("n.id DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}

	// 加载触发人展示名 + 批量加载 zt_action 元数据
	displayMap, _ := r.loadAccountDisplayMap(ctx)
	actionIDs := make([]int64, 0)
	seen := make(map[int64]struct{})
	for _, row := range rows {
		if row.Action > 0 {
			if _, ok := seen[row.Action]; !ok {
				actionIDs = append(actionIDs, row.Action)
				seen[row.Action] = struct{}{}
			}
		}
	}
	actionMeta := make(map[int64]struct{ ActionCode, ObjectType, Actor string })
	if len(actionIDs) > 0 {
		var metas []struct {
			ID         int64  `gorm:"column:id"`
			Action     string `gorm:"column:action"`
			ObjectType string `gorm:"column:objectType"`
			Actor      string `gorm:"column:actor"`
		}
		if err := r.db.WithContext(ctx).Table("zt_action").
			Where("id IN ?", actionIDs).
			Select("id, action, objectType, actor").
			Find(&metas).Error; err == nil {
			for _, m := range metas {
				actionMeta[m.ID] = struct{ ActionCode, ObjectType, Actor string }{m.Action, m.ObjectType, m.Actor}
			}
		}
	}

	items := make([]NoticeItem, 0, len(rows))
	for _, row := range rows {
		item := NoticeItem{
			ID:         row.ID,
			ObjectType: row.ObjectType,
			ObjectID:   row.ObjectID,
			Subject:    strings.TrimSpace(stripHTMLTags(row.Subject)),
			Data:       strings.TrimSpace(stripHTMLTags(row.Data)),
			Actor:      displayMap[row.CreatedBy],
			Read:       row.IsRead == 1,
			Date:       row.CreatedDate.Format("2006-01-02 15:04:05"),
		}
		if row.Action > 0 {
			if meta, ok := actionMeta[row.Action]; ok {
				if meta.Actor != "" {
					item.Actor = displayMap[meta.Actor]
				}
				if meta.ObjectType != "" {
					item.ObjectType = meta.ObjectType
				}
				item.Action = meta.ActionCode
			}
		}
		item.Category = inferCategory(item.ObjectType, item.Action)
		items = append(items, item)
	}

	return items, total, unread, actionCount, abnormal, today, nil
}

// SaveNoticeRead 标记当前账号可见的单条通知已读。
func (r *Repo) SaveNoticeRead(ctx context.Context, account string, notifyID int64) error {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || notifyID <= 0 {
		return nil
	}
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO zt_workbench_notify_reads (notify, account, readAt)
		SELECT n.id, ?, NOW()
		FROM zt_notify n
		LEFT JOIN zt_workbench_notify_reads nr ON nr.notify = n.id AND nr.account = ?
		WHERE n.id = ?
		  AND FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0
		  AND nr.id IS NULL`, account, account, notifyID, account).Error
}

// SaveAllNoticeReads 标记当前账号所有未读通知为已读。
func (r *Repo) SaveAllNoticeReads(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Exec(`
		INSERT INTO zt_workbench_notify_reads (notify, account, readAt)
		SELECT n.id, ?, NOW()
		FROM zt_notify n
		LEFT JOIN zt_workbench_notify_reads nr ON nr.notify = n.id AND nr.account = ?
		WHERE FIND_IN_SET(?, REPLACE(n.toList, ' ', '')) > 0
		  AND nr.id IS NULL`,
		account, account, account,
	)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// inferCategory workbench 通知分类推断（按 objectType + action 字符串启发式）。
// workbench 为主: 不强套 V10.1 6 类, 分类用最直接的语义映射。
func inferCategory(objectType, action string) string {
	switch action {
	case "reviewed", "clarify", "submitted", "submit", "returned":
		return "approval"
	case "rejected", "bugconfirmed", "paused", "suspended", "hangup", "archive":
		return "risk"
	case "reminded":
		return "reminder"
	case "assigned", "assignedTo":
		return "collaboration"
	}
	switch objectType {
	case "demand", "story", "task", "bug", "testtask":
		return "business"
	case "doc", "release":
		return "system"
	}
	return "business"
}

// stripHTMLTags 简单剥 HTML 标签。
func stripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}
