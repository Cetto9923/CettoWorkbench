// =============================================================================
// 文件: internal/module/testtask/gateway.go
// 模块: 提测办理
// 类型: action
// 职责: 出站适配层（对称于 Repo）：封装创建版本（POST /projects/:id/builds）禅道 REST 调用。
//       写操作走 DoAs(actor.Account, …)，与 LinkStory 一致。
// 依赖: internal/pkg/zentao
// =============================================================================

package testtask

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"workbench/internal/pkg/zentao"
)

// createProjectBuildReq 创建项目版本（POST /projects/:id/builds）。
type createProjectBuildReq struct {
	ProjectID   uint
	ExecutionID uint
	ProductID   uint
	Name        string
	Builder     string // 当前登录账号；走 DoAs
	Date        string
	Desc        string
}

// projectBuild 禅道版本创建响应的必要字段。
type projectBuild struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// createProjectBuild 调用禅道创建版本接口（用户态 DoAs）。
func createProjectBuild(ctx context.Context, client *zentao.Client, req createProjectBuildReq) (*projectBuild, error) {
	if client == nil {
		return nil, fmt.Errorf("禅道 API 未配置")
	}
	if req.ProjectID == 0 {
		return nil, fmt.Errorf("projectId 无效")
	}
	account := strings.TrimSpace(req.Builder)
	if account == "" {
		return nil, fmt.Errorf("操作人账号为空")
	}
	payload := map[string]any{
		"execution": req.ExecutionID,
		"product":   req.ProductID,
		"name":      strings.TrimSpace(req.Name),
		"builder":   account,
		"date":      strings.TrimSpace(req.Date),
		"desc":      req.Desc,
	}
	var out projectBuild
	path := fmt.Sprintf("/projects/%d/builds", req.ProjectID)
	if err := client.DoAs(ctx, account, http.MethodPost, path, payload, &out); err != nil {
		return nil, err
	}
	if out.ID == 0 {
		return nil, fmt.Errorf("禅道未返回版本 ID")
	}
	return &out, nil
}
