// =============================================================================
// 文件: internal/module/po/repodone.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的已办列表（V10.1 02 节：本人真实执行的正式业务动作真源 = zt_action）。
//       待办消失不能自动变已办（已办形成条件 = actor + action + objectType/objectID + date）。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/pkg/zentao"
)

// RepoFindDoneActionsReq 查询正式已办动作的数据参数。
type RepoFindDoneActionsReq struct {
	Account    string
	Tab        DoneTab
	TimeRange  TimeRange
	CustomFrom string
	CustomTo   string
	ObjectType string
	Page       int
	PageSize   int
}

type doneActionMeta struct {
	Label  string
	Result string
}

var formalDoneActions = map[string]doneActionMeta{
	"demand:reviewed":             {Label: "需求评审", Result: "done"},
	"demand:reviewpassed":         {Label: "需求审批通过", Result: "approved"},
	"demand:reviewrejected":       {Label: "需求审批驳回", Result: "rejected"},
	"demand:clarify":              {Label: "完成需求澄清", Result: "done"},
	"demand:updateclarified":      {Label: "完成需求澄清", Result: "done"},
	"demand:dispatch":             {Label: "需求派单", Result: "done"},
	"demand:plan":                 {Label: "完成排期", Result: "done"},
	"demand:demandplan":           {Label: "完成排期", Result: "done"},
	"demand:updatedeveloping":     {Label: "推进至研发中", Result: "done"},
	"demand:updatetesting":        {Label: "推进至测试中", Result: "done"},
	"demand:tostory":              {Label: "转研发需求", Result: "done"},
	"demand:hangup":               {Label: "挂起需求", Result: "done"},
	"demand:closed":               {Label: "关闭需求", Result: "closed"},
	"demand:activated":            {Label: "激活需求", Result: "activated"},
	"demand:deliver":              {Label: "发起交付", Result: "submitted"},
	"demand:withdrawdelivery":     {Label: "撤销交付", Result: "returned"},
	"demand:acceptance":           {Label: "发起验收", Result: "submitted"},
	"demand:startacceptance":      {Label: "发起验收", Result: "submitted"},
	"demand:acceptanced":          {Label: "确认验收", Result: "verified"},
	"demand:releasedbyallstories": {Label: "需求发布", Result: "done"},
	"demand:releasedbyticket":     {Label: "需求发布", Result: "done"},
	"story:submitreview":          {Label: "提交评审", Result: "submitted"},
	"story:reviewed":              {Label: "评审研发需求", Result: "done"},
	"story:reviewpassed":          {Label: "评审通过", Result: "approved"},
	"story:reviewrejected":        {Label: "评审不通过", Result: "rejected"},
	"story:verified":              {Label: "验收研发需求", Result: "verified"},
	"story:releasedbyrelease":     {Label: "发布研发需求", Result: "done"},
	"story:closed":                {Label: "关闭研发需求", Result: "closed"},
	"story:activated":             {Label: "激活研发需求", Result: "activated"},
	"task:started":                {Label: "开始任务", Result: "done"},
	"task:finished":               {Label: "完成任务", Result: "done"},
	"task:closed":                 {Label: "关闭任务", Result: "closed"},
	"task:canceled":               {Label: "取消任务", Result: "done"},
	"task:activated":              {Label: "激活任务", Result: "activated"},
	"task:restarted":              {Label: "重启任务", Result: "activated"},
	"task:paused":                 {Label: "暂停任务", Result: "done"},
	"task:confirmed":              {Label: "确认任务", Result: "done"},
	"bug:resolved":                {Label: "解决 Bug", Result: "resolved"},
	"bug:closed":                  {Label: "关闭 Bug", Result: "closed"},
	"bug:activated":               {Label: "激活 Bug", Result: "activated"},
	"bug:bugconfirmed":            {Label: "确认 Bug", Result: "done"},
	"bug:tostory":                 {Label: "转研发需求", Result: "done"},
}

func (r *Repo) FindDoneActions(ctx context.Context, req RepoFindDoneActionsReq) ([]DoneAction, int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(req.Account) == "" {
		return nil, 0, nil
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	q := r.db.WithContext(ctx).Table("zt_action AS a").
		Where("a.actor = ?", req.Account).
		Where(`
			(a.objectType = ? AND a.action IN ?) OR
			(a.objectType = ? AND a.action IN ?) OR
			(a.objectType = ? AND a.action IN ?) OR
			(a.objectType = ? AND a.action IN ?)`,
			"demand", formalDoneActionCodes("demand"),
			"story", formalDoneActionCodes("story"),
			"task", formalDoneActionCodes("task"),
			"bug", formalDoneActionCodes("bug"))
	if req.ObjectType != "" && req.ObjectType != "all" {
		q = q.Where("a.objectType = ?", req.ObjectType)
	} else {
		switch req.Tab {
		case DoneTabDemand:
			q = q.Where("a.objectType IN ?", []string{"demand", "story"})
		case DoneTabTask:
			q = q.Where("a.objectType = ?", "task")
		case DoneTabBug:
			q = q.Where("a.objectType = ?", "bug")
		case DoneTabTest:
			q = q.Where("a.objectType = ?", "testtask")
		}
	}

	// 时间段 bound
	now := time.Now()
	switch req.TimeRange {
	case TimeRangeToday:
		q = q.Where("DATE(a.date) = CURDATE()")
	case TimeRange7d:
		q = q.Where("a.date >= ?", now.AddDate(0, 0, -7))
	case TimeRangeWeek:
		q = q.Where("YEARWEEK(a.date, 3) = YEARWEEK(CURDATE(), 3)")
	case TimeRange30d:
		q = q.Where("a.date >= ?", now.AddDate(0, 0, -30))
	case TimeRangeMonth:
		q = q.Where("DATE_FORMAT(a.date, '%Y-%m') = DATE_FORMAT(CURDATE(), '%Y-%m')")
	case TimeRangeLastMonth:
		q = q.Where("DATE_FORMAT(a.date, '%Y-%m') = DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 1 MONTH), '%Y-%m')")
	case TimeRangeQuarter:
		q = q.Where("QUARTER(a.date) = QUARTER(CURDATE()) AND YEAR(a.date) = YEAR(CURDATE())")
	case TimeRangeCustom:
		if req.CustomFrom != "" {
			q = q.Where("a.date >= ?", req.CustomFrom)
		}
		if req.CustomTo != "" {
			q = q.Where("a.date <= ?", req.CustomTo)
		}
	case TimeRangeAll:
		// 不加时间过滤
	default:
		// 兜底：当作 all
	}

	// 总数
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	type row struct {
		ID         int64     `gorm:"column:id"`
		ObjectType string    `gorm:"column:objectType"`
		ObjectID   int64     `gorm:"column:objectID"`
		Action     string    `gorm:"column:action"`
		Date       time.Time `gorm:"column:date"`
	}
	var rows []row
	if err := q.
		Order("a.id DESC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Select("a.id, a.objectType, a.objectID, a.action, a.date").
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	// 批量加载对象标题
	demandIDs := make([]int64, 0)
	taskIDs := make([]int64, 0)
	bugIDs := make([]int64, 0)
	storyIDs := make([]int64, 0)
	for _, row := range rows {
		switch row.ObjectType {
		case "demand":
			demandIDs = append(demandIDs, row.ObjectID)
		case "task":
			taskIDs = append(taskIDs, row.ObjectID)
		case "bug":
			bugIDs = append(bugIDs, row.ObjectID)
		case "story":
			storyIDs = append(storyIDs, row.ObjectID)
		}
	}

	nameByKey := make(map[string]string)
	if len(demandIDs) > 0 {
		var ns []struct {
			ID   int64  `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := r.db.WithContext(ctx).Table("zt_demand").
			Where("id IN ?", demandIDs).
			Select("id, name").
			Find(&ns).Error; err == nil {
			for _, n := range ns {
				nameByKey[fmt.Sprintf("demand:%d", n.ID)] = n.Name
			}
		}
	}
	if len(taskIDs) > 0 {
		var ns []struct {
			ID   int64  `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := r.db.WithContext(ctx).Table("zt_task").
			Where("id IN ?", taskIDs).
			Select("id, name").
			Find(&ns).Error; err == nil {
			for _, n := range ns {
				nameByKey[fmt.Sprintf("task:%d", n.ID)] = n.Name
			}
		}
	}
	if len(bugIDs) > 0 {
		var ns []struct {
			ID    int64  `gorm:"column:id"`
			Title string `gorm:"column:title"`
		}
		if err := r.db.WithContext(ctx).Table("zt_bug").
			Where("id IN ?", bugIDs).
			Select("id, title").
			Find(&ns).Error; err == nil {
			for _, n := range ns {
				nameByKey[fmt.Sprintf("bug:%d", n.ID)] = n.Title
			}
		}
	}
	if len(storyIDs) > 0 {
		var ns []struct {
			ID    int64  `gorm:"column:id"`
			Title string `gorm:"column:title"`
		}
		if err := r.db.WithContext(ctx).Table("zt_story").
			Where("id IN ?", storyIDs).
			Select("id, title").
			Find(&ns).Error; err == nil {
			for _, n := range ns {
				nameByKey[fmt.Sprintf("story:%d", n.ID)] = n.Title
			}
		}
	}

	displayMap, _ := r.loadAccountDisplayMap(ctx)
	actor := displayMap[req.Account]
	if actor == "" {
		actor = req.Account
	}

	items := make([]DoneAction, 0, len(rows))
	for _, row := range rows {
		meta := formalDoneActions[row.ObjectType+":"+row.Action]
		items = append(items, DoneAction{
			ID:         row.ID,
			Actor:      actor,
			Action:     meta.Label,
			ObjectType: row.ObjectType,
			ObjectID:   row.ObjectID,
			ObjectName: nameByKey[fmt.Sprintf("%s:%d", row.ObjectType, row.ObjectID)],
			Date:       row.Date.Format("2006-01-02 15:04:05"),
			Result:     meta.Result,
			URL:        objectViewURL(row.ObjectType, uint(row.ObjectID)),
		})
	}

	return items, total, nil
}

func formalDoneActionCodes(objectType string) []string {
	codes := make([]string, 0)
	prefix := objectType + ":"
	for key := range formalDoneActions {
		if strings.HasPrefix(key, prefix) {
			codes = append(codes, strings.TrimPrefix(key, prefix))
		}
	}
	return codes
}

// objectViewURL 按 zentao 对象类型拼详情页链接。
func objectViewURL(objectType string, id uint) string {
	switch objectType {
	case "demand":
		return zentao.DemandViewURL(id)
	case "story":
		return zentao.StoryViewURL(id)
	case "task":
		return zentao.TaskViewURL(id)
	case "bug":
		return zentao.BugViewURL(id)
	case "testtask":
		return zentao.TesttaskViewURL(id)
	}
	return ""
}
