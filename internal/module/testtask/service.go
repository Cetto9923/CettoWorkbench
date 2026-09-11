// =============================================================================
// 文件: internal/module/testtask/service.go
// 模块: 提测办理
// 类型: action
// 职责: 提测上下文、产品执行列表与创建版本业务装配。
// 依赖: internal/module/user
//       internal/pkg/errorx
//       internal/pkg/zentao
// =============================================================================

package testtask

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/user"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// Service 提测办理业务逻辑。
type Service struct {
	repo    *Repo
	userSvc *user.Service
	ztAPI   *zentao.Client
	logger  *zap.Logger
}

// NewService 创建 Service。ztAPI 可为空，写入禅道时回退 zentao.DefaultClient()。
func NewService(repo *Repo, userSvc *user.Service, ztAPI *zentao.Client, logger *zap.Logger) *Service {
	return &Service{repo: repo, userSvc: userSvc, ztAPI: ztAPI, logger: logger}
}

// GetContext 获取业需提测上下文。提测办理人为当前登录用户。
func (s *Service) GetContext(ctx context.Context, actor *model.User, demandID uint) (*ContextResp, error) {
	if demandID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	row, err := s.repo.FindDemandContext(ctx, demandID)
	if err != nil {
		if errors.Is(err, errDemandNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
		}
		return nil, err
	}

	displayMap := map[string]string{}
	if s.userSvc != nil {
		m, mapErr := s.userSvc.AccountDisplayMap(ctx, actor)
		if mapErr != nil {
			return nil, mapErr
		}
		if m != nil {
			displayMap = m
		}
	}

	stories, storiesErr := s.repo.FindDemandConvertedStories(ctx, demandID)
	if storiesErr != nil {
		return nil, storiesErr
	}
	products, productsErr := s.repo.FindDemandInvolvedProducts(ctx, demandID)
	if productsErr != nil {
		return nil, productsErr
	}
	systems := BuildSystemItemsWithStories(stories, products, row.MainSystemID, row.MainSystemName)

	users, usersErr := s.repo.ListInsideUsers(ctx)
	if usersErr != nil {
		return nil, usersErr
	}

	account, name := "", ""
	if actor != nil {
		account = actor.Account
		name = actor.DisplayName
	}
	return BuildContextResp(*row, displayMap, account, name, systems, users), nil
}

// ListProductExecutions 当前产品下执行列表（对齐禅道版本创建 stagefilter|leaf|order_asc[+noclosed]）。
func (s *Service) ListProductExecutions(ctx context.Context, actor *model.User, productID uint) ([]ExecutionOption, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if productID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "产品 ID 无效")
	}
	projectIDs, err := s.repo.FindProductProjectIDs(ctx, productID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.FindExecutionsByProjectIDs(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	crExec, err := s.repo.FindCRExecution(ctx)
	if err != nil {
		return nil, err
	}
	noClosed := crExec == 0
	return BuildExecutionOptions(rows, noClosed), nil
}

// ListProductBuilds 返回产品已有版本。
func (s *Service) ListProductBuilds(ctx context.Context, actor *model.User, productID uint) ([]BuildOption, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if productID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "产品 ID 无效")
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.DefaultClient()
	}
	if client == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}
	items, err := listProductBuilds(ctx, client, productID)
	if err != nil {
		return nil, err
	}
	out := make([]BuildOption, 0, len(items))
	for _, item := range items {
		if item.ID == 0 {
			continue
		}
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = fmt.Sprint(item.ID)
		}
		out = append(out, BuildOption{Value: fmt.Sprint(item.ID), Label: label})
	}
	return out, nil
}

// CreateBuilds 将「创建新版本」项同步到禅道 POST /projects/:id/builds；builder 为当前用户 account。
func (s *Service) CreateBuilds(ctx context.Context, actor *model.User, demandID uint, req CreateBuildsReq) (*CreateBuildsResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if demandID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	if s.repo == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "提测上下文不可用")
	}
	if _, err := s.repo.FindDemandContext(ctx, demandID); err != nil {
		if errors.Is(err, errDemandNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
		}
		return nil, err
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.DefaultClient()
	}
	if client == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	out := &CreateBuildsResp{Builds: make([]CreateBuildResult, 0, len(req.Builds))}
	for _, item := range req.Builds {
		build, err := createProjectBuild(ctx, client, createProjectBuildReq{
			ProjectID:   item.ProjectID,
			ExecutionID: item.ExecutionID,
			ProductID:   item.ProductID,
			Name:        strings.TrimSpace(item.Name),
			Builder:     strings.TrimSpace(actor.Account),
			Date:        strings.TrimSpace(item.Date),
			Desc:        item.Desc,
		})
		if err != nil {
			if s.logger != nil {
				s.logger.Error("zentao create build",
					zap.Error(err),
					zap.Uint("productId", item.ProductID),
					zap.Uint("projectId", item.ProjectID),
					zap.String("name", item.Name),
				)
			}
			return nil, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("保存版本失败：%s", err.Error()), err)
		}
		out.Builds = append(out.Builds, CreateBuildResult{
			ProductID: item.ProductID,
			BuildID:   build.ID,
			Name:      build.Name,
		})
	}
	return out, nil
}

// PartialTesttaskError 部分成功错误：携带失败原因与已成功创建的测试单结果。
// Handler 通过 errors.As 识别并以 HTTP 207 Multi-Status 返回。
type PartialTesttaskError struct {
	Cause     string
	Succeeded []CreateTesttaskResult
}

// Error 实现 error 接口。
func (e *PartialTesttaskError) Error() string {
	if e == nil {
		return ""
	}
	return "testtask partial: " + e.Cause
}

// CreateTesttasks 创建非联调测试单（POST 禅道 /projects/:id/testtasks，DoAs(actor.Account, …)）。
//
// 流程：
//  1. 业务校验（joint=1 拒绝、空批拒绝）。
//  2. 需求预检（不存在/已删除 → 拒绝；任一预检失败不会发起远程写入）。
//  3. 同批 (buildID, name) 去重（同一请求内重复条目只取首次，避免重复远程创建）。
//  4. 版本预取与归属校验：build.productID 必须等于 item.productID；
//     build.execution 必须属于 build.project；任一不通过不发起远程写入。
//  5. 逐项 DoAs 创建；每条 10s 超时（与 zentao.Client 自身 15s 客户端超时配合）。
//  6. 第 N 项失败：停止后续创建，把已成功 ID 与失败原因包装为 PartialTesttaskError 返回，
//     业务层可基于 errors.As 取回已建 ID 提示用户「部分成功」，禁止自动重试。
func (s *Service) CreateTesttasks(ctx context.Context, actor *model.User, demandID uint, req CreateTesttasksReq) (*CreateTesttasksResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)
	if demandID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	if req.Joint == 1 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "联调测试单暂不支持")
	}
	if len(req.Tasks) == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "请至少配置一个测试单")
	}
	if s.repo == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "提测上下文不可用")
	}
	if _, err := s.repo.FindDemandContext(ctx, demandID); err != nil {
		if errors.Is(err, errDemandNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
		}
		return nil, err
	}

	// 同批 (buildID, name) 去重：只保留首次出现的条目。
	type dedupItem struct {
		idx     int
		item    CreateTesttaskItem
		key     string
		origIdx []int
	}
	dedup := make([]dedupItem, 0, len(req.Tasks))
	seen := make(map[string]int, len(req.Tasks))
	for i, it := range req.Tasks {
		key := it.IdempotencyKey()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = i
		dedup = append(dedup, dedupItem{idx: i, item: it, key: key})
	}
	if len(dedup) == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "请至少配置一个测试单")
	}

	// 一次性预取全部 build 归属信息（产品/项目/执行）。
	buildIDs := make([]uint, 0, len(dedup))
	for _, d := range dedup {
		if d.item.BuildID > 0 {
			buildIDs = append(buildIDs, d.item.BuildID)
		}
	}
	builds, err := s.repo.FindBuildsByIDs(ctx, buildIDs)
	if err != nil {
		return nil, err
	}

	client := s.ztAPI
	if client == nil {
		client = zentao.DefaultClient()
	}
	if client == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	// 已成功条目容器。失败时通过 PartialTesttaskError 透传给 handler，避免在内存里复制大对象。
	succeeded := make([]CreateTesttaskResult, 0, len(dedup))

	for _, d := range dedup {
		meta, ok := builds[d.item.BuildID]
		if !ok || meta.ID == 0 {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 不存在", d.item.BuildID))
		}
		if d.item.ProductID > 0 && meta.ProductID > 0 && d.item.ProductID != meta.ProductID {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 不属于所选产品", d.item.BuildID))
		}
		if meta.Execution == 0 {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 缺少所属执行", d.item.BuildID))
		}
		// 项目-执行归属校验：执行必须属于该项目（项目=0 时仅校验执行存在）。
		belongs, belErr := s.repo.VerifyExecutionBelongsToProject(ctx, meta.ProjectID, meta.Execution)
		if belErr != nil {
			return nil, belErr
		}
		if !belongs {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 所属执行 %d 不属于项目 %d", d.item.BuildID, meta.Execution, meta.ProjectID))
		}

		productID := d.item.ProductID
		if productID == 0 {
			productID = meta.ProductID
		}

		callCtx, cancel := withTesttaskTimeout(ctx)
		created, createErr := createTesttask(callCtx, client, createTesttaskReq{
			ProjectID:   meta.ProjectID,
			ProductID:   productID,
			ExecutionID: meta.Execution,
			BuildID:     d.item.BuildID,
			Name:        strings.TrimSpace(d.item.Name),
			Begin:       strings.TrimSpace(d.item.Begin),
			End:         strings.TrimSpace(d.item.End),
			Owner:       account,
			Type:        strings.TrimSpace(d.item.Type),
			Pri:         d.item.Pri,
			Desc:        d.item.Desc,
		})
		cancel()
		if createErr != nil {
			if s.logger != nil {
				s.logger.Error("zentao create testtask",
					zap.Error(createErr),
					zap.Uint("productId", productID),
					zap.Uint("buildId", d.item.BuildID),
					zap.String("name", d.item.Name),
				)
			}
			// 区分上下文超时与一般失败：超时显式标记为 unknown，避免前端误判为「确认失败」并自动重发。
			if errors.Is(createErr, context.DeadlineExceeded) {
				return &CreateTesttasksResp{Tasks: succeeded}, &PartialTesttaskError{
					Cause:     fmt.Sprintf("第 %d 项创建超时：结果未知（已成功 %d 条，请勿重复提交）", d.idx+1, len(succeeded)),
					Succeeded: succeeded,
				}
			}
			return &CreateTesttasksResp{Tasks: succeeded}, &PartialTesttaskError{
				Cause:     fmt.Sprintf("第 %d 项创建失败：%s（已成功 %d 条）", d.idx+1, createErr.Error(), len(succeeded)),
				Succeeded: succeeded,
			}
		}
		succeeded = append(succeeded, CreateTesttaskResult{
			ProductID:  productID,
			BuildID:    d.item.BuildID,
			TesttaskID: created.ID,
			Name:       created.Name,
		})
	}
	return &CreateTesttasksResp{Tasks: succeeded}, nil
}
