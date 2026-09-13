// =============================================================================
// 文件: internal/module/build/gateway.go
// 模块: 版本管理
// 类型: action
// 职责: 出站适配层：封装版本关联/解除研发需求的禅道原生控制层调用。
// 依赖: internal/pkg/zentao
// =============================================================================

package build

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"workbench/internal/pkg/zentao"
)

// linkBuildStoriesReq 版本关联/解除研发需求入参。
type linkBuildStoriesReq struct {
	BuildID uint
	Account string // 当前登录账号；以该账号的用户态 Token 调用
	Stories string // 逗号分隔需求 ID
}

// 禅道 build 模块的原生控制层动作（站点 PATH_INFO），两边读取的 $_POST 字段名不同。
const (
	buildActionLinkStories   = "linkStory"        // build::linkStory，读 $_POST['stories']
	buildActionUnlinkStories = "batchUnlinkStory" // build::batchUnlinkStory，读 $_POST['storyIdList']
)

// linkBuildStories 调用禅道原生控制层动作 build::linkStory 关联研发需求。
func linkBuildStories(ctx context.Context, client *zentao.Client, req linkBuildStoriesReq) error {
	return postBuildStories(ctx, client, req, buildActionLinkStories)
}

// unlinkBuildStories 调用禅道原生控制层动作 build::batchUnlinkStory 解除关联。
func unlinkBuildStories(ctx context.Context, client *zentao.Client, req linkBuildStoriesReq) error {
	return postBuildStories(ctx, client, req, buildActionUnlinkStories)
}

func postBuildStories(ctx context.Context, client *zentao.Client, req linkBuildStoriesReq, action string) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if req.BuildID == 0 {
		return fmt.Errorf("版本 ID 无效")
	}
	account := strings.TrimSpace(req.Account)
	if account == "" {
		return fmt.Errorf("操作人账号为空")
	}
	ids := splitStoryIDCSV(req.Stories)
	if len(ids) == 0 {
		return fmt.Errorf("研发需求为空")
	}

	// 字段名以禅道控制层实际读取的 $_POST 键为准；数组形式提交，否则控制层 foreach 会拿到标量。
	field := "stories[]"
	if action == buildActionUnlinkStories {
		field = "storyIdList[]"
	}
	form := url.Values{}
	for _, id := range ids {
		form.Add(field, id)
	}

	path := fmt.Sprintf("/build-%s-%d.html", action, req.BuildID)
	return client.DoSiteForm(ctx, account, http.MethodPost, path, form)
}

// splitStoryIDCSV 拆分逗号分隔的需求 ID，去空、去重且保序。
func splitStoryIDCSV(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}
