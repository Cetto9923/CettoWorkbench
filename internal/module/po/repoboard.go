// =============================================================================
// 文件: internal/module/po/repoboard.go
// 模块: PO 工作台
// 类型: action
// 职责: 需求看板数据访问。V1.3 原则：阶段不可拖拽（状态派生）。
//       树形：根节点 = actor scope 内的 zt_demand；子节点 = zt_demand.parent=父ID；下钻 = zt_story 摘要计数。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FindBoardDemandTree 查需求树：actor scope 内业务需求 + 子业务 + 研需下钻摘要计数。
func (r *Repo) FindBoardDemandTree(ctx context.Context, req BoardDemandReq) ([]*BoardDemandItem, BoardDemandSummary, error) {
	summary := BoardDemandSummary{}
	if r == nil || r.db == nil || strings.TrimSpace(req.POAccount) == "" {
		return nil, summary, nil
	}

	type rootRow struct {
		ID         int64      `gorm:"column:id"`
		Parent     int64      `gorm:"column:parent"`
		Name       string     `gorm:"column:name"`
		Status     string     `gorm:"column:status"`
		Priority   string     `gorm:"column:pri"`
		AssignedTo string     `gorm:"column:assignedTo"`
		Deadline   *time.Time `gorm:"column:deadline"`
	}
	var roots []rootRow
	q := r.db.WithContext(ctx).Table("zt_demand").
		Where(`id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
			OR QD = ? OR RD = ? OR BRA = ?`, req.POAccount, req.POAccount, req.POAccount, req.POAccount).
		Where("parent = ?", 0).
		Where("status NOT IN ?", []string{"closed", "cancel"})
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
		return nil, summary, nil
	}

	rootIDs := make([]int64, 0, len(roots))
	for _, rr := range roots {
		rootIDs = append(rootIDs, rr.ID)
	}

	// 摘要计数
	var cntSummary struct {
		Total         int64
		PendingReview int64
		Blocked       int64
		Overdue       int64
	}
	_ = r.db.WithContext(ctx).Table("zt_demand").
		Where("id IN ?", rootIDs).
		Count(&cntSummary.Total)
	_ = r.db.WithContext(ctx).Table("zt_demand").
		Where("id IN ?", rootIDs).
		Where("status = ?", "active").
		Count(&cntSummary.PendingReview)
	_ = r.db.WithContext(ctx).Table("zt_demand").
		Where("id IN ?", rootIDs).
		Where("status IN ?", []string{"hang", "refuse"}).
		Count(&cntSummary.Blocked)
	now := time.Now()
	_ = r.db.WithContext(ctx).Table("zt_demand").
		Where("id IN ?", rootIDs).
		Where("deadline IS NOT NULL AND deadline < ?", now).
		Count(&cntSummary.Overdue)
	summary = BoardDemandSummary{
		Total: cntSummary.Total, PendingReview: cntSummary.PendingReview,
		Blocked: cntSummary.Blocked, Overdue: cntSummary.Overdue,
	}

	// 子业务（parent != 0）
	type childRow struct {
		ID         int64      `gorm:"column:id"`
		Parent     int64      `gorm:"column:parent"`
		Name       string     `gorm:"column:name"`
		Status     string     `gorm:"column:status"`
		Priority   string     `gorm:"column:pri"`
		AssignedTo string     `gorm:"column:assignedTo"`
		Deadline   *time.Time `gorm:"column:deadline"`
	}
	var children []childRow
	_ = r.db.WithContext(ctx).Table("zt_demand").
		Where("parent IN ?", rootIDs).
		Where("deleted = ?", "0").
		Where("status NOT IN ?", []string{"closed", "cancel"}).
		Find(&children).Error

	// 研需（按 story.demand 关联父业务需求 + 子业务）
	type storyRow struct {
		ID       int64  `gorm:"column:id"`
		Title    string `gorm:"column:title"`
		DemandID int64  `gorm:"column:demand"`
		Status   string `gorm:"column:status"`
	}
	demandIDs := rootIDs
	for _, c := range children {
		demandIDs = append(demandIDs, c.ID)
	}
	var stories []storyRow
	if len(demandIDs) > 0 {
		_ = r.db.WithContext(ctx).Table("zt_story").
			Where("demand IN ?", demandIDs).
			Where("deleted = ?", "0").
			Select("id, title, demand, status").
			Find(&stories).Error
	}
	storyCountByDemand := map[int64]int{}
	for _, s := range stories {
		storyCountByDemand[s.DemandID]++
	}

	// 任务开放数（按需求聚合）：先取 storyIDs 拼 IN 子句
	taskOpenByDemand := map[int64]int{}
	if len(stories) > 0 {
		storyIDs := make([]int64, 0, len(stories))
		for _, s := range stories {
			storyIDs = append(storyIDs, s.ID)
		}
		var tqRows []struct {
			StoryID  int64 `gorm:"column:storyID"`
			DemandID int64 `gorm:"column:demandID"`
		}
		_ = r.db.WithContext(ctx).Table("zt_task").
			Where("story IN ?", storyIDs).
			Where("deleted = ?", "0").
			Where("status NOT IN ?", []string{"done", "closed", "cancel"}).
			Select("story, demand").
			Find(&tqRows).Error
		for _, tr := range tqRows {
			taskOpenByDemand[tr.DemandID]++
		}
	}

	// 拼树
	rootByID := map[int64]*BoardDemandItem{}
	out := make([]*BoardDemandItem, 0, len(roots))
	for _, rr := range roots {
		dl := ""
		if rr.Deadline != nil {
			dl = rr.Deadline.Format("2006-01-02")
		}
		owner := rr.AssignedTo
		if owner == "" {
			owner = req.POAccount
		}
		rootByID[rr.ID] = &BoardDemandItem{
			Kind: "demand", ID: rr.ID, DisplayID: fmt.Sprintf("BR-%d", rr.ID),
			Title: rr.Name, Stage: deriveStageFromStatus(rr.Status), Status: rr.Status,
			Priority: rr.Priority, Owner: owner, Deadline: dl,
			ActionLabel: deriveActionLabel(rr.Status),
			Children:    []*BoardDemandItem{}, URL: fmt.Sprintf("http://10.211.55.4:8080/demand-view-%d.html", rr.ID),
		}
		out = append(out, rootByID[rr.ID])
	}
	for _, c := range children {
		dl := ""
		if c.Deadline != nil {
			dl = c.Deadline.Format("2006-01-02")
		}
		node := &BoardDemandItem{
			Kind: "sub_demand", ID: c.ID, DisplayID: fmt.Sprintf("BR-%d", c.ID),
			Title: c.Name, Stage: deriveStageFromStatus(c.Status), Status: c.Status,
			Priority: c.Priority, Owner: c.AssignedTo, Deadline: dl,
			StoryCount:    storyCountByDemand[c.ID],
			TaskOpenCount: taskOpenByDemand[c.ID],
			ActionLabel:   deriveActionLabel(c.Status),
			Children:      []*BoardDemandItem{}, URL: fmt.Sprintf("http://10.211.55.4:8080/demand-view-%d.html", c.ID),
		}
		if parent, ok := rootByID[c.Parent]; ok {
			parent.Children = append(parent.Children, node)
		}
	}
	return out, summary, nil
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

// deriveStageFromStatus 状态映射 V1.3 五大阶段（不可拖拽，状态派生）。
func deriveStageFromStatus(s string) string {
	switch s {
	case "draft", "wait", "refuse":
		return "受理/澄清"
	case "active":
		return "受理/澄清"
	case "clarified":
		return "排期"
	case "developing":
		return "研发/提测"
	case "testing":
		return "联调/验收"
	case "waitacceptance":
		return "联调/验收"
	case "acceptanced", "waitdeliver":
		return "交付/评价"
	case "released":
		return "交付/评价"
	}
	return s
}

// deriveActionLabel 顶部主操作按钮文本（V1.3 节点级操作）。
func deriveActionLabel(s string) string {
	switch s {
	case "draft", "wait", "refuse":
		return "去梳理"
	case "active":
		return "去梳理"
	case "clarified":
		return "去排期"
	case "developing", "testing", "waitacceptance":
		return "查看任务"
	case "acceptanced":
		return "跟进发布"
	}
	return "查看任务"
}
