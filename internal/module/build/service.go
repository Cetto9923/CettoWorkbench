// =============================================================================
// 文件: internal/module/build/service.go
// 模块: 版本管理
// 类型: action
// 职责: 关联研发需求默认列表业务装配。
// 依赖: internal/module/user
//       internal/pkg/errorx
//       internal/pkg/pagination
// =============================================================================

package build

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/user"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/pagination"
)

// Service 版本业务逻辑。
type Service struct {
	repo    *Repo
	userSvc *user.Service
	logger  *zap.Logger
}

// NewService 创建 Service。
func NewService(repo *Repo, userSvc *user.Service, logger *zap.Logger) *Service {
	return &Service{repo: repo, userSvc: userSvc, logger: logger}
}

// LinkStory 对齐禅道 projectbuild-linkStory 默认列表（非 bySearch）。
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

	executionID := ResolveExecutionID(build.Execution, build.Project)
	if executionID == 0 {
		pager := pagination.New(0, req.Page, req.PageSize)
		return &LinkStoryListResp{
			BuildID:  buildID,
			BaseUrl:  fmt.Sprintf("/builds/%d/linkstory", buildID),
			Stories:  []LinkStoryItem{},
			Total:    0,
			Page:     pager.CurrentPage,
			PageSize: pager.PageSize,
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
	rows, total, err := s.repo.FindLinkableStories(ctx, RepoFindLinkableStoriesReq{
		ExecutionID: executionID,
		ProductID:   build.Product,
		BranchCSV:   build.Branch,
		ExcludeIDs:  excludeIDs,
		Limit:       pager.Limit(),
		Offset:      pager.Offset(),
	})
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
		BuildID:  buildID,
		BaseUrl:  fmt.Sprintf("/builds/%d/linkstory", buildID),
		Stories:  items,
		Total:    total,
		Page:     pager.CurrentPage,
		PageSize: pager.PageSize,
	}, pager, nil
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
