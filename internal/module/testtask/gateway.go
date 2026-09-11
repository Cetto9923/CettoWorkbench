// =============================================================================
// 文件: internal/module/testtask/gateway.go
// 模块: 提测办理
// 类型: action
// 职责: 出站适配层（对称于 Repo）：封装提测相关禅道 REST 调用。
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
	Builder     string
	Date        string
	Desc        string
}

// projectBuild 禅道版本创建响应的必要字段。
type projectBuild struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// productBuild 禅道产品下已有版本项。
type productBuild struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// createProjectBuild 调用禅道创建版本接口。
func createProjectBuild(ctx context.Context, client *zentao.Client, req createProjectBuildReq) (*projectBuild, error) {
	if client == nil {
		return nil, fmt.Errorf("禅道 API 未配置")
	}
	if req.ProjectID == 0 {
		return nil, fmt.Errorf("projectId 无效")
	}

	payload := map[string]any{
		"execution": req.ExecutionID,
		"product":   req.ProductID,
		"name":      strings.TrimSpace(req.Name),
		"builder":   strings.TrimSpace(req.Builder),
		"date":      strings.TrimSpace(req.Date),
		"desc":      req.Desc,
	}
	var out projectBuild
	path := fmt.Sprintf("/projects/%d/builds", req.ProjectID)
	if err := client.Do(ctx, http.MethodPost, path, payload, &out); err != nil {
		return nil, err
	}
	if out.ID == 0 {
		return nil, fmt.Errorf("禅道未返回版本 ID")
	}
	return &out, nil
}

// listProductBuilds 调用禅道 GET /products/:id/builds 拉取产品已有版本。
func listProductBuilds(ctx context.Context, client *zentao.Client, productID uint) ([]productBuild, error) {
	if client == nil {
		return nil, fmt.Errorf("禅道 API 未配置")
	}
	if productID == 0 {
		return nil, fmt.Errorf("productId 无效")
	}
	var resp struct {
		Builds []productBuild `json:"builds"`
	}
	path := fmt.Sprintf("/products/%d/builds", productID)
	if err := client.Do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Builds == nil {
		return []productBuild{}, nil
	}
	return resp.Builds, nil
}
