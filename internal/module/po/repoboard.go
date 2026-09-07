// =============================================================================
// 文件: internal/module/po/repoboard.go
// 模块: PO 工作台
// 类型: action
// 职责: 需求看板数据访问。V1.3 原则：阶段不可拖拽（状态派生）。
//       树形：业务需求(根) → 子业务 → 研发需求(交付进展)；独立研发需求单列根。
//       研发需求为「最细有效推进对象」，携带任务完成进度与当前执行负责人。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// boardDemandRow 业务需求/子业务共用列。
type boardDemandRow struct {
	ID         int64      `gorm:"column:id"`
	Parent     int64      `gorm:"column:parent"`
	Name       string     `gorm:"column:name"`
	Status     string     `gorm:"column:status"`
	Priority   string     `gorm:"column:pri"` // zt_demand.pri 为 varchar，可能为 ""，须按字符串扫描
	AssignedTo string     `gorm:"column:assignedTo"`
	Deadline   *time.Time `gorm:"column:deadline"`
}

// boardStoryRow 研发需求（zt_story）供树挂载所用列。
type boardStoryRow struct {
	ID         int64  `gorm:"column:id"`
	DemandID   int64  `gorm:"column:fromDemand"`
	Title      string `gorm:"column:title"`
	Status     string `gorm:"column:status"`
	Stage      string `gorm:"column:stage"`
	Priority   string `gorm:"column:pri"` // zt_story.pri 可能为 ""，按字符串扫描
	AssignedTo string `gorm:"column:assignedTo"`
	Product    int64  `gorm:"column:product"`
	SourceType string `gorm:"column:sourceType"`
}

// taskAgg 一组任务的聚合：用于研需进度 / 当前执行负责人。
type taskAgg struct {
	Total     int
	Done      int
	CurrOwner string
}

// FindBoardDemandTree 查需求树：业务需求根 + 子需求 + 研发需求(交付进展行) + 独立研发需求单列。
// 小组过滤通过 zt_demand.teamGroup 真实关联（无关联表时探明为空则保留全部不假过滤）。
func (r *Repo) FindBoardDemandTree(ctx context.Context, req BoardDemandReq, displayMap map[string]string) ([]*BoardDemandItem, BoardDemandSummary, error) {
	summary := BoardDemandSummary{}
	if r == nil || r.db == nil || strings.TrimSpace(req.POAccount) == "" {
		return nil, summary, nil
	}
	account := req.POAccount

	// 业务需求根（parent=0，PO 行动 scope）
	var roots []boardDemandRow
	q := r.db.WithContext(ctx).Table("zt_demand").
		Where(`id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
			OR QD = ? OR RD = ? OR BRA = ?`, account, account, account, account).
		Where("parent = ?", 0).
		Where("deleted = ?", "0").
		Where("status NOT IN ?", []string{"closed", "cancel"})
	if req.TeamgroupID > 0 {
		q = q.Where("teamGroup = ?", req.TeamgroupID)
	}
	if req.Stage != "" && req.Stage != "all" {
		q = q.Where("status = ?", req.Stage)
	}
	if req.Keyword != "" {
		q = q.Where("name LIKE ?", "%"+req.Keyword+"%")
	}
	if err := q.Order("id DESC").Limit(200).Find(&roots).Error; err != nil {
		return nil, summary, err
	}
	if len(roots) == 0 {
		// 无业务需求时仍尝试渲染独立研发需求
		return r.buildDemandTree(ctx, account, nil, nil, displayMap)
	}

	rootIDs := make([]int64, 0, len(roots))
	for _, rr := range roots {
		rootIDs = append(rootIDs, rr.ID)
	}

	// 子需求（parent = 根需求，子需求继承父需求小组）
	var children []boardDemandRow
	childQ := r.db.WithContext(ctx).Table("zt_demand").
		Where("parent IN ?", rootIDs).
		Where("deleted = ?", "0").
		Where("status NOT IN ?", []string{"closed", "cancel"})
	if err := childQ.Find(&children).Error; err != nil {
		return nil, summary, err
	}

	return r.buildDemandTree(ctx, account, roots, children, displayMap)
}

// buildDemandTree 组装树并统计摘要。roots/children 为 nil/空时仅输出独立研发需求。
// 树规则：业务需求(根) → 子需求 → 研发需求；独立研发需求单列根。
// 研发需求挂到其直接 parent.demand 节点（可能是子需求）；父节点负责汇总，最细对象承载阶段卡。
func (r *Repo) buildDemandTree(ctx context.Context, account string, roots, children []boardDemandRow, displayMap map[string]string) ([]*BoardDemandItem, BoardDemandSummary, error) {
	summary := BoardDemandSummary{}

	allDemand := make([]int64, 0, len(roots)+len(children))
	seen := make(map[int64]bool, len(roots)+len(children))
	nodeIDs := func(rows []boardDemandRow) {
		for _, row := range rows {
			if !seen[row.ID] {
				seen[row.ID] = true
				allDemand = append(allDemand, row.ID)
			}
		}
	}
	nodeIDs(roots)
	nodeIDs(children)

	// 树内研需（story.fromDemand ∈ 需求集）
	var stories []boardStoryRow
	if len(allDemand) > 0 {
		if err := r.db.WithContext(ctx).Table("zt_story").
			Where("fromDemand IN ?", allDemand).
			Where("deleted = ?", "0").
			Where("status != ?", "closed").
			Select("id, fromDemand, title, status, stage, pri, assignedTo, product, sourceType").
			Find(&stories).Error; err != nil {
			return nil, summary, err
		}
	}

	// 独立研发需求：非需求池来源且无 parent demand，PO 直接工作对象
	var independent []boardStoryRow
	excludedSource := []string{"", "demandpool", "demandlib", "feedback"}
	if err := r.db.WithContext(ctx).Table("zt_story").
		Where("(fromDemand IS NULL OR fromDemand = 0)").
		Where("deleted = ?", "0").
		Where("status != ?", "closed").
		Where("sourceType NOT IN ?", excludedSource).
		Where("(assignedTo = ? OR openedBy = ?)", account, account).
		Select("id, fromDemand, title, status, stage, pri, assignedTo, product, sourceType").
		Order("id DESC").Limit(100).
		Find(&independent).Error; err != nil {
		return nil, summary, err
	}

	// 任务进度聚合（作用于树内研需 + 独立研需）
	taskMap, err := r.aggregateStoryTasks(ctx, stories, independent)
	if err != nil {
		return nil, summary, err
	}
	productNames, err := r.storyProductNames(ctx, stories, independent)
	if err != nil {
		return nil, summary, err
	}

	// 组装：业务需求根 +
	rootByID := map[int64]*BoardDemandItem{}
	subByID := map[int64]*BoardDemandItem{}
	out := make([]*BoardDemandItem, 0, len(roots)+len(independent))
	for _, rr := range roots {
		item := r.demandNode(rr, "demand", reqOwner(account, rr.AssignedTo), displayMap)
		item.URL = zentao.DemandViewURL(uint(rr.ID))
		rootByID[rr.ID] = item
		out = append(out, item)
	}
	for _, c := range children {
		node := r.demandNode(c, "sub_demand", reqOwner(account, c.AssignedTo), displayMap)
		node.URL = zentao.DemandViewURL(uint(c.ID))
		subByID[c.ID] = node
		if parent, ok := rootByID[c.Parent]; ok {
			parent.Children = append(parent.Children, node)
		}
	}
	// 树内研需挂到其直接父级需求（根或子业务均可能）
	for _, s := range stories {
		node := r.storyNode(s, taskMap[s.ID], productNames[s.Product], displayMap)
		if parent, ok := subByID[s.DemandID]; ok {
			parent.Children = append(parent.Children, node)
		} else if parent, ok := rootByID[s.DemandID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}
	// 独立研发需求单列根
	for _, s := range independent {
		node := r.storyNode(s, taskMap[s.ID], productNames[s.Product], displayMap)
		node.Independent = true
		out = append(out, node)
	}

	// 各级节点统计（子业务数 / 研发需求数），供前端父级汇总。
	r.computeNodeCounts(out)

	// 摘要：遍历叶节点（最细有效推进对象）
	summary = r.computeDemandSummary(out)
	return out, summary, nil
}

// computeNodeCounts 后序遍历为每个节点回填 subDemandCount（直接子业务数）与 storyCount（后代研需数）。
// 显式栈实现后序（子先于父），严禁递归。
func (r *Repo) computeNodeCounts(nodes []*BoardDemandItem) {
	// order 记录先序序列（父→子），逆序即后序（子→父），自底向上累计。
	order := make([]*BoardDemandItem, 0, len(nodes)*2)
	stack := make([]*BoardDemandItem, 0, len(nodes))
	for _, n := range nodes {
		stack = append(stack, n)
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			order = append(order, cur)
			for _, c := range cur.Children {
				stack = append(stack, c)
			}
		}
	}
	for i := len(order) - 1; i >= 0; i-- {
		n := order[i]
		subs, stories := 0, 0
		if n.Kind == "story" {
			stories = 1
		}
		for _, c := range n.Children {
			if c.Kind == "sub_demand" {
				subs++
			}
			subs += c.SubDemandCount
			stories += c.StoryCount
		}
		n.SubDemandCount = subs
		n.StoryCount = stories
	}
}

func (r *Repo) demandNode(row boardDemandRow, kind, owner string, displayMap map[string]string) *BoardDemandItem {
	dl := ""
	if row.Deadline != nil && !row.Deadline.IsZero() {
		dl = row.Deadline.Format("2006-01-02")
	}
	if o := displayMap[owner]; o != "" {
		owner = o
	}
	prefix := "US"
	return &BoardDemandItem{
		Kind: kind, ID: row.ID, DisplayID: fmt.Sprintf("%s%d", prefix, row.ID),
		Title: row.Name, Stage: deriveStageFromStatus(row.Status), Status: row.Status,
		Priority: priOf(row.Priority), Owner: owner, Deadline: dl,
		ActionLabel: deriveActionLabel(row.Status), Children: []*BoardDemandItem{},
	}
}

// storyNode 构造研发需求交付进展行。
func (r *Repo) storyNode(s boardStoryRow, agg taskAgg, productName string, displayMap map[string]string) *BoardDemandItem {
	owner := displayMap[s.AssignedTo]
	if owner == "" {
		owner = s.AssignedTo
	}
	currOwner := displayMap[agg.CurrOwner]
	if currOwner == "" {
		currOwner = owner
	}
	stage := deriveStoryStage(s.Status, s.Stage)
	progress := 0
	if agg.Total > 0 {
		progress = agg.Done * 100 / agg.Total
	}
	return &BoardDemandItem{
		Kind: "story", ID: s.ID, DisplayID: fmt.Sprintf("%d", s.ID),
		Title: s.Title, Stage: stage, Status: s.Status,
		Priority: priOf(s.Priority), Owner: owner, CurrentOwner: currOwner,
		ProductName: productName, Progress: progress,
		TaskTotal: agg.Total, TaskDone: agg.Done,
		ActionLabel: deriveStoryAction(s.Status, agg.Total),
		Children:    []*BoardDemandItem{},
		URL:         zentao.StoryViewURL(uint(s.ID)),
	}
}

// aggregateStoryTasks 按 story 聚合任务：总数 / 已完成 / 当前非完成负责人。
func (r *Repo) aggregateStoryTasks(ctx context.Context, groups ...[]boardStoryRow) (map[int64]taskAgg, error) {
	out := map[int64]taskAgg{}
	ids := map[int64]bool{}
	for _, g := range groups {
		for _, s := range g {
			ids[s.ID] = true
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	storyIDs := make([]int64, 0, len(ids))
	for id := range ids {
		storyIDs = append(storyIDs, id)
	}
	var rows []struct {
		StoryID    int64  `gorm:"column:story"`
		Status     string `gorm:"column:status"`
		AssignedTo string `gorm:"column:assignedTo"`
	}
	if err := r.db.WithContext(ctx).Table("zt_task").
		Where("story IN ?", storyIDs).
		Where("deleted = ?", "0").
		Where("status IN ?", []string{"wait", "doing", "done", "pause"}).
		Select("story, status, assignedTo").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, t := range rows {
		agg := out[t.StoryID]
		agg.Total++
		if t.Status == "done" {
			agg.Done++
		} else if agg.CurrOwner == "" && strings.TrimSpace(t.AssignedTo) != "" {
			agg.CurrOwner = t.AssignedTo
		}
		out[t.StoryID] = agg
	}
	return out, nil
}

// storyProductNames 解析研需所属产品名。
func (r *Repo) storyProductNames(ctx context.Context, groups ...[]boardStoryRow) (map[int64]string, error) {
	pids := map[int64]bool{}
	for _, g := range groups {
		for _, s := range g {
			if s.Product > 0 {
				pids[s.Product] = true
			}
		}
	}
	out := map[int64]string{}
	if len(pids) == 0 {
		return out, nil
	}
	ids := make([]int64, 0, len(pids))
	for id := range pids {
		ids = append(ids, id)
	}
	var rows []struct {
		ID   int64  `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := r.db.WithContext(ctx).Table("zt_product").
		Where("id IN ?", ids).
		Where("deleted = ?", "0").
		Select("id, name").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, pr := range rows {
		out[pr.ID] = pr.Name
	}
	return out, nil
}

// FindBoardTeamgroups 返回当前用户实际可参与的敏捷小组。
func (r *Repo) FindBoardTeamgroups(ctx context.Context, account string) ([]BoardTeamgroupOption, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	account = strings.TrimSpace(account)
	if account == "" {
		return []BoardTeamgroupOption{}, nil
	}
	rows := []struct {
		ID   int64  `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}{}
	if err := r.db.WithContext(ctx).Table("zt_teamgroup AS tg").
		Joins("INNER JOIN zt_team AS t ON t.root = tg.id AND t.type = ?", "teamgroup").
		Where("t.account = ? AND tg.deleted = ?", account, "0").
		Select("DISTINCT tg.id, tg.name").Order("tg.id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]BoardTeamgroupOption, 0, len(rows))
	for _, r := range rows {
		out = append(out, BoardTeamgroupOption{ID: uint(r.ID), Name: r.Name})
	}
	return out, nil
}

// FindBoardTeamgroupMembers 查询小组真实成员账号（用于任务按小组过滤负责人/任务）。
func (r *Repo) FindBoardTeamgroupMembers(ctx context.Context, teamgroupID uint) ([]string, error) {
	if r == nil || r.db == nil || teamgroupID == 0 {
		return nil, nil
	}
	var rows []struct {
		Account string `gorm:"column:account"`
	}
	if err := r.db.WithContext(ctx).Table("zt_team").
		Where("root = ? AND type = ?", teamgroupID, "teamgroup").
		Where("account <> ''").
		Select("DISTINCT account").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Account)
	}
	return out, nil
}

// FindBoardIssues 查询当前账号可见真实问题（创建或指派给自己），按 未解决/已关闭 聚类。
func (r *Repo) FindBoardIssues(ctx context.Context, account string) (*BoardIssueResp, error) {
	resp := &BoardIssueResp{Items: []BoardIssueItem{}}
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return resp, nil
	}
	rows := []struct {
		ID        int64  `gorm:"column:id"`
		Title     string `gorm:"column:title"`
		Priority  string `gorm:"column:pri"`
		Severity  string `gorm:"column:severity"`
		Status    string `gorm:"column:status"`
		CreatedBy string `gorm:"column:createdBy"`
	}{}
	if err := r.db.WithContext(ctx).Table("zt_issue").
		Where("deleted = ?", "0").
		Where("(createdBy = ? OR assignedTo = ?)", account, account).
		Order("id DESC").Limit(100).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		resp.Items = append(resp.Items, BoardIssueItem{
			ID: row.ID, Title: row.Title, Priority: row.Priority,
			Severity: row.Severity, Status: row.Status, CreatedBy: row.CreatedBy,
		})
		if row.Status == "closed" {
			resp.Closed++
		} else {
			resp.Open++
		}
	}
	resp.Total = int64(len(resp.Items))
	return resp, nil
}

// FindBoardIssueActions 查询问题从创建开始的禅道审计记录，并在同一 scope 内校验读取权限。
func (r *Repo) FindBoardIssueActions(ctx context.Context, account string, issueID, afterID int64) (*BoardIssueActionPage, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || issueID <= 0 {
		return nil, errorx.New(errorx.ErrCodeForbidden, "无权查看该问题")
	}
	var visible int64
	if err := r.db.WithContext(ctx).Table("zt_issue").
		Where("id = ? AND deleted = ?", issueID, "0").
		Where("(createdBy = ? OR assignedTo = ?)", account, account).
		Count(&visible).Error; err != nil {
		return nil, err
	}
	if visible == 0 {
		return nil, errorx.New(errorx.ErrCodeForbidden, "无权查看该问题")
	}
	rows := []struct {
		ID        int64      `gorm:"column:id"`
		Date      *time.Time `gorm:"column:date"`
		Actor     string     `gorm:"column:actor"`
		ActorName string     `gorm:"column:actor_name"`
		Action    string     `gorm:"column:action"`
		Extra     string     `gorm:"column:extra"`
		Comment   string     `gorm:"column:comment"`
	}{}
	if err := r.db.WithContext(ctx).Raw(`
SELECT a.id, a.date, a.actor, COALESCE(u.realname, a.actor) AS actor_name,
       a.action, a.extra, a.comment
FROM zt_action a
LEFT JOIN zt_user u ON u.account = a.actor AND u.deleted = '0'
WHERE a.objectType = 'issue' AND a.objectID = ?
  AND a.id > ?
ORDER BY a.date ASC, a.id ASC
LIMIT 201`, issueID, afterID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	hasMore := len(rows) > 200
	if hasMore {
		rows = rows[:200]
	}
	items := make([]BoardIssueAction, 0, len(rows))
	for _, row := range rows {
		date := ""
		if row.Date != nil {
			date = row.Date.Format("2006-01-02 15:04:05")
		}
		items = append(items, BoardIssueAction{ID: row.ID, Date: date, Actor: row.Actor, ActorName: row.ActorName, Action: row.Action, Extra: row.Extra, Comment: row.Comment})
	}
	page := &BoardIssueActionPage{Items: items}
	if hasMore && len(items) > 0 {
		page.NextAfterID = items[len(items)-1].ID
	}
	return page, nil
}
