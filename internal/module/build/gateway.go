// =============================================================================
// 文件: internal/module/build/gateway.go
// 模块: 版本管理
// 类型: action
// 职责: 出站适配层：封装版本关联/解除研发需求的禅道 REST 调用。
// 依赖: internal/pkg/zentao
// =============================================================================

package build

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"workbench/internal/pkg/zentao"
)

// linkBuildStoriesReq 禅道 POST /build/:id/linkstories 或 /unlinkstories。
type linkBuildStoriesReq struct {
	BuildID uint
	Stories string // 逗号分隔需求 ID
}

// linkBuildStories 调用禅道关联研发需求接口。
func linkBuildStories(ctx context.Context, client *zentao.Client, req linkBuildStoriesReq) error {
	return postBuildStories(ctx, client, req, "linkstories")
}

// unlinkBuildStories 调用禅道解除版本与研发需求关联接口。
func unlinkBuildStories(ctx context.Context, client *zentao.Client, req linkBuildStoriesReq) error {
	return postBuildStories(ctx, client, req, "unlinkstories")
}

func postBuildStories(ctx context.Context, client *zentao.Client, req linkBuildStoriesReq, action string) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if req.BuildID == 0 {
		return fmt.Errorf("版本 ID 无效")
	}
	stories := strings.TrimSpace(req.Stories)
	if stories == "" {
		return fmt.Errorf("研发需求为空")
	}
	payload := map[string]any{
		"stories": stories,
	}
	path := fmt.Sprintf("/build/%d/%s", req.BuildID, action)
	return client.Do(ctx, http.MethodPost, path, payload, nil)
}
