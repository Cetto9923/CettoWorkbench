// =============================================================================
// 文件: internal/module/schedule/gateway.go
// 模块: 排期工作台
// 类型: action
// 职责: 出站适配层：调用禅道检查业需转研发需求前的主管部门审批提醒。
// 依赖: internal/pkg/zentao
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"workbench/internal/pkg/zentao"
)

const reviewToStoryNoticeFallback = "请完成主管部门审批后，再进行转研发需求操作。"

// checkReviewToStoryNotice 调用禅道 GET /demand/{id}/checkReviewToStoryNotice。
// 需拦截时返回提示文案；允许转研发时返回空串。
func checkReviewToStoryNotice(ctx context.Context, client *zentao.Client, demandID uint) (string, error) {
	if client == nil {
		return "", fmt.Errorf("禅道 API 未配置")
	}
	if demandID == 0 {
		return "", fmt.Errorf("业需 ID 无效")
	}

	var out struct {
		Result  string `json:"result"`
		Message string `json:"message"`
	}
	path := fmt.Sprintf("/demand/%d/checkReviewToStoryNotice", demandID)
	err := client.Do(ctx, http.MethodGet, path, nil, &out)
	if err != nil {
		if zentao.StatusCode(err) == http.StatusBadRequest {
			return reviewNoticeMessage(err.Error()), nil
		}
		return "", err
	}
	if strings.EqualFold(strings.TrimSpace(out.Result), "fail") {
		return reviewNoticeMessage(out.Message), nil
	}
	return "", nil
}

func reviewNoticeMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return reviewToStoryNoticeFallback
	}
	return message
}
