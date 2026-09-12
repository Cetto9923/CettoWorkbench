// =============================================================================
// 文件: internal/module/testtask/servicetask.go
// 模块: 提测办理
// 类型: action
// 职责: 非联调逐条创建与联调总单创建的业务装配。
// 依赖: internal/pkg/errorx
//       internal/pkg/zentao
// =============================================================================

package testtask

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// CreateTesttasks 将测试单同步到禅道 POST /projects/:id/testtasks。
// Joint=1 创建一张联调总单；否则按系统逐条创建。所属执行取版本上的 execution。
func (s *Service) CreateTesttasks(ctx context.Context, actor *model.User, demandID uint, req CreateTesttasksReq) (*CreateTesttasksResp, error) {
	_ = demandID
	_ = actor
	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}
	if req.Joint == 1 {
		return s.createJointTesttasks(ctx, client, req)
	}
	return s.createIndependentTesttasks(ctx, client, req)
}

func (s *Service) createIndependentTesttasks(ctx context.Context, client *zentao.Client, req CreateTesttasksReq) (*CreateTesttasksResp, error) {
	buildIDs := make([]uint, 0, len(req.Tasks))
	for _, item := range req.Tasks {
		if item.BuildID > 0 {
			buildIDs = append(buildIDs, item.BuildID)
		}
	}
	builds, err := s.repo.FindBuildsByIDs(ctx, buildIDs)
	if err != nil {
		return nil, err
	}

	out := &CreateTesttasksResp{Tasks: make([]CreateTesttaskResult, 0, len(req.Tasks))}
	for _, item := range req.Tasks {
		meta, ok := builds[item.BuildID]
		if !ok || meta.ID == 0 {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 不存在", item.BuildID))
		}
		if item.ProductID > 0 && meta.ProductID > 0 && item.ProductID != meta.ProductID {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 不属于所选产品", item.BuildID))
		}
		productID := item.ProductID
		if productID == 0 {
			productID = meta.ProductID
		}

		if meta.Execution == 0 {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 缺少所属执行", item.BuildID))
		}

		created, createErr := createTesttask(ctx, client, createTesttaskReq{
			ProjectID:   meta.ProjectID,
			ProductID:   productID,
			ExecutionID: meta.Execution,
			BuildID:     item.BuildID,
			Name:        strings.TrimSpace(item.Name),
			Begin:       strings.TrimSpace(item.Begin),
			End:         strings.TrimSpace(item.End),
			Owner:       strings.TrimSpace(item.Owner),
			Type:        strings.TrimSpace(item.Type),
			Pri:         item.Pri,
			Status:      "wait",
			Desc:        item.Desc,
			Joint:       "0",
		})
		if createErr != nil {
			if s.logger != nil {
				s.logger.Error("zentao create testtask",
					zap.Error(createErr),
					zap.Uint("productId", productID),
					zap.Uint("buildId", item.BuildID),
					zap.String("name", item.Name),
				)
			}
			return nil, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("保存测试单失败：%s", createErr.Error()), createErr)
		}
		out.Tasks = append(out.Tasks, CreateTesttaskResult{
			ProductID:  productID,
			BuildID:    item.BuildID,
			TesttaskID: created.ID,
			Name:       created.Name,
		})
	}
	return out, nil
}

func (s *Service) createJointTesttasks(ctx context.Context, client *zentao.Client, req CreateTesttasksReq) (*CreateTesttasksResp, error) {
	buildIDs := make([]uint, 0)
	for _, ids := range req.Builds {
		for _, id := range ids {
			if id > 0 {
				buildIDs = append(buildIDs, id)
			}
		}
	}
	builds, err := s.repo.FindBuildsByIDs(ctx, buildIDs)
	if err != nil {
		return nil, err
	}

	var projectID uint
	var firstBuildID uint
	for i, productID := range req.Products {
		if i >= len(req.Builds) {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, "各系统版本不能为空")
		}
		for _, buildID := range req.Builds[i] {
			meta, ok := builds[buildID]
			if !ok || meta.ID == 0 {
				return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 不存在", buildID))
			}
			if productID > 0 && meta.ProductID > 0 && productID != meta.ProductID {
				return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 不属于所选产品", buildID))
			}
			if firstBuildID == 0 {
				firstBuildID = buildID
			}
			if projectID == 0 && meta.ProjectID > 0 {
				projectID = meta.ProjectID
			}
		}
	}
	if projectID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "无法确定所属项目")
	}

	created, createErr := createJointTesttask(ctx, client, createJointTesttaskReq{
		ProjectID: projectID,
		Name:      strings.TrimSpace(req.Name),
		Begin:     strings.TrimSpace(req.Begin),
		End:       strings.TrimSpace(req.End),
		Owner:     strings.TrimSpace(req.Owner),
		Members:   req.Members,
		Products:  req.Products,
		Builds:    req.Builds,
		Type:      strings.TrimSpace(req.Type),
		Pri:       req.Pri,
		Status:    "wait",
		Desc:      req.Desc,
	})
	if createErr != nil {
		if s.logger != nil {
			s.logger.Error("zentao create joint testtask",
				zap.Error(createErr),
				zap.Uint("projectId", projectID),
				zap.String("name", req.Name),
			)
		}
		return nil, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("保存测试单失败：%s", createErr.Error()), createErr)
	}

	productID := uint(0)
	if len(req.Products) > 0 {
		productID = req.Products[0]
	}
	return &CreateTesttasksResp{Tasks: []CreateTesttaskResult{{
		ProductID:  productID,
		BuildID:    firstBuildID,
		TesttaskID: created.ID,
		Name:       created.Name,
	}}}, nil
}
