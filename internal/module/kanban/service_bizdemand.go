// =============================================================================
// 文件: internal/module/kanban/service_bizdemand.go
// 模块: 工作看板
// 类型: readonly
// 职责: 复用首页价值流 Demands，按选中负责人组装需求树（业需 + 独立研需）。
// 依赖: internal/model
//       internal/module/po
//       internal/pkg/errorx
// =============================================================================

package kanban

import (
	"context"
	"strconv"
	"strings"

	"workbench/internal/model"
	"workbench/internal/module/po"
	"workbench/internal/pkg/zentao"
)

const bizDemandPageSize = 100

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
			key := it.Kind + ":" + it.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			all = append(all, it)
		}
	}

	items := toBizDemandItems(all)
	if s.repo != nil && len(targets) > 0 {
		if indStories, indErr := s.repo.FindIndependentStoriesByAccounts(ctx, targets); indErr == nil && len(indStories) > 0 {
			var displayMap map[string]string
			if s.userSvc != nil {
				displayMap, _ = s.userSvc.AccountDisplayMap(ctx, actor)
			}
			for _, st := range indStories {
				key := "story:" + strconv.FormatInt(st.ID, 10)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				owner := st.AssignedTo
				if owner == "" {
					owner = st.OpenedBy
				}
				if disp, exists := displayMap[owner]; exists && disp != "" {
					owner = disp
				}
				pri := ""
				if st.Pri != "" && st.Pri != "0" {
					pri = "P" + st.Pri
				}
				items = append(items, BizDemandItem{
					Kind:         "story",
					ID:           strconv.FormatInt(st.ID, 10),
					Pri:          pri,
					Title:        st.Title,
					Owner:        owner,
					ValueStream:  deriveStoryStage(st.Status, st.Stage),
					ZentaoUrl:    zentao.URL("story", "view", "storyID="+strconv.FormatInt(st.ID, 10)),
					ZentaoStatus: st.Status,
				})
			}
		}
	}
	s.enrichDemandCounts(ctx, items)
	return ListBizDemandsResp{Items: items}, nil
}

func deriveStoryStage(status, stage string) string {
	switch status {
	case "draft", "wait", "active":
		if stage == "" || stage == "wait" {
			return "受理/澄清"
		}
	case "clarified", "planned", "projected", "designed", "designing":
		return "排期"
	case "developing", "developed":
		return "研发/提测"
	case "testing", "tested", "verified", "reviewing":
		return "联调/验收"
	case "delivering", "delivered", "releasing", "released":
		return "交付/评价"
	}
	switch stage {
	case "developing", "developed":
		return "研发/提测"
	case "testing", "tested", "verified":
		return "联调/验收"
	case "delivering", "delivered", "released":
		return "交付/评价"
	}
	return "受理/澄清"
}


func (s *Service) fetchDemandsForAccount(ctx context.Context, viewAs *model.User) ([]po.WorkItemDetail, error) {
	dreq := po.DemandsReq{Status: "all", Page: 1, PageSize: bizDemandPageSize}
	dreq.Normalize()
	resp, demErr := s.poSvc.Demands(ctx, viewAs, dreq)
	if demErr != nil {
		return nil, demErr
	}
	if resp == nil || len(resp.Items) == 0 {
		return nil, nil
	}
	return resp.Items, nil
}

func (s *Service) enrichDemandCounts(ctx context.Context, items []BizDemandItem) {
	if s.repo == nil || len(items) == 0 {
		return
	}
	var demandIDs []int64
	var storyIDs []int64
	demandIdxMap := make(map[int64][]int)
	storyIdxMap := make(map[int64][]int)

	for i, it := range items {
		numID, err := strconv.ParseInt(extractNumericID(it.ID), 10, 64)
		if err != nil || numID <= 0 {
			continue
		}
		kind := strings.ToLower(it.Kind)
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
		if storyCounts, err := s.repo.FindStoryCountsByDemands(ctx, demandIDs); err == nil {
			for did, cnt := range storyCounts {
				for _, idx := range demandIdxMap[did] {
					items[idx].StoryCount = cnt
				}
			}
		}
	}

	if len(storyIDs) > 0 {
		if taskStats, err := s.repo.FindTaskStatsByStories(ctx, storyIDs); err == nil {
			for sid, stats := range taskStats {
				for _, idx := range storyIdxMap[sid] {
					items[idx].TaskDone = stats[0]
					items[idx].TaskTotal = stats[1]
				}
			}
		}
	}
}

func extractNumericID(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "#")
	for _, prefix := range []string{"US", "REQ", "SUB", "RD", "U", "us", "req", "sub", "rd", "u"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			s = strings.TrimPrefix(s, "-")
			break
		}
	}
	return s
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
			ValueStream:  it.ValueStream,
			ZentaoUrl:    it.ZentaoUrl,
			ZentaoStatus: it.ZentaoStatus,
		})
	}
	return out
}
