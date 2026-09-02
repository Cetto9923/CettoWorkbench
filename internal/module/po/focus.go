// =============================================================================
// 文件: internal/module/po/focus.go
// 模块: PO 工作台
// 类型: action
// 职责: 实现 PO 工作台首页"今日推进焦点"——按价值流指定阶段（澄清/排期/提测/验收/发起交付）合并
//       业需与独立研发需求，去重后按禅道优先级（P1<P2<P3<P4，前者高）排序取前 5 条。
// 依赖: internal/model
//       internal/module/po/service.go（共享 stageFilters / valueStreamStages）
// =============================================================================

package po

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

// focusStatuses 今日推进焦点覆盖的价值流阶段。
// 注意：released 在 service.go 内部 label 仍为"评价反馈"，但 PO 口径下"发起交付"
// 仅指"验收通过已发起交付"——尚未到反馈评价阶段。该映射由持续讨论决定，本文件
// 当前按 PO 心智直接复用 status 枚举，避免误导前端。
var focusStatuses = []string{
	"clarify",       // 澄清
	"schedule",      // 排期
	"developing",    // 提测
	"waitacceptance", // 验收
	"released",      // 发起交付（验收通过）
}

// FocusLimit 今日推进焦点的最大条数。
const FocusLimit = 5

// FocusReq 今日推进焦点请求参数（预留扩展位，当前为空）。
type FocusReq struct{}

// Validate 校验请求参数（语义为"无入参"，保留结构以遵循规范 §7/§9）。
func (r *FocusReq) Validate() []FieldError {
	return nil
}

// FocusResp 今日推进焦点响应。
type FocusResp struct {
	Items []WorkItemDetail `json:"items"`
}

// priLabelToRank 把展示标签 "P1"/"P2"/"P3"/"P4" 转回数字权重（1=最高）。
// 与 priority.go 的 ParsePriDemand / ParsePriStory 共用同一定义，避免重复硬编码表。
// 该函数在 sort 时使用：pri 已在 WorkItemDetail.Pri 写成 "P1" 形式，但更精确的排序
// 使用 PriRank 字段（detail 入库时已带上权重，避免再次解析）。
func priLabelToRank(label string) int {
	if len(label) < 2 || label[0] != 'P' {
		return PriRankUnknown
	}
	n, err := strconv.Atoi(label[1:])
	if err != nil || n < 1 || n > 4 {
		return PriRankUnknown
	}
	return n
}

// effectiveRank 取 WorkItemDetail 已解析好的权重（PriRank 字段）；
// 若为空（旧数据/老版本兼容），从 Pri 标签反解。
func effectiveRank(it WorkItemDetail) int {
	if it.PriRank > 0 {
		return it.PriRank
	}
	return priLabelToRank(it.Pri)
}

// sortItemsByPriASC 按禅道优先级数字升序排序（数字小者在前）、同级按 kind + id 稳定排序。
func sortItemsByPriASC(items []WorkItemDetail) {
	sort.SliceStable(items, func(i, j int) bool {
		pi, pj := effectiveRank(items[i]), effectiveRank(items[j])
		if pi != pj {
			return pi < pj
		}
		if items[i].ValueStream != items[j].ValueStream {
			return items[i].ValueStream < items[j].ValueStream
		}
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].ID < items[j].ID
	})
}

// Focus 加载今日推进焦点。返回 5 条以内（按优先级）；service.schedule 缺时降级为空。
func (s *Service) Focus(ctx context.Context, actor *model.User, _ FocusReq) (*FocusResp, error) {
	items := make([]WorkItemDetail, 0)
	seen := make(map[string]struct{})

	for _, status := range focusStatuses {
		filter, ok := mysqlStageFilters[status]
		if !ok {
			continue
		}
		label := valueStreamLabelForStatus(status)
		resp, err := s.listMySQLDemands(ctx, actor, status, filter)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.Items {
			key := workItemKey(item.Kind, item.ID)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			if item.ValueStream == "" {
				item.ValueStream = label
			}
			// 真实优先级权重只走 PriRank 字段：Pri 标签可能在某些路径下空或
			// 解析失败，effectiveRank 会从 Pri 反解并落到 PriRankUnknown=99。
			if item.Pri != "" {
				if rank, _ := strconv.Atoi(item.Pri[1:]); rank >= 1 && rank <= 4 {
					item.PriRank = rank
				}
			}
			if item.ZentaoUrl == "" {
				if item.Kind == "demand" {
					item.ZentaoUrl = zentao.URL("demand", "view",
						fmt.Sprintf("demandID=%s", item.ID))
				} else if item.Kind == "story" {
					item.ZentaoUrl = zentao.URL("story", "view",
						fmt.Sprintf("storyID=%s", item.ID))
				}
			}
			items = append(items, item)
		}
	}

	sortItemsByPriASC(items)
	if len(items) > FocusLimit {
		items = items[:FocusLimit]
	}

	return &FocusResp{Items: items}, nil
}
