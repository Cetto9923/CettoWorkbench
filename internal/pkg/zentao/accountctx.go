// =============================================================================
// 文件: internal/pkg/zentao/accountctx.go
// 模块: 基础设施
// 类型: infra
// 职责: 在 context 中携带当前登录账号，供禅道 Client.Do 免密取 Token。
// 依赖: 无
// =============================================================================

package zentao

import (
	"context"
	"strings"
)

type accountCtxKey struct{}

// WithAccount 将当前登录账号写入 ctx（由 RequireLogin 注入）。
func WithAccount(ctx context.Context, account string) context.Context {
	account = strings.TrimSpace(account)
	if ctx == nil {
		ctx = context.Background()
	}
	if account == "" {
		return ctx
	}
	return context.WithValue(ctx, accountCtxKey{}, account)
}

// AccountFrom 读取 ctx 中的当前账号；未注入时返回空串。
func AccountFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(accountCtxKey{}).(string)
	return strings.TrimSpace(v)
}
