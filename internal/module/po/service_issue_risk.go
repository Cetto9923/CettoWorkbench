package po

import (
	"context"
	"strconv"
	"strings"
	"time"
	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

// irOpenStatuses / irClosedStatuses 与现有 issueRiskClosed 互补：
// open = active/tracked/wait/unconfirmed/doing/confirmed；closed = resolved/closed/cancel/canceled。
// 集中维护，供 Repo 层 WHERE status IN ? 与 Service 层 LoopCounts 复用。
var (
	irOpenStatuses   = []string{"active", "tracked", "wait", "unconfirmed", "doing", "confirmed"}
	irClosedStatuses = []string{"resolved", "closed", "cancel", "canceled"}
)

// IssueRiskProject 用于 IssuesRiskList 响应中的项目下拉选项，列表按 p.name 升序，最多 100 条。
type IssueRiskProject struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

func (s *Service) IssueRiskList(ctx context.Context, actor *model.User, req IssueRiskListReq) (*IssueRiskListResp, error) {
	resp := &IssueRiskListResp{
		Items:      []IssueRiskItem{},
		KindCounts: map[string]int64{"issue": 0, "risk": 0},
		LoopCounts: map[string]int64{"all": 0, "open": 0, "closed": 0},
		Projects:   []IssueRiskProject{},
	}
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return resp, nil
	}
	rows, total, err := s.repo.FindIssueRiskList(ctx, actor.Account, req)
	if err != nil {
		return nil, err
	}
	resp.Total = total
	for _, kind := range []string{"issue", "risk"} {
		countReq := req
		countReq.Kind = kind
		count, countErr := s.repo.CountIssueRisk(ctx, actor.Account, countReq)
		if countErr != nil {
			return nil, countErr
		}
		resp.KindCounts[kind] = count
	}
	for _, loop := range []string{"open", "closed", "all"} {
		loopReq := req
		loopReq.Loop = loop
		loopReq.Overdue = false
		loopReq.Page = 1
		loopReq.PageSize = 1
		cnt, loopErr := s.repo.CountIssueRisk(ctx, actor.Account, loopReq)
		if loopErr != nil {
			return nil, loopErr
		}
		resp.LoopCounts[loop] = cnt
	}
	if req.Loop == "all" {
		overReq := req
		overReq.Loop = "open"
		overReq.Overdue = true
		overReq.Page = 1
		overReq.PageSize = 1
		cnt, overErr := s.repo.CountIssueRisk(ctx, actor.Account, overReq)
		if overErr != nil {
			return nil, overErr
		}
		resp.OverdueCount = cnt
	}
	projectReq := req
	projectReq.Project = 0
	projects, projErr := s.repo.CountIssueRiskProjects(ctx, actor.Account, projectReq)
	if projErr != nil {
		return nil, projErr
	}
	resp.Projects = projects
	for _, row := range rows {
		resp.Items = append(resp.Items, mapIssueRiskRow(row, req.Kind))
	}
	return resp, nil
}
func mapIssueRiskRow(row issueRiskRow, kind string) IssueRiskItem {
	url := zentao.URL("issue", "view", "issueID="+strconv.FormatInt(row.ID, 10))
	if kind == "risk" {
		url = zentao.URL("risk", "view", "riskID="+strconv.FormatInt(row.ID, 10))
	}
	created, plan := normalizeIssueRiskDate(row.CreatedDate), normalizeIssueRiskDate(row.PlanDate)
	days := 0
	if t, e := time.Parse("2006-01-02", created); e == nil {
		days = int(time.Since(t).Hours() / 24)
		if days < 0 {
			days = 0
		}
	}
	overdue, od := false, 0
	if t, e := time.Parse("2006-01-02", plan); e == nil && !issueRiskClosed(kind, row.Status) && t.Before(time.Now().Truncate(24*time.Hour)) {
		overdue = true
		od = int(time.Since(t).Hours() / 24)
		if od < 1 {
			od = 1
		}
	}
	handler := row.HandlerName
	if issueRiskClosed(kind, row.Status) {
		if r := strings.TrimSpace(row.ResolvedName); r != "" && !strings.EqualFold(r, "closed") {
			handler = r
		} else if c := strings.TrimSpace(row.ClosedName); c != "" && !strings.EqualFold(c, "closed") {
			handler = c
		} else if h := strings.TrimSpace(row.HandlerName); h != "" && !strings.EqualFold(h, "closed") {
			handler = h
		} else if cr := strings.TrimSpace(row.CreatorName); cr != "" {
			handler = cr
		} else {
			handler = "—"
		}
	} else {
		if strings.EqualFold(strings.TrimSpace(handler), "closed") {
			handler = "待分配"
		}
	}
	if strings.EqualFold(strings.TrimSpace(handler), "closed") {
		handler = "—"
	}

	return IssueRiskItem{ID: row.ID, DisplayID: strconv.FormatInt(row.ID, 10), Kind: kind, Title: row.Title, Project: row.ProjectName, Severity: issueRiskSeverity(row.Severity), Priority: priorityLabel(row.Pri), Handler: handler, Submitter: row.CreatorName, Status: issueRiskStatusLabel(row.Status), StatusCode: row.Status, CreatedDate: created, PlanDate: plan, Days: days, IsOverdue: overdue, OverdueDays: od, URL: url}
}
func normalizeIssueRiskDate(raw string) string {
	if len(raw) >= 10 && raw[:10] != "0000-00-00" {
		return raw[:10]
	}
	return ""
}
func issueRiskClosed(kind, status string) bool {
	if kind == "risk" {
		return status == "closed" || status == "cancel" || status == "canceled"
	}
	return status == "resolved" || status == "closed" || status == "cancel" || status == "canceled"
}
func issueRiskStatusLabel(status string) string {
	m := map[string]string{"active": "激活", "tracked": "跟踪中", "hangup": "已挂起", "unconfirmed": "未处理", "wait": "未处理", "doing": "处理中", "confirmed": "处理中", "resolved": "已解决", "closed": "已关闭", "cancel": "已取消", "canceled": "已取消"}
	if v := m[status]; v != "" {
		return v
	}
	return "未处理"
}
func issueRiskSeverity(raw string) string {
	switch strings.TrimSpace(raw) {
	case "1":
		return "致命"
	case "2":
		return "严重"
	default:
		return "一般"
	}
}
func priorityLabel(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "-"
	}
	return "P" + raw
}
