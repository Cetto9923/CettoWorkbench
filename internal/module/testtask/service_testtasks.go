package testtask

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"strings"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

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
//  1. 业务校验与独立/联调分流。
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
	if errs := req.Validate(); len(errs) > 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, errs[0].Message)
	}
	if req.Joint == 1 {
		return s.createJointTesttasks(ctx, actor, demandID, req)
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

	}

	for _, d := range dedup {
		meta := builds[d.item.BuildID]
		productID := meta.ProductID
		callCtx, cancel := withTesttaskTimeout(ctx)
		created, createErr := createTesttask(callCtx, client, createTesttaskReq{
			ProjectID:   meta.ProjectID,
			ProductID:   productID,
			ExecutionID: meta.Execution,
			BuildID:     d.item.BuildID,
			Name:        strings.TrimSpace(d.item.Name),
			Begin:       strings.TrimSpace(d.item.Begin),
			End:         strings.TrimSpace(d.item.End),
			Account:     account,
			Owner:       strings.TrimSpace(d.item.Owner),
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
