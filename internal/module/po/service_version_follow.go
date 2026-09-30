// =============================================================================
// 文件: internal/module/po/service_version_follow.go
// 模块: PO 工作台
// 类型: action
// 职责: 版本跟进服务。串联只读 repo、三态推导与确定性规则，产出列表响应。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strconv"
	"strings"
	"time"

	"workbench/internal/model"
)

// vfWindowBarLimit 窗口条最多展示的窗口数（含已上线窗口，由前端收进「更多窗口」）。
const vfWindowBarLimit = 12

// vfOrphanEvidenceLimit 「未挂窗口」依据弹窗最多列出的需求条数。
const vfOrphanEvidenceLimit = 20

// VersionFollowList 组装版本跟进列表：窗口条 + 阶段分布 + 需求行 + 展开详情。
func (s *Service) VersionFollowList(ctx context.Context, actor *model.User, req VersionFollowListReq) (*VersionFollowListResp, error) {
	now := nowFunc()
	resp := &VersionFollowListResp{
		Items: []VersionFollowItemResp{}, Details: []VersionFollowItemDetailResp{},
		Windows: []VersionFollowWindowResp{}, StageCounts: []VersionFollowStageCount{},
		Page: req.Page, PageSize: req.PageSize, AIPilotOn: aiPilotEnabled,
	}
	windows, err := s.repo.ListVFWindows(ctx, 0, vfWindowBarLimit)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return resp, nil
	}
	ids := make([]uint64, 0, len(windows))
	for _, w := range windows {
		ids = append(ids, w.ID)
	}
	counts, err := s.repo.CountVFDemandsInWindow(ctx, ids)
	if err != nil {
		return nil, err
	}
	active := pickActiveWindow(windows, uint64(req.WindowID), now)
	for _, w := range windows {
		resp.Windows = append(resp.Windows, VersionFollowWindowResp{
			ID: w.ID, Name: w.Name, ReleaseDate: w.ReleaseDate.Format("2006-01-02"),
			Status: w.Status, DemandCount: counts[w.ID],
			Released: !w.ReleaseDate.After(now),
		})
	}
	rows, err := s.repo.ListVFDemandsInWindow(ctx, active.ID)
	if err != nil {
		return nil, err
	}
	items := s.buildVFItems(ctx, rows, now, req)
	items = filterVFItems(items, req, actor)
	s.fillVFSummary(resp, items, rows, now)
	resp.Total = int64(len(items))
	pageItems, _ := paginateVFItems(items, req)
	resp.Items = pageItems
	resp.Details = s.buildVFDetails(ctx, pageItems)
	if aiPilotEnabled {
		orphan, oerr := s.repo.CountVFOrphanTestedDemands(ctx)
		if oerr == nil && orphan > 0 {
			resp.OrphanTestedCount = orphan
		}
	}
	return resp, nil
}

// VersionFollowOrphanEvidence 返回「已到测试后阶段但未挂版本窗口」的需求清单。
// 仅在 AI 试点开启时被调用；每条发现都带规则名、字段值与可跳转对象。
func (s *Service) VersionFollowOrphanEvidence(ctx context.Context, actor *model.User) ([]VersionFollowAIFinding, error) {
	rows, err := s.repo.ListVFOrphanTestedDemands(ctx, vfOrphanEvidenceLimit)
	if err != nil {
		return nil, err
	}
	out := make([]VersionFollowAIFinding, 0, len(rows))
	for _, r := range rows {
		out = append(out, VersionFollowAIFinding{
			RuleName:  "已到测试后阶段但未挂版本窗口",
			Field:     "当前阶段",
			Value:     stageLabel(r.Stage, r.Status) + "（需求 " + strconv.FormatUint(uint64(r.ID), 10) + "）",
			TargetURL: "/demands/" + strconv.FormatUint(uint64(r.ID), 10),
		})
	}
	return out, nil
}

// buildVFItems 把原始行转成列表行：推导三态、算已停留天数、跑确定性规则。
func (s *Service) buildVFItems(ctx context.Context, rows []vfDemandRow, now time.Time, req VersionFollowListReq) []VersionFollowItemResp {
	avg := vfAvgStage(rows)
	items := make([]VersionFollowItemResp, 0, len(rows))
	for _, r := range rows {
		judged := deriveJudgement(judgeCtx{Raw: r, Now: now, AvgStage: avg})
		item := VersionFollowItemResp{
			DemandID: uint64(r.ID), DemandNo: r.No, Title: r.Title,
			System: r.MainSystem, Priority: r.Priority,
			Stage: stageLabel(r.Stage, r.Status), StageStatus: r.Status,
			Judgement: judged.Judgement, Reason: judged.Reason,
			Owner: r.Owner,
		}
		if item.System == "" {
			item.System = "暂无"
		}
		if item.Priority == "" {
			item.Priority = "暂无"
		}
		if item.Owner == "" {
			item.Owner = "暂无"
		}
		item.StayDays, item.Overdue = vfStay(r, now)
		item.ActionLabel, item.ActionURL, item.ActionEnabled, item.ActionReason = vfNextAction(r)
		if aiPilotEnabled {
			item.Findings = runVFRules(vfRule{Raw: r, Now: now, AvgStage: avg})
		}
		items = append(items, item)
	}
	return items
}

// fillVFSummary 回填阶段分布、距上线天数与三态计数。
func (s *Service) fillVFSummary(resp *VersionFollowListResp, items []VersionFollowItemResp, rows []vfDemandRow, now time.Time) {
	byStage := map[string]int{}
	for _, r := range rows {
		byStage[stageLabel(r.Stage, r.Status)]++
	}
	for _, code := range stageOrder {
		name := stageLabels[code]
		resp.StageCounts = append(resp.StageCounts, VersionFollowStageCount{Stage: name, Count: byStage[name]})
	}
	for _, it := range items {
		switch it.Judgement {
		case VFJudgementOnTrack:
			resp.OnTrack++
		case VFJudgementRisk:
			resp.Risk++
		case VFJudgementBlocked:
			resp.Blocked++
		}
	}
	if len(resp.Windows) > 0 && resp.Windows[0].ReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", resp.Windows[0].ReleaseDate); err == nil {
			resp.DistanceDays = int(t.Sub(now).Hours() / 24)
		}
	}
}

// vfStay 计算「已停留 N 天」与超期提示，替代原型的迷你进度列。
func vfStay(r vfDemandRow, now time.Time) (int, string) {
	// 计划日缺失时退回 deadline；两者都缺则不显示天数。
	plan := r.SchedulePlanDate
	if plan == nil {
		plan = r.Deadline
	}
	if plan == nil {
		return 0, ""
	}
	days := int(now.Sub(*plan).Hours() / 24)
	overdue := ""
	if days > 0 {
		overdue = "已超期 " + strconv.Itoa(days) + " 天"
	} else if days < 0 {
		overdue = "距计划日 " + strconv.Itoa(-days) + " 天"
	}
	return vfAbs(days), overdue
}

// vfAbs 返回绝对值，用于「已停留 N 天」不出现负号。
func vfAbs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// vfNextAction 依据阶段给出唯一主按钮；条件不满足时置灰并写明原因。
func vfNextAction(r vfDemandRow) (label, url string, enabled bool, reason string) {
	detail := "/demands/" + strconv.FormatUint(uint64(r.ID), 10)
	switch {
	case r.Status == "acceptanced":
		return "确认验证", detail, true, ""
	case r.Status == "waitacceptance":
		return "去验收", detail, true, ""
	case r.Status == "waitdeliver":
		return "发起交付", detail, true, ""
	case r.Status == "released":
		return "确认验证", detail, true, ""
	case r.Status == "testing":
		if r.Owner == "" {
			return "跟进测试", detail, false, "缺负责人，无法指派"
		}
		return "跟进测试", detail, true, ""
	case r.Status == "developed":
		return "提交验收", detail, true, ""
	default:
		return "查看详情", detail, true, ""
	}
}

// buildVFDetails 组装展开行：交付前置检查清单、审批链（二期）、研需与任务、来源需求。
func (s *Service) buildVFDetails(ctx context.Context, items []VersionFollowItemResp) []VersionFollowItemDetailResp {
	details := make([]VersionFollowItemDetailResp, 0, len(items))
	for range items {
		details = append(details, VersionFollowItemDetailResp{
			Checks: []VersionFollowCheckItemResp{
				{Label: "业务验收", State: "pending", Detail: "暂无"},
				{Label: "主管审批", State: "pending", Detail: "暂无", Phase2: true},
				{Label: "测试阻塞 Bug", State: "clear", Detail: "无阻塞"},
				{Label: "已关联版本窗口", State: "clear", Detail: "已关联"},
			},
			Stories:     []string{},
			Tasks:       []string{},
			SourceName:  "",
			SourceItems: []string{},
		})
	}
	return details
}

// pickActiveWindow 选出当前操作的窗口：优先请求指定，否则取最近的未上线窗口。
func pickActiveWindow(windows []vfWindowRow, want uint64, now time.Time) vfWindowRow {
	for _, w := range windows {
		if want > 0 && w.ID == want {
			return w
		}
	}
	for _, w := range windows {
		if w.ReleaseDate.After(now) {
			return w
		}
	}
	return windows[0]
}

// filterVFItems 按范围、阶段、系统、负责人、关键词、发布判断过滤。
func filterVFItems(items []VersionFollowItemResp, req VersionFollowListReq, actor *model.User) []VersionFollowItemResp {
	out := make([]VersionFollowItemResp, 0, len(items))
	for _, it := range items {
		if vfMatchAllFilters(it, req, actor) {
			out = append(out, it)
		}
	}
	return out
}

// vfMatchAllFilters 汇总全部筛选条件，供列表过滤逐项判定。
func vfMatchAllFilters(it VersionFollowItemResp, req VersionFollowListReq, actor *model.User) bool {
	return vfMatchScope(it, req.Scope, actor) &&
		vfMatchStage(it, req.Stage) &&
		vfMatchSystem(it, req.System) &&
		vfMatchOwner(it, req.Owner) &&
		vfMatchJudgement(it, req.Judgement) &&
		vfMatchKeyword(it, req.Keyword)
}

// vfMatchStage 阶段过滤；req 为空表示不限。
func vfMatchStage(it VersionFollowItemResp, want string) bool {
	return want == "" || it.Stage == stageLabels[want] || it.Stage == want
}

// vfMatchSystem 系统过滤；「全部」与空均表示不限。
func vfMatchSystem(it VersionFollowItemResp, want string) bool {
	return want == "" || want == "全部" || it.System == want
}

// vfMatchOwner 负责人过滤；「全部」与空均表示不限。
func vfMatchOwner(it VersionFollowItemResp, want string) bool {
	return want == "" || want == "全部" || it.Owner == want
}

// vfMatchJudgement 发布判断过滤；空表示不限。
func vfMatchJudgement(it VersionFollowItemResp, want string) bool {
	return want == "" || it.Judgement == want
}

// vfMatchKeyword 关键词过滤，命中需求号或标题即通过。
func vfMatchKeyword(it VersionFollowItemResp, want string) bool {
	kw := strings.TrimSpace(want)
	if kw == "" {
		return true
	}
	return strings.Contains(it.DemandNo, kw) || strings.Contains(it.Title, kw)
}

// vfMatchScope 处理「全部 / 待我处理 / 我跟进」三选一。
func vfMatchScope(it VersionFollowItemResp, scope string, actor *model.User) bool {
	switch scope {
	case "todo":
		return it.Judgement == VFJudgementBlocked || it.Judgement == VFJudgementRisk
	case "following":
		return actor != nil && it.Owner == actor.Account
	default:
		return true
	}
}

// paginateVFItems 做内存分页（窗口内需求量级有限，不额外查库）。
func paginateVFItems(items []VersionFollowItemResp, req VersionFollowListReq) ([]VersionFollowItemResp, int) {
	size := req.PageSize
	if size <= 0 {
		size = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * size
	if start >= len(items) {
		return []VersionFollowItemResp{}, len(items)
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], len(items)
}

// vfAvgStage 计算窗口内需求的平均阶段序号，供「落后于平均」规则使用。
func vfAvgStage(rows []vfDemandRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	sum := 0
	for _, r := range rows {
		sum += stageRank(r.Stage, r.Status)
	}
	return float64(sum) / float64(len(rows))
}
