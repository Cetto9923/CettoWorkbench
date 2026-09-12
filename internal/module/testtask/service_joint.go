package testtask

import (
	"context"
	"errors"
	"fmt"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// createJointTesttasks 使用当前用户身份创建一张联调总单；负责人是独立的业务字段。
func (s *Service) createJointTesttasks(ctx context.Context, actor *model.User, demandID uint, req CreateTesttasksReq) (*CreateTesttasksResp, error) {
	if s.repo == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "提测上下文不可用")
	}
	if _, err := s.repo.FindDemandContext(ctx, demandID); err != nil {
		if errors.Is(err, errDemandNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
		}
		return nil, err
	}
	ids := []uint{}
	seen := map[uint]bool{}
	for _, list := range req.Builds {
		for _, id := range list {
			if !seen[id] {
				ids = append(ids, id)
				seen[id] = true
			}
		}
	}
	builds, err := s.repo.FindBuildsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	var project uint
	for i, product := range req.Products {
		for _, id := range req.Builds[i] {
			build, ok := builds[id]
			if !ok || build.ProductID != product || build.ProjectID == 0 {
				return nil, errorx.New(errorx.ErrCodeInvalidParam, fmt.Sprintf("版本 %d 不存在或不属于所选产品/项目", id))
			}
			if project != 0 && project != build.ProjectID {
				return nil, errorx.New(errorx.ErrCodeInvalidParam, "联调版本必须属于同一项目")
			}
			project = build.ProjectID
		}
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.DefaultClient()
	}
	if client == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}
	callCtx, cancel := withTesttaskTimeout(ctx)
	defer cancel()
	created, err := createJointTesttask(callCtx, client, createJointTesttaskReq{
		Account: actor.Account, ProjectID: project, Name: req.Name, Begin: req.Begin, End: req.End,
		Owner: req.Owner, Members: req.Members, Products: req.Products, Builds: req.Builds,
		Type: req.Type, Pri: req.Pri, Status: "wait", Desc: req.Desc,
	})
	if err != nil {
		message := "联调测试单创建失败：" + err.Error()
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			message = "联调测试单创建结果未知，请先到禅道核对，勿重复提交"
		}
		return &CreateTesttasksResp{Tasks: []CreateTesttaskResult{}}, &PartialTesttaskError{Cause: message, Succeeded: []CreateTesttaskResult{}}
	}
	return &CreateTesttasksResp{Tasks: []CreateTesttaskResult{{ProductID: req.Products[0], BuildID: req.Builds[0][0], TesttaskID: created.ID, Name: created.Name}}}, nil
}
