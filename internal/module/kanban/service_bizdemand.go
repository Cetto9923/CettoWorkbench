// =============================================================================
// 文件: internal/module/kanban/service_bizdemand.go
// 模块: 工作看板
// 类型: readonly
// 职责: 复用首页价值流 Demands，按选中负责人组装需求树（业需 + 独立研需）。
// 依赖: internal/model
//       internal/module/po
//       internal/pkg/zentao
// =============================================================================

package kanban

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"workbench/internal/model"
	"workbench/internal/module/po"
	"workbench/internal/pkg/zentao"
)

const (
	bizDemandPageSize   = 100
	maxDemandFetchPages = 10 // 限制拉取上限（1000条），在保证卡片不被截断的同时防止大表级联扫描超时
)

// ListValueStreamBizDemands 按选中负责人拉取价值流「全部」业需与研需。
// account 为空时回落到 actor；account=all 时按当前敏捷小组全员聚合去重。
func (s *Service) ListValueStreamBizDemands(ctx context.Context, actor *model.User, req ListDemandsReq) (ListBizDemandsResp, error) {
	if s.poSvc == nil {
		return ListBizDemandsResp{Items: []BizDemandItem{}}, nil
	}
	actorAccount := ""
	if actor != nil {
		actorAccount = actor.Account
	}

	groups, err := s.ListMyTeamgroups(ctx, actor)
	if err != nil {
		return ListBizDemandsResp{}, err
	}
	targets, err := resolveKanbanAccounts(actorAccount, req.Account, req.TeamgroupID, groups)
	if err != nil {
		return ListBizDemandsResp{}, err
	}
	if len(targets) == 0 {
		return ListBizDemandsResp{Items: []BizDemandItem{}}, nil
	}

	seen := make(map[string]struct{})
	var all []po.WorkItemDetail
	for _, target := range targets {
		items, demErr := s.fetchDemandsForAccount(ctx, viewAsUser(actor, target))
		if demErr != nil {
			return ListBizDemandsResp{}, demErr
		}
		for _, it := range items {
			key := demandItemKey(it.Kind, it.ID)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			all = append(all, it)
		}
	}

	items := toBizDemandItems(all)
	if s.repo != nil {
		indStories, indErr := s.repo.FindIndependentStoriesByAccounts(ctx, targets)
		if indErr != nil {
			return ListBizDemandsResp{}, indErr
		}
		if len(indStories) > 0 {
			displayMap, _ := s.loadAccountDisplayMap(ctx, actor)
			for _, st := range indStories {
				numIDStr := strconv.FormatInt(st.ID, 10)
				key := demandItemKey("story", numIDStr)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}

				owner := lookupDisplay(displayMap, st.AssignedTo)
				pri := ""
				if st.Pri > 0 {
					pri = fmt.Sprintf("P%d", st.Pri)
				}
				items = append(items, BizDemandItem{
					Kind:         "story",
					ID:           fmt.Sprintf("U%d", st.ID),
					Pri:          pri,
					Title:        st.Title,
					Owner:        owner,
					OwnerAccount: st.AssignedTo,
					ValueStream:  deriveStoryStage(st.Status, st.Stage, st.DevelopFinish, st.TestFinish, st.VerifyFinish, st.DeliverDate),
					ZentaoUrl:    zentao.URL("story", "view", "storyID="+numIDStr),
					ZentaoStatus: st.Status,
				})
			}
		}
	}
	if err := s.enrichDemandCounts(ctx, items); err != nil {
		return ListBizDemandsResp{}, err
	}
	summary := computeDemandSummary(items)
	memberCounts := computeMemberCounts(items)
	return ListBizDemandsResp{
		Items:        items,
		Summary:      summary,
		MemberCounts: memberCounts,
	}, nil
}

// deriveStoryStage 严格按照 PRD《价值流阶段数据统计逻辑.xlsx》将研发需求派生到看板 4 列（研需无受理/澄清）：
// 1. 交付/评价：deliverDate <= 今天，或处于 launched(已发布) 状态，或 stage 为 delivering/delivered/released
// 2. 联调/验收：stage 为 testing/tested/verified，或开发、测试、验证三段完成时间均已填写
// 3. 研发/提测：status 为 developing，或 stage 为 developing/developed
// 4. 排期（默认）：未完整填写完成时间或处于 wait/planned/projected 等状态，统一归入排期
func deriveStoryStage(status, stage string, developFinish, testFinish, verifyFinish, deliverDate *time.Time) string {
	if isEffectiveDateBeforeOrEqualToday(deliverDate) {
		return "交付/评价"
	}
	if status == "launched" || stage == "delivering" || stage == "delivered" || stage == "released" {
		return "交付/评价"
	}

	if stage == "testing" || stage == "tested" || stage == "verified" {
		return "联调/验收"
	}
	if isEffectiveDate(developFinish) && isEffectiveDate(testFinish) && isEffectiveDate(verifyFinish) {
		return "联调/验收"
	}

	if status == "developing" || stage == "developing" || stage == "developed" {
		return "研发/提测"
	}

	return "排期"
}

func isEffectiveDate(t *time.Time) bool {
	return t != nil && !t.IsZero() && t.Format("2006-01-02") != "0000-00-00"
}

func isEffectiveDateBeforeOrEqualToday(t *time.Time) bool {
	if !isEffectiveDate(t) {
		return false
	}
	today := time.Now().Format("2006-01-02")
	return t.Format("2006-01-02") <= today
}

func rawWorkItemID(id string) string {
	raw := strings.TrimSpace(id)
	raw = strings.TrimPrefix(raw, "#")
	upper := strings.ToUpper(raw)
	for _, p := range []string{"US", "U", "REQ", "SUB", "RD"} {
		if strings.HasPrefix(upper, p) {
			raw = raw[len(p):]
			upper = upper[len(p):]
			break
		}
	}
	return strings.TrimLeft(strings.TrimSpace(raw), "-")
}

func demandItemKey(kind, id string) string {
	return strings.ToLower(strings.TrimSpace(kind)) + ":" + rawWorkItemID(id)
}

func (s *Service) fetchDemandsForAccount(ctx context.Context, viewAs *model.User) ([]po.WorkItemDetail, error) {
	var all []po.WorkItemDetail
	page := 1
	for page <= maxDemandFetchPages {
		dreq := po.DemandsReq{Status: "all", Page: page, PageSize: bizDemandPageSize}
		dreq.Normalize()
		resp, demErr := s.poSvc.Demands(ctx, viewAs, dreq)
		if demErr != nil {
			return nil, demErr
		}
		if resp == nil || len(resp.Items) == 0 {
			break
		}
		all = append(all, resp.Items...)
		if int64(page*bizDemandPageSize) >= resp.Total || len(resp.Items) < bizDemandPageSize {
			break
		}
		page++
	}
	return all, nil
}

func viewAsUser(actor *model.User, account string) *model.User {
	if actor == nil {
		return &model.User{Account: account}
	}
	u := *actor
	u.Account = account
	if account != strings.TrimSpace(actor.Account) {
		u.DisplayName = ""
	}
	return &u
}

func toBizDemandItems(items []po.WorkItemDetail) []BizDemandItem {
	out := make([]BizDemandItem, 0, len(items))
	for _, it := range items {
		out = append(out, BizDemandItem{
			Kind:         it.Kind,
			ID:           it.ID,
			Pri:          it.Pri,
			Title:        it.Title,
			Owner:        it.Owner,
			OwnerAccount: it.AssignedTo,
			ValueStream:  it.ValueStream,
			ZentaoUrl:    it.ZentaoUrl,
			ZentaoStatus: it.ZentaoStatus,
		})
	}
	return out
}

func computeDemandSummary(items []BizDemandItem) DemandSummary {
	var sum DemandSummary
	for _, it := range items {
		vs := it.ValueStream
		if strings.Contains(vs, "受理") || strings.Contains(vs, "澄清") {
			sum.Clarify++
		}
		if strings.Contains(vs, "排期") {
			sum.Schedule++
		}
		// 阻塞：status 为 suspended (挂起) 或 refuse (已驳回)
		if it.ZentaoStatus == "suspended" || it.ZentaoStatus == "refuse" {
			sum.Blocked++
		}
	}
	return sum
}

func computeMemberCounts(items []BizDemandItem) map[string]int {
	counts := make(map[string]int)
	for _, it := range items {
		acc := strings.TrimSpace(it.OwnerAccount)
		if acc != "" {
			counts[acc]++
		}
	}
	return counts
}

func (s *Service) enrichDemandCounts(ctx context.Context, items []BizDemandItem) error {
	if s.repo == nil || len(items) == 0 {
		return nil
	}
	var demandIDs []int64
	var storyIDs []int64
	demandIdxMap := make(map[int64][]int)
	storyIdxMap := make(map[int64][]int)

	for i, it := range items {
		numID, err := strconv.ParseInt(rawWorkItemID(it.ID), 10, 64)
		if err != nil || numID <= 0 {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(it.Kind))
		if kind == "demand" || kind == "business" || kind == "sub_demand" {
			if _, exists := demandIdxMap[numID]; !exists {
				demandIDs = append(demandIDs, numID)
			}
			demandIdxMap[numID] = append(demandIdxMap[numID], i)
		} else if kind == "story" || kind == "independent_story" {
			if _, exists := storyIdxMap[numID]; !exists {
				storyIDs = append(storyIDs, numID)
			}
			storyIdxMap[numID] = append(storyIdxMap[numID], i)
		}
	}

	if len(demandIDs) > 0 {
		storyCounts, err := s.repo.FindStoryCountsByDemands(ctx, demandIDs)
		if err != nil {
			return err
		}
		for did, cnt := range storyCounts {
			for _, idx := range demandIdxMap[did] {
				items[idx].StoryCount = cnt
			}
		}
	}
	if len(storyIDs) > 0 {
		taskCounts, err := s.repo.FindTaskCountsByStories(ctx, storyIDs)
		if err != nil {
			return err
		}
		for sid, counts := range taskCounts {
			for _, idx := range storyIdxMap[sid] {
				items[idx].TaskDone = counts[0]
				items[idx].TaskTotal = counts[1]
			}
		}
	}
	return nil
}
