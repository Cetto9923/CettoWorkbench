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

	"gorm.io/gorm"
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
	"demand:reviewed":               {Label: "需求评审", Result: "done"},
	"demand:reviewbymanager":        {Label: "主管部门审批", Result: "approved"},
	"demand:reviewchange":           {Label: "需求变更评审", Result: "done"},
	"demand:clarify":                {Label: "完成需求澄清", Result: "done"},
	"demand:updateclarified":        {Label: "完成需求澄清", Result: "done"},
	"demand:dispatch":               {Label: "需求派单", Result: "done"},
	"demand:plan":                   {Label: "完成排期", Result: "done"},
	"demand:demandplan":             {Label: "完成排期", Result: "done"},
	"demand:updatedeveloping":       {Label: "推进至研发中", Result: "done"},
	"demand:updatetesting":          {Label: "推进至测试中", Result: "done"},
	"demand:tostory":                {Label: "转研发需求", Result: "done"},
	"demand:hangup":                 {Label: "挂起需求", Result: "done"},
	"demand:closed":                 {Label: "关闭需求", Result: "closed"},
	"demand:activated":              {Label: "激活需求", Result: "activated"},
	"demand:deliver":                {Label: "发起交付", Result: "submitted"},
	"demand:withdrawdelivery":       {Label: "撤销交付", Result: "returned"},
	"demand:acceptance":             {Label: "发起验收", Result: "submitted"},
	"demand:startacceptance":        {Label: "发起验收", Result: "submitted"},
	"demand:acceptanced":            {Label: "确认验收", Result: "verified"},
	"demand:releasedbyallstories":   {Label: "需求发布", Result: "done"},
	"demand:releasedbyticket":       {Label: "需求发布", Result: "done"},
	"story:submitreview":            {Label: "提交评审", Result: "submitted"},
	"story:reviewed":                {Label: "评审研发需求", Result: "done"},
	"story:verified":                {Label: "验收研发需求", Result: "verified"},
	"story:releasedbyrelease":       {Label: "发布研发需求", Result: "done"},
	"story:closed":                  {Label: "关闭研发需求", Result: "closed"},
	"story:activated":               {Label: "激活研发需求", Result: "activated"},
	"charter:approvalreview":        {Label: "项目章程审批", Result: "approved"},
	"planchange:approvalreview":     {Label: "计划变更审批", Result: "approved"},
	"buildguideline:approvalreview": {Label: "项目建设指引审批", Result: "approved"},
	"review:reviewed":               {Label: "项目评审", Result: "done"},
	"case:reviewed":                 {Label: "用例评审", Result: "done"},
	"task:started":                  {Label: "开始任务", Result: "done"},
	"task:finished":                 {Label: "完成任务", Result: "done"},
	"task:closed":                   {Label: "关闭任务", Result: "closed"},
	"task:canceled":                 {Label: "取消任务", Result: "done"},
	"task:activated":                {Label: "激活任务", Result: "activated"},
	"task:restarted":                {Label: "重启任务", Result: "activated"},
	"task:paused":                   {Label: "暂停任务", Result: "done"},
	"task:confirmed":                {Label: "确认任务", Result: "done"},
	"bug:resolved":                  {Label: "解决 Bug", Result: "resolved"},
	"bug:closed":                    {Label: "关闭 Bug", Result: "closed"},
	"bug:activated":                 {Label: "激活 Bug", Result: "activated"},
	"bug:bugconfirmed":              {Label: "确认 Bug", Result: "done"},
	"bug:tostory":                   {Label: "转研发需求", Result: "done"},

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
		resSQL, resArgs := buildDoneResultFilterSQL(req.Result)
		q = q.Where(resSQL, resArgs...)
	}
	if objectScopeSQL, objectScopeArgs := buildDoneObjectScopeSQL(req.Tab, req.ObjectType); objectScopeSQL != "" {
		q = q.Where(objectScopeSQL, objectScopeArgs...)
	}
	if req.Action != "" && req.Action != "all" {
		sql, args := buildActionFilterSQL(req.Action)
		q = q.Where(sql, args...)
	}
	if req.Keyword != "" {
		kw := "%" + req.Keyword + "%"
		q = q.Where(buildDoneKeywordFilterSQL(), kw, kw, kw, kw, kw, kw)
	}

	q = applyDoneTimeRange(q, req, time.Now())

	// 总数
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var rows []doneActionDBRow
	if err := q.
		Order("a.id DESC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Select("a.id, a.objectType, a.objectID, a.action, a.actor, a.date, a.extra, a.comment").
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	actionIDs := make([]int64, len(rows))
	for i, r := range rows {
		actionIDs[i] = r.ID
	}
	hists := r.fetchActionHistories(ctx, actionIDs)
	objCtxs, err := r.fetchObjectContexts(ctx, rows)
	if err != nil {
		return nil, 0, err
	}

	displayMap, _ := r.loadAccountDisplayMap(ctx)
	actor := displayMap[req.Account]
	if actor == "" {
		actor = req.Account
	}

	items := make([]DoneAction, 0, len(rows))
	for _, row := range rows {
		meta := formalDoneActions[row.ObjectType+":"+row.Action]
		actionLabel := doneHistoryActionLabel(row.ObjectType, row.Action)
		chg := hists[row.ID]
		ctx := objCtxs[fmt.Sprintf("%s:%d", row.ObjectType, row.ObjectID)]
		title := ctx.Title
		if title == "" {
			title = strings.TrimSpace(doneObjectTypeLabel(row.ObjectType) + " " + doneObjectCode(row.ObjectType, row.ObjectID))
		}
		url := objectViewURLWithProject(row.ObjectType, uint(row.ObjectID), uint(ctx.ProjectID))
		resultCode, resultText := resolveDoneActionResult(row.Action, row.ObjectType, row.Extra, meta.Result)
		items = append(items, DoneAction{
			ID:              row.ID,
			SourceActionId:  row.ID,
			SourceSystem:    "zentao",
			Actor:           actor,
			ActorName:       actor,
			Action:          actionLabel,
			ActionKey:       row.Action,
			ActionName:      actionLabel,
			IsCoreAction:    true,
			ObjectType:      row.ObjectType,
			ObjectTypeLabel: doneObjectTypeLabel(row.ObjectType),
			ObjectID:        row.ObjectID,
			ObjectCode:      doneObjectCode(row.ObjectType, row.ObjectID),
			ObjectName:      title,
			ObjectTitle:     title,
			Date:            row.Date.Format("2006-01-02 15:04:05"),
			HandledAt:       row.Date.Format(time.RFC3339),
			Result:          resultCode,
			ResultCode:      resultCode,
			ResultText:      resultText,
			BeforeStatus:    chg[0],
			AfterStatus:     chg[1],
			// 当前状态只来自对象本身；已办动作结果不能冒充对象状态。
			CurrentStatus: ctx.Status,
			ProjectName:   ctx.ProjectName,
			ExecutionName: ctx.ExecutionName,
			ProductName:   ctx.ProductName,
			PoolName:      ctx.PoolName,
			NextOwnerName: ctx.CurrentOwner,
			CanOpenObject: true,
			URL:           url,
		})
	}

	return items, total, nil
}

// applyDoneTimeRange converts calendar filters to half-open timestamp ranges so
// MySQL can use an index on zt_action.date. The custom range keeps the existing
// inclusive upper-bound contract.
func applyDoneTimeRange(q *gorm.DB, req RepoFindDoneActionsReq, now time.Time) *gorm.DB {
	startOfDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	dayStart := startOfDay(now)
	switch req.TimeRange {
	case TimeRangeToday:
		return q.Where("a.date >= ? AND a.date < ?", dayStart, dayStart.AddDate(0, 0, 1))
	case TimeRange7d:
		return q.Where("a.date >= ?", now.AddDate(0, 0, -7))
	case TimeRangeWeek:
		weekday := int(dayStart.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekStart := dayStart.AddDate(0, 0, 1-weekday)
		return q.Where("a.date >= ? AND a.date < ?", weekStart, weekStart.AddDate(0, 0, 7))
	case TimeRange30d:
		return q.Where("a.date >= ?", now.AddDate(0, 0, -30))
	case TimeRangeMonth:
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return q.Where("a.date >= ? AND a.date < ?", monthStart, monthStart.AddDate(0, 1, 0))
	case TimeRangeLastMonth:
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -1, 0)
		return q.Where("a.date >= ? AND a.date < ?", monthStart, monthStart.AddDate(0, 1, 0))
	case TimeRangeQuarter:
		quarterMonth := ((int(now.Month())-1)/3)*3 + 1
		quarterStart := time.Date(now.Year(), time.Month(quarterMonth), 1, 0, 0, 0, 0, now.Location())
		return q.Where("a.date >= ? AND a.date < ?", quarterStart, quarterStart.AddDate(0, 3, 0))
	case TimeRangeCustom:
		if req.CustomFrom != "" {
			q = q.Where("a.date >= ?", req.CustomFrom)
		}
		if req.CustomTo != "" {
			q = q.Where("a.date <= ?", req.CustomTo)
		}
	}
	return q
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
	Tab        DoneTab
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
	if objectScopeSQL, objectScopeArgs := buildDoneObjectScopeSQL(req.Tab, req.ObjectType); objectScopeSQL != "" {
		q = q.Where(objectScopeSQL, objectScopeArgs...)
	}

	type row struct {
		Total   int64 `gorm:"column:c_all"`
		Today   int64 `gorm:"column:c_today"`
		Last7d  int64 `gorm:"column:c_last7d"`
		Week    int64 `gorm:"column:c_week"`
		Last30d int64 `gorm:"column:c_last30d"`
		Month   int64 `gorm:"column:c_month"`
		Quarter int64 `gorm:"column:c_quarter"`
		Objects int64 `gorm:"column:c_objects"`
	}
	var out row
	err := q.Select(`
		COUNT(*) AS c_all,
		SUM(CASE WHEN DATE(a.date) = CURDATE() THEN 1 ELSE 0 END) AS c_today,
		SUM(CASE WHEN a.date >= DATE_SUB(NOW(), INTERVAL 7 DAY) THEN 1 ELSE 0 END) AS c_last7d,
		SUM(CASE WHEN YEARWEEK(a.date, 3) = YEARWEEK(CURDATE(), 3) THEN 1 ELSE 0 END) AS c_week,
		SUM(CASE WHEN a.date >= DATE_SUB(NOW(), INTERVAL 30 DAY) THEN 1 ELSE 0 END) AS c_last30d,
		SUM(CASE WHEN DATE_FORMAT(a.date, '%Y-%m') = DATE_FORMAT(CURDATE(), '%Y-%m') THEN 1 ELSE 0 END) AS c_month,
		SUM(CASE WHEN QUARTER(a.date) = QUARTER(CURDATE()) AND YEAR(a.date) = YEAR(CURDATE()) THEN 1 ELSE 0 END) AS c_quarter,
		COUNT(DISTINCT CONCAT(a.objectType, ':', a.objectID)) AS c_objects`).
		Scan(&out).Error
	if err != nil {
		return summary, err
	}
	return DoneSummary{
		All: out.Total, Today: out.Today, Last7d: out.Last7d, Week: out.Week,
		Last30d: out.Last30d, Month: out.Month, Quarter: out.Quarter, Objects: out.Objects,
	}, nil
}

func doneResultText(result string) string {
	switch result {
	case "done":
		return "已完成"
	case "approved":
		return "已通过"
	case "rejected":
		return "已驳回"
	case "closed":
		return "已关闭"
	case "activated":
		return "已激活"
	case "submitted":
		return "已提交"
	case "verified":
		return "已验收"
	case "resolved":
		return "已解决"
	case "returned":
		return "已退回"
	default:
		return "已记录"
	}
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
	"build": "构建", "todo": "待办", "testtask": "测试单", "charter": "项目章程",
	"planchange": "计划变更", "buildguideline": "项目建设指引", "review": "项目评审", "case": "用例",
}

// doneObjectTypeLabel 对象类型中文标签；未知类型原样返回。
func doneObjectTypeLabel(objectType string) string {
	if label, ok := doneObjectTypeLabels[objectType]; ok {
		return label
	}
	return objectType
}

func doneObjectCode(objectType string, objectID int64) string {
	if objectID <= 0 {
		return ""
	}
	if objectType == "demand" {
		return fmt.Sprintf("US%d", objectID)
	}
	return fmt.Sprintf("%d", objectID)
}

// doneScopeObjectTypes 我的已办操作对象范围（顺序与 CRCBWorkbench objectTypeOrder 对齐）。
var doneScopeObjectTypes = []string{"demand", "story", "task", "bug", "risk", "issue", "feedback", "release", "build", "todo", "testtask", "charter", "planchange", "buildguideline", "review", "case"}

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
	var typeList []string
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
	b.WriteString(")")
	return b.String(), args
}

// 对象 → 禅道详情页链接映射见 repodone_url.go（objectViewURL /
// objectViewURLWithProject）。
