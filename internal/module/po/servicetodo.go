// =============================================================================
// 文件: internal/module/po/servicetodo.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办服务（V10.1 02 节 7 维 AND）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
)

// TodoList 我的待办列表服务。
// V10.1 02 节：7 维 AND 公式。本期完整实现：Tab + 办理场景 + 阶段 + 对象 + 我的关系 + 办理责任 + 关键词。
// 仅已接入统一查询的数据域可通过页面进入，需求治理 / 全部 Tab 走 actor scope 聚合。
func (s *Service) TodoList(ctx context.Context, actor *model.User, req TodoListReq) (*TodoListResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &TodoListResp{Items: []TodoItem{}, Page: req.Page, PageSize: req.PageSize, Facets: buildTodoFacets(nil)}, nil
	}
	pagedResult, err := s.repo.QueryTodoUnified(ctx, actor.Account, req)
	if err != nil {
		return nil, err
	}
	return &TodoListResp{
		Items:    pagedResult.Items,
		Total:    pagedResult.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Summary:  pagedResult.Summary,
		Groups:   pagedResult.Groups,
		Facets:   pagedResult.Facets,
	}, nil
}
