// =============================================================================
// 文件: internal/module/po/gateway.go
// 模块: PO 工作台
// 类型: action
// 职责: 出站适配层：封装业需评审的禅道 REST 调用（POST /demand/:id/review）。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"workbench/internal/pkg/zentao"
)

// reviewDemandViaZentaoReq 禅道 POST /demand/:id/review 入参。
type reviewDemandViaZentaoReq struct {
	DemandID int64
	Result   string // pass / refuse
	Comment  string
}

// reviewDemandViaZentao 以当前登录账号（ctx）调用禅道评审接口。
func reviewDemandViaZentao(ctx context.Context, client *zentao.Client, req reviewDemandViaZentaoReq) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if req.DemandID <= 0 {
		return fmt.Errorf("需求 ID 无效")
	}
	result := strings.TrimSpace(req.Result)
	if result != "pass" && result != "refuse" {
		return fmt.Errorf("评审结果无效")
	}
	payload := map[string]any{
		"result": result,
	}
	if comment := strings.TrimSpace(req.Comment); comment != "" {
		payload["comment"] = comment
	}
	path := fmt.Sprintf("/demand/%d/review", req.DemandID)
	return client.Do(ctx, http.MethodPost, path, payload, nil)
}
