// =============================================================================
// 文件: internal/module/build/service.go
// 模块: 版本管理
// 类型: action
// 职责: 关联研发需求默认列表、bySearch 与写入禅道关联。
// 依赖: internal/module/user
//       internal/pkg/errorx
//       internal/pkg/pagination
//       internal/pkg/zentao
// =============================================================================

package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/user"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/zentao"
)

// Service 版本业务逻辑。
type Service struct {
	repo    *Repo
	userSvc *user.Service
	ztAPI   *zentao.Client
	logger  *zap.Logger
}

// NewService 创建 Service。ztAPI 可为空，写入禅道时回退 zentao.API()。
func NewService(repo *Repo, userSvc *user.Service, ztAPI *zentao.Client, logger *zap.Logger) *Service {
	return &Service{repo: repo, userSvc: userSvc, ztAPI: ztAPI, logger: logger}
}

// LinkStory 对齐禅道 projectbuild-linkStory（默认列表或 bySearch）。
func (s *Service) LinkStory(ctx context.Context, actor *model.User, buildID uint, req LinkStoryListReq) (*LinkStoryListResp, *pagination.Pager, error) {
	req.Normalize()
	if buildID == 0 {
		return nil, nil, errorx.New(errorx.ErrCodeInvalidParam, "版本 ID 无效")
	}

	build, err := s.repo.FindBuildByID(ctx, buildID)
	if err != nil {
		if errors.Is(err, errBuildNotFound) {
			return nil, nil, errorx.New(errorx.ErrCodeNotFound, "版本不存在")
		}
		return nil, nil, err
	}

	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	req.Value1 = ReplaceMeToken(req.Value1, account)
	req.Value2 = ReplaceMeToken(req.Value2, account)

	productType, err := s.repo.FindProductType(ctx, build.Product)
	if err != nil {
		return nil, nil, err
	}
	includeBranch := productType != "normal"
	defs := StorySearchFieldDefs(includeBranch)
	searchForm, err := s.buildSearchForm(ctx, actor, build.Product, includeBranch, defs, req)
	if err != nil {
		return nil, nil, err
	}

	executionID := ResolveExecutionID(build.Execution, build.Project)
	baseURL := fmt.Sprintf("/builds/%d/linkstory", buildID)
	querySuffix := QuerySuffixFromReq(req)

	if !req.IsBySearch() && executionID == 0 {
		pager := pagination.New(0, req.Page, req.PageSize)
		return &LinkStoryListResp{
			BuildID:     buildID,
			BaseUrl:     baseURL,
			Stories:     []LinkStoryItem{},
			Total:       0,
			Page:        pager.CurrentPage,
			PageSize:    pager.PageSize,
			SearchForm:  searchForm,
			QuerySuffix: querySuffix,
		}, pager, nil
	}

	childIDs := ParseCSVUintIDs(build.Builds)
	childStories, err := s.repo.FindChildBuildStories(ctx, childIDs)
	if err != nil {
		return nil, nil, err
	}
	linkedIDs := MergeAllStoriesCSV(build.Stories, childStories)

	parentIDs, err := s.repo.FindParentStoryIDs(ctx, build.Product)
	if err != nil {
		return nil, nil, err
	}
	excludeIDs := BuildExcludeIDList(parentIDs, linkedIDs)

	pager := pagination.New(0, req.Page, req.PageSize)
	var rows []linkableStoryRow
	var total int64

	if req.IsBySearch() {
		userWhere, userArgs, whereErr := s.repo.BuildUserSearchWhere(ctx, req, defs)
		if whereErr != nil {
			return nil, nil, whereErr
		}
		rows, total, err = s.repo.FindStoriesBySearch(ctx, RepoFindStoriesBySearchReq{
			ProductID:  build.Product,
			BranchCSV:  build.Branch,
			ExcludeIDs: excludeIDs,
			UserWhere:  userWhere,
			UserArgs:   userArgs,
			Limit:      pager.Limit(),
			Offset:     pager.Offset(),
		})
	} else {
		rows, total, err = s.repo.FindLinkableStories(ctx, RepoFindLinkableStoriesReq{
			ExecutionID: executionID,
			ProductID:   build.Product,
			BranchCSV:   build.Branch,
			ExcludeIDs:  excludeIDs,
			Limit:       pager.Limit(),
			Offset:      pager.Offset(),
		})
	}
	if err != nil {
		return nil, nil, err
	}
	pager = pagination.New(total, req.Page, req.PageSize)

	displayMap := map[string]string{}
	if s.userSvc != nil {
		m, mapErr := s.userSvc.AccountDisplayMap(ctx, actor)
		if mapErr != nil {
			return nil, nil, mapErr
		}
		if m != nil {
			displayMap = m
		}
	}

	items := make([]LinkStoryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, LinkStoryItem{
			ID:             row.ID,
			Pri:            row.Pri,
			PriClass:       PriorityClass(row.Pri),
			Title:          row.Title,
			ZentaoUrl:      zentao.StoryViewURL(row.ID),
			OpenedBy:       lookupDisplay(displayMap, row.OpenedBy),
			AssignedTo:     lookupDisplay(displayMap, row.AssignedTo),
			EstimateText:   FormatEstimate(row.Estimate),
			StatusLabel:    StoryStatusLabel(row.Status),
			StageLabel:     StoryStageLabel(row.Stage),
			Stage:          row.Stage,
			DefaultChecked: ShouldDefaultCheckStage(row.Stage),
		})
	}

	return &LinkStoryListResp{
		BuildID:     buildID,
		BaseUrl:     baseURL,
		Stories:     items,
		Total:       total,
		Page:        pager.CurrentPage,
		PageSize:    pager.PageSize,
		SearchForm:  searchForm,
		QuerySuffix: querySuffix,
	}, pager, nil
}

// LinkStories 将勾选的研发需求关联到版本（POST 禅道 /build/:id/linkstories）。
func (s *Service) LinkStories(ctx context.Context, actor *model.User, buildID uint, req LinkStoriesReq) error {
	_ = actor // 预留：对象级权限 / 操作人审计
	if buildID == 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "版本 ID 无效")
	}
	stories := req.NormalizedStories()
	if stories == "" {
		return errorx.New(errorx.ErrCodeInvalidParam, "请选择要关联的研发需求")
	}

	_, err := s.repo.FindBuildByID(ctx, buildID)
	if err != nil {
		if errors.Is(err, errBuildNotFound) {
			return errorx.New(errorx.ErrCodeNotFound, "版本不存在")
		}
		return err
	}

	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	if err := linkBuildStories(ctx, client, linkBuildStoriesReq{
		BuildID: buildID,
		Stories: stories,
	}); err != nil {
		if s.logger != nil {
			s.logger.Error("zentao link stories",
				zap.Error(err),
				zap.Uint("buildId", buildID),
				zap.String("stories", stories),
			)
		}
		return errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("关联需求失败：%s", err.Error()), err)
	}
	return nil
}

// UnlinkStories 将研发需求从版本解除关联（POST 禅道 /build/:id/unlinkstories）。
func (s *Service) UnlinkStories(ctx context.Context, actor *model.User, buildID uint, req LinkStoriesReq) error {
	_ = actor // 预留：对象级权限 / 操作人审计
	if buildID == 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "版本 ID 无效")
	}
	stories := req.NormalizedStories()
	if stories == "" {
		return errorx.New(errorx.ErrCodeInvalidParam, "请选择要解除关联的研发需求")
	}

	_, err := s.repo.FindBuildByID(ctx, buildID)
	if err != nil {
		if errors.Is(err, errBuildNotFound) {
			return errorx.New(errorx.ErrCodeNotFound, "版本不存在")
		}
		return err
	}

	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	if err := unlinkBuildStories(ctx, client, linkBuildStoriesReq{
		BuildID: buildID,
		Stories: stories,
	}); err != nil {
		if s.logger != nil {
			s.logger.Error("zentao unlink stories",
				zap.Error(err),
				zap.Uint("buildId", buildID),
				zap.String("stories", stories),
			)
		}
		return errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("解除关联失败：%s", err.Error()), err)
	}
	return nil
}

func (s *Service) buildSearchForm(ctx context.Context, actor *model.User, productID uint, includeBranch bool, defs []SearchFieldDef, req LinkStoryListReq) (LinkStorySearchForm, error) {
	opts := StaticSearchOptions()
	modules, err := s.repo.FindModuleOptions(ctx, productID)
	if err != nil {
		return LinkStorySearchForm{}, err
	}
	plans, err := s.repo.FindPlanOptions(ctx, productID)
	if err != nil {
		return LinkStorySearchForm{}, err
	}
	opts["modules"] = modules
	opts["plans"] = plans
	if includeBranch {
		opts["branches"] = []SearchOption{{Value: "", Label: "所有"}, {Value: "0", Label: "主干"}}
	}

	return LinkStorySearchForm{
		BrowseType: req.BrowseType,
		Field1:     req.Field1,
		Operator1:  req.Operator1,
		Value1:     req.Value1,
		AndOr:      req.AndOr,
		Field2:     req.Field2,
		Operator2:  req.Operator2,
		Value2:     req.Value2,
		Fields:     defs,
		Options:    opts,
	}, nil
}

// SearchMetaJSON 供前端动态控件使用。
func SearchMetaJSON(form LinkStorySearchForm) string {
	type meta struct {
		Fields  []SearchFieldDef           `json:"fields"`
		Options map[string][]SearchOption  `json:"options"`
	}
	b, err := json.Marshal(meta{Fields: form.Fields, Options: form.Options})
	if err != nil {
		return "{}"
	}
	return string(b)
}

func lookupDisplay(m map[string]string, account string) string {
	account = strings.TrimSpace(account)
	if account == "" {
		return ""
	}
	if label, ok := m[account]; ok && label != "" {
		return label
	}
	return account
}
