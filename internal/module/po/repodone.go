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
	Result     string
	Action     string
	Keyword    string
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

	// 反馈/发布/待办等无 formal 明细对象的常用动作（中文展示，result 尽力归类）
	"feedback:closed":   {Label: "关闭反馈", Result: "closed"},
	"feedback:replied":  {Label: "回复反馈", Result: "done"},
	"feedback:resolved": {Label: "解决反馈", Result: "resolved"},
	"release:delivered": {Label: "发布上线", Result: "done"},
	"release:closed":    {Label: "关闭发布", Result: "closed"},
	"todo:opened":       {Label: "创建待办", Result: "done"},
	"todo:finished":     {Label: "完成待办", Result: "done"},
	"todo:closed":       {Label: "关闭待办", Result: "closed"},
	"todo:assigned":     {Label: "指派待办", Result: "done"},
	"todo:started":      {Label: "开始待办", Result: "done"},
	"todo:deleted":      {Label: "删除待办", Result: "done"},
	"todo:activated":    {Label: "激活待办", Result: "activated"},
	"risk:closed":       {Label: "关闭风险", Result: "closed"},
	"risk:resolved":     {Label: "解决风险", Result: "resolved"},
	"issue:closed":      {Label: "关闭问题", Result: "closed"},
	"issue:resolved":    {Label: "解决问题", Result: "resolved"},
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

	scopeSQL, scopeArgs := buildFormalDoneScopeSQL()
	q := r.db.WithContext(ctx).Table("zt_action AS a").
		Where("a.actor = ?", req.Account).
		Where(scopeSQL, scopeArgs...)
	if req.Result != "" && req.Result != "all" {
		q = q.Where("a.action IN ?", codesWithResult(req.Result))
	}
	if req.ObjectType != "" && req.ObjectType != "all" {
		q = q.Where("a.objectType = ?", req.ObjectType)
	} else {
		switch req.Tab {
		case DoneTabApproval:
			q = q.Where(buildApprovalDoneScopeSQL())
		case DoneTabDemand:
			q = q.Where("a.objectType IN ?", []string{"demand", "story"})
		case DoneTabExecution:
			q = q.Where("a.objectType IN ?", []string{"task", "build", "release"})
		case DoneTabQuality:
			q = q.Where("a.objectType IN ?", []string{"bug", "testtask"})
		case DoneTabRisks:
			q = q.Where("a.objectType IN ?", []string{"risk", "issue"})
		}
	}
	if req.Action != "" && req.Action != "all" {
		sql, args := buildActionFilterSQL(req.Action)
		q = q.Where(sql, args...)
	}
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		q = q.Where(buildDoneKeywordFilterSQL(), kw, kw, kw, kw, kw, kw)
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
		actionLabel := meta.Label
		if actionLabel == "" {
			actionLabel = row.Action // 无 formal 定义（如 todo/release/feedback）时展示原始动作 code
		}
		items = append(items, DoneAction{
			ID:              row.ID,
			Actor:           actor,
			Action:          actionLabel,
			ObjectType:      row.ObjectType,
			ObjectTypeLabel: doneObjectTypeLabel(row.ObjectType),
			ObjectID:        row.ObjectID,
			ObjectName:      nameByKey[fmt.Sprintf("%s:%d", row.ObjectType, row.ObjectID)],
			Date:            row.Date.Format("2006-01-02 15:04:05"),
			Result:          meta.Result,
			URL:             objectViewURL(row.ObjectType, uint(row.ObjectID)),
		})
	}

	return items, total, nil
}

// buildApprovalDoneScopeSQL 返回真正构成审批决策的已办动作范围。
// 审批决策不是独立对象表，需按需求和研发需求的正式评审动作筛选。
func buildApprovalDoneScopeSQL() string {
	return "((a.objectType = 'demand' AND a.action IN ('reviewed', 'reviewpassed', 'reviewrejected')) OR " +
		"(a.objectType = 'story' AND a.action IN ('submitreview', 'reviewed', 'reviewpassed', 'reviewrejected')))"
}

// buildActionFilterSQL 处理动作筛选：req.Action 为 "objectType:action" 全键，逗号分隔。
// 按 (objectType=? AND action=?) 复合条件过滤，避免同名动作跨对象串扰（如 "closed"）。
func buildActionFilterSQL(raw string) (string, []interface{}) {
	var b strings.Builder
	args := make([]interface{}, 0, 4)
	first := true
	for _, key := range strings.Split(raw, ",") {
		key = strings.TrimSpace(key)
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		if !first {
			b.WriteString(" OR ")
		}
		first = false
		b.WriteString("(a.objectType = ? AND a.action = ?)")
		args = append(args, parts[0], parts[1])
	}
	if first {
		return "1=1", nil
	}
	return "(" + b.String() + ")", args
}

// buildDoneKeywordFilterSQL 搜索 ID / 标题 / 操作内容：
// 命中对象 ID、动作 code，或按对象类型匹配其名称/标题（demand/story/task/bug 有确认的标题列）。
func buildDoneKeywordFilterSQL() string {
	return `(CAST(a.objectID AS CHAR) LIKE ?
		OR a.action LIKE ?
		OR (a.objectType = 'demand' AND a.objectID IN (SELECT id FROM zt_demand WHERE name LIKE ?))
		OR (a.objectType = 'story'  AND a.objectID IN (SELECT id FROM zt_story  WHERE title LIKE ?))
		OR (a.objectType = 'task'   AND a.objectID IN (SELECT id FROM zt_task   WHERE name LIKE ?))
		OR (a.objectType = 'bug'    AND a.objectID IN (SELECT id FROM zt_bug    WHERE title LIKE ?)))`
}

// RepoCountDoneActionsReq 已办时间段概览计数参数。
type RepoCountDoneActionsReq struct {
	Account    string
	ObjectType string // 空 = 全部对象
}

// CountDoneActions 一次 SQL 聚合统计当前账号正式已办动作的 7 个时间段计数。
// 时间段依据 MySQL 函数归一，无 Go 侧日期参数；objectType 空表示全部对象。
func (r *Repo) CountDoneActions(ctx context.Context, req RepoCountDoneActionsReq) (DoneSummary, error) {
	summary := DoneSummary{}
	if r == nil || r.db == nil || strings.TrimSpace(req.Account) == "" {
		return summary, nil
	}
	q := r.db.WithContext(ctx).Table("zt_action AS a").
		Where("a.actor = ?", req.Account)
	scopeSQL, scopeArgs := buildFormalDoneScopeSQL()
	q = q.Where(scopeSQL, scopeArgs...)
	if req.ObjectType != "" {
		q = q.Where("a.objectType = ?", req.ObjectType)
	}

	type row struct {
		Total   int64 `gorm:"column:c_all"`
		Today   int64 `gorm:"column:c_today"`
		Last7d  int64 `gorm:"column:c_last7d"`
		Week    int64 `gorm:"column:c_week"`
		Last30d int64 `gorm:"column:c_last30d"`
		Month   int64 `gorm:"column:c_month"`
		Quarter int64 `gorm:"column:c_quarter"`
	}
	var out row
	err := q.Select(`
		COUNT(*) AS c_all,
		SUM(CASE WHEN DATE(a.date) = CURDATE() THEN 1 ELSE 0 END) AS c_today,
		SUM(CASE WHEN a.date >= DATE_SUB(NOW(), INTERVAL 7 DAY) THEN 1 ELSE 0 END) AS c_last7d,
		SUM(CASE WHEN YEARWEEK(a.date, 3) = YEARWEEK(CURDATE(), 3) THEN 1 ELSE 0 END) AS c_week,
		SUM(CASE WHEN a.date >= DATE_SUB(NOW(), INTERVAL 30 DAY) THEN 1 ELSE 0 END) AS c_last30d,
		SUM(CASE WHEN DATE_FORMAT(a.date, '%Y-%m') = DATE_FORMAT(CURDATE(), '%Y-%m') THEN 1 ELSE 0 END) AS c_month,
		SUM(CASE WHEN QUARTER(a.date) = QUARTER(CURDATE()) AND YEAR(a.date) = YEAR(CURDATE()) THEN 1 ELSE 0 END) AS c_quarter`).
		Scan(&out).Error
	if err != nil {
		return summary, err
	}
	return DoneSummary{
		All: out.Total, Today: out.Today, Last7d: out.Last7d, Week: out.Week,
		Last30d: out.Last30d, Month: out.Month, Quarter: out.Quarter,
	}, nil
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

// codesWithResult 返回 formalDoneActions 中指定处理结果的全部 action code。
func codesWithResult(result string) []string {
	codes := make([]string, 0)
	for key, meta := range formalDoneActions {
		if meta.Result == result {
			codes = append(codes, strings.TrimPrefix(key, objectTypePrefix(key)))
		}
	}
	return codes
}

// objectTypePrefix 取 "demand:reviewed" 的 "demand:" 前缀。
func objectTypePrefix(key string) string {
	if i := strings.Index(key, ":"); i >= 0 {
		return key[:i+1]
	}
	return ""
}

// doneObjectTypeLabels 对象类型中文标签（与操作对象 chips 顺序一致）。
var doneObjectTypeLabels = map[string]string{
	"demand": "业务需求", "story": "研发需求", "task": "任务", "bug": "Bug",
	"risk": "风险", "issue": "问题", "feedback": "反馈", "release": "发布",
	"build": "构建", "todo": "待办", "testtask": "测试单",
}

// doneObjectTypeLabel 对象类型中文标签；未知类型原样返回。
func doneObjectTypeLabel(objectType string) string {
	if label, ok := doneObjectTypeLabels[objectType]; ok {
		return label
	}
	return objectType
}

// doneScopeObjectTypes 我的已办操作对象范围（顺序与 CRCBWorkbench objectTypeOrder 对齐）。
var doneScopeObjectTypes = []string{"demand", "story", "task", "bug", "risk", "issue", "feedback", "release", "build", "todo", "testtask"}

// buildFormalDoneScopeSQL 构造已办正式动作范围 SQL：有 formal 白名单的对象按 action 过滤，
// 其余对象类型（risk/issue/feedback/release/build/todo/testtask）按 objectType 放行（任何 action）。
// 返回 SQL 片段（不带 WHERE 关键字）与对应参数；SQL 全部使用 ? 占位符。
func buildFormalDoneScopeSQL() (string, []interface{}) {
	var b strings.Builder
	args := make([]interface{}, 0, len(doneScopeObjectTypes)*2)
	b.WriteString("(")
	first := true
	for _, ot := range doneScopeObjectTypes {
		codes := formalDoneActionCodes(ot)
		if len(codes) == 0 {
			continue // formalDoneActions 未定义的对象类型交给兜底分支处理
		}
		if !first {
			b.WriteString(" OR ")
		}
		first = false
		b.WriteString("(a.objectType = ? AND a.action IN ?)")
		args = append(args, ot, codes)
	}
	// 兜底：无 formal 白名单的对象类型，只要 actor=本人即视为已办动作
	if len(doneScopeObjectTypes) > 0 {
		typeList := make([]string, 0)
		for _, ot := range doneScopeObjectTypes {
			if len(formalDoneActionCodes(ot)) == 0 {
				typeList = append(typeList, ot)
			}
		}
		if len(typeList) > 0 {
			if !first {
				b.WriteString(" OR ")
			}
			b.WriteString("a.objectType IN ?")
			args = append(args, typeList)
		}
	}
	b.WriteString(")")
	return b.String(), args
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
	case "risk", "issue", "feedback", "release", "build", "todo", "case":
		return zentao.URL(objectType, "view", fmt.Sprintf("%sID=%d", objectType, id))
	}
	return ""
}
