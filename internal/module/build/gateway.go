// =============================================================================
// 文件: internal/module/build/gateway.go
// 模块: 版本管理
// 类型: action
// 职责: 出站适配层：封装版本关联研发需求的禅道 REST 调用。
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

// linkBuildStoriesReq 禅道 POST /build/:id/linkstories。
type linkBuildStoriesReq struct {
	BuildID uint
	Stories string // 逗号分隔需求 ID
}

// linkBuildStories 调用禅道关联研发需求接口。
func linkBuildStories(ctx context.Context, client *zentao.Client, req linkBuildStoriesReq) error {
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
	path := fmt.Sprintf("/build/%d/linkstories", req.BuildID)
	return client.Do(ctx, http.MethodPost, path, payload, nil)
}
