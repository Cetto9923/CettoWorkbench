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
	"time"

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

type productBuild struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

func listProductBuilds(ctx context.Context, client *zentao.Client, productID uint) ([]productBuild, error) {
	if client == nil || productID == 0 {
		return nil, fmt.Errorf("产品 ID 无效")
	}
	var out struct {
		Builds []productBuild `json:"builds"`
	}
	if err := client.Do(ctx, http.MethodGet, fmt.Sprintf("/products/%d/builds", productID), nil, &out); err != nil {
		return nil, err
	}
	if out.Builds == nil {
		return []productBuild{}, nil
	}
	return out.Builds, nil
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

// =============================================================================
// 创建测试单（POST /projects/:id/testtasks）
// =============================================================================

// createTesttaskReq 创建测试单请求参数。
// 走用户态 DoAs(actor.Account, …)，与 createProjectBuild、linkBuildStories 保持一致。
type createTesttaskReq struct {
	Account     string
	ProjectID   uint
	ProductID   uint
	ExecutionID uint
	BuildID     uint
	Name        string
	Begin       string
	End         string
	Owner       string
	Type        string
	Pri         int
	Desc        string
}

// createdTesttask 禅道测试单创建响应的必要字段。
type createdTesttask struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// testtaskCallTimeout 单条测试单创建的最大允许耗时；超过则取消远程调用并返回超时错误。
// 该值小于 zentao.Client 自身的 http 15s 客户端超时，确保上下文是首要因果。
const testtaskCallTimeout = 10 * time.Second

// withTesttaskTimeout 把 parent ctx 派生为单条测试单写入的子上下文。
func withTesttaskTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, testtaskCallTimeout)
}

// createTesttask 调用禅道创建测试单接口（POST /projects/:id/testtasks）。
// 返回禅道侧的测试单 ID；未拿到 ID 视作不可信结果，函数返回明确错误。
func createTesttask(ctx context.Context, client *zentao.Client, req createTesttaskReq) (*createdTesttask, error) {
	if client == nil {
		return nil, fmt.Errorf("禅道 API 未配置")
	}
	if req.ProjectID == 0 {
		return nil, fmt.Errorf("projectId 无效")
	}
	if req.ProductID == 0 {
		return nil, fmt.Errorf("productId 无效")
	}
	if req.ExecutionID == 0 {
		return nil, fmt.Errorf("executionId 无效")
	}
	if req.BuildID == 0 {
		return nil, fmt.Errorf("buildId 无效")
	}
	payload := map[string]any{
		"project":   req.ProjectID,
		"product":   req.ProductID,
		"execution": req.ExecutionID,
		"build":     req.BuildID,
		"name":      strings.TrimSpace(req.Name),
		"begin":     strings.TrimSpace(req.Begin),
		"end":       strings.TrimSpace(req.End),
		"owner":     strings.TrimSpace(req.Owner),
		"type":      strings.TrimSpace(req.Type),
		"pri":       req.Pri,
		"status":    "wait",
		"joint":     "0",
		"desc":      req.Desc,
	}
	var out createdTesttask
	path := fmt.Sprintf("/projects/%d/testtasks", req.ProjectID)
	if err := client.DoAs(ctx, strings.TrimSpace(req.Account), http.MethodPost, path, payload, &out); err != nil {
		return nil, err
	}
	if out.ID == 0 {
		return nil, fmt.Errorf("禅道未返回测试单 ID")
	}
	return &out, nil
}
