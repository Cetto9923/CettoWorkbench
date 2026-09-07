// =============================================================================
// 文件: internal/module/po/servicedone.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的已办服务（六步查询、指标卡统计、Facet 芯片聚合与动作详情）。
//       F01：动作详情 DoneDetail 在入口执行对象级授权：
//       actor 必须是动作 actor 本身，或与动作对象存在关系（PO/提出/责任/
//       测试/验收/评审/创建/闭环）。其他无关账号读取视为 403。
// =============================================================================

package po

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// DoneList 我的已办列表服务。
func (s *Service) DoneList(ctx context.Context, actor *model.User, req DoneListReq) (*DoneListResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &DoneListResp{Items: []DoneAction{}, Page: req.Page, PageSize: req.PageSize}, nil
	}

	items, total, err := s.repo.FindDoneActions(ctx, RepoFindDoneActionsReq{
		Account:    actor.Account,
		Tab:        req.Tab,
		TimeRange:  req.TimeRange,
		CustomFrom: req.CustomFrom,
		CustomTo:   req.CustomTo,
		ObjectType: req.ObjectType,
		Result:     req.Result,
		Action:     req.Action,
		Keyword:    req.Keyword,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	objectType := req.ObjectType
	if objectType == "all" {
		objectType = ""
	}
	summary, err := s.repo.CountDoneActions(ctx, RepoCountDoneActionsReq{
		Account:    actor.Account,
		ObjectType: objectType,
	})
	if err != nil {
		return nil, err
	}

	facets := s.repo.CountDoneFacetCounts(ctx, actor.Account)

	return &DoneListResp{
		Items:    items,
		Total:    total,
		Summary:  summary,
		Facets:   facets,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DoneMeta 获取已办筛选项元数据。
func (s *Service) DoneMeta(ctx context.Context, actor *model.User) (*DoneMetaResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}

	timeRanges := []DoneMetaOption{
		{Key: "all", Label: "全部时间"},
		{Key: "today", Label: "今天"},
		{Key: "7d", Label: "近 7 天"},
		{Key: "week", Label: "本周"},
		{Key: "30d", Label: "近 30 天"},
		{Key: "month", Label: "本月"},
		{Key: "lastMonth", Label: "上月"},
		{Key: "quarter", Label: "本季度"},
		{Key: "custom", Label: "自定义"},
	}

	objectTypes := []DoneMetaOption{
		{Key: "demand", Label: "业务需求"},
		{Key: "story", Label: "研发需求"},
		{Key: "task", Label: "任务"},
		{Key: "bug", Label: "Bug"},
		{Key: "risk", Label: "风险"},
		{Key: "issue", Label: "问题"},
		{Key: "feedback", Label: "反馈"},
		{Key: "release", Label: "发布"},
		{Key: "build", Label: "构建"},
		{Key: "todo", Label: "待办"},
	}

	results := []DoneMetaOption{
		{Key: "done", Label: "完成"},
		{Key: "approved", Label: "通过"},
		{Key: "rejected", Label: "驳回"},
		{Key: "closed", Label: "关闭"},
		{Key: "activated", Label: "激活"},
		{Key: "submitted", Label: "已提交"},
		{Key: "verified", Label: "已验收"},
		{Key: "resolved", Label: "已解决"},
		{Key: "returned", Label: "已退回"},
	}

	actionTypes := []DoneMetaAction{
		{Key: "demand:reviewed", Label: "需求评审", ObjectType: "demand"},
		{Key: "demand:reviewpassed", Label: "审批通过", ObjectType: "demand"},
		{Key: "demand:reviewrejected", Label: "审批驳回", ObjectType: "demand"},
		{Key: "demand:plan", Label: "完成排期", ObjectType: "demand"},
		{Key: "demand:deliver", Label: "发起交付", ObjectType: "demand"},
		{Key: "story:submitreview", Label: "提交评审", ObjectType: "story"},
		{Key: "story:reviewed", Label: "评审研发需求", ObjectType: "story"},
		{Key: "task:started", Label: "开始任务", ObjectType: "task"},
		{Key: "task:finished", Label: "完成任务", ObjectType: "task"},
		{Key: "task:closed", Label: "关闭任务", ObjectType: "task"},
		{Key: "bug:resolved", Label: "解决 Bug", ObjectType: "bug"},
		{Key: "bug:closed", Label: "关闭 Bug", ObjectType: "bug"},
		{Key: "todo:finished", Label: "完成待办", ObjectType: "todo"},
	}

	projects := s.repo.FindDoneProjects(ctx, account)

	return &DoneMetaResp{
		Products:        []DoneMetaOption{},
		PageSizeOptions: []int{20, 50, 100},
		ObjectTypes:     objectTypes,
		ActionTypes:     actionTypes,
		Results:         results,
		TimeRanges:      timeRanges,
		Projects:        projects,
		Executions:      []DoneMetaOption{},
	}, nil
}

// DoneDetail 获取单个已办动作详情。
//
// F01：动作详情是对象级资源，必须由 Service 强制授权：
//   - 无 actor → 401（路由已拦截认证，此处再校验防御）
//   - 动作记录本身不存在 → 404
//   - actor 既不是该动作的 actor，也不在动作对象的任一 PO/责任/提出/
//     测试/验收/评审/创建/闭环关系里 → 403
//   - 命中授权 → Repo 才允许扩展其他用户的对象上下文与邻近时间线
func (s *Service) DoneDetail(ctx context.Context, actor *model.User, actionId int64) (*DoneDetailResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "缺少用户身份")
	}
	if actionId <= 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "动作 ID 无效")
	}

	actorAccount, err := s.repo.FindDoneActionActor(ctx, actionId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "动作不存在")
		}
		return nil, err
	}

	if !actor.IsSuperAdmin && actorAccount != actor.Account {
		ok, err := s.repo.CheckDoneActionObjectVisibility(ctx, actionId, actor.Account)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorx.New(errorx.ErrCodeForbidden, "无权查看该已办动作")
		}
	}

	return s.repo.FindDoneActionDetail(ctx, actionId)
}
