// =============================================================================
// 文件: internal/module/po/service_todo.go
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
// 审批决策 Tab 占位（真实审批流后续接入），需求治理 / 全部 Tab 走 actor scope 聚合。
func (s *Service) TodoList(ctx context.Context, actor *model.User, req TodoListReq) (*TodoListResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &TodoListResp{Items: []TodoItem{}, Page: req.Page, PageSize: req.PageSize}, nil
	}
	// 审批决策 Tab：本期占位（真实审批流后续接入）
	if req.Tab == TodoTabApproval {
		return &TodoListResp{Items: []TodoItem{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
	}
	// 需求治理 / 全部 Tab：actor scope ∪ 7 维 AND 公式
	items, total, err := s.repo.FindTodoItems(ctx, actor.Account, req)
	if err != nil {
		return nil, err
	}
	page := req.Page
	pageSize := req.PageSize
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}
	return &TodoListResp{
		Items:    items[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

