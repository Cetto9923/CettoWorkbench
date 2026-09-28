package zentao

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ZenTaoSessionCookieName 是禅道默认会话 Cookie 名；不能转发工作台 Cookie。
const ZenTaoSessionCookieName = "zentaosid"

// ErrIssueActionUnavailable 表示原生动作接口或用户会话不可用，调用方必须失败关闭。
var ErrIssueActionUnavailable = errors.New("zentao issue action unavailable")

type IssueAction string

const (
	IssueActionResolve  IssueAction = "resolve"
	IssueActionClose    IssueAction = "close"
	IssueActionActivate IssueAction = "activate"
)

// IssueActionRequest 是传给禅道客户端的最小上下文。SessionID 仅来自 zentaosid Cookie。
type IssueActionRequest struct {
	IssueID   int64
	Action    IssueAction
	SessionID string
}

// IssueActionGateway 隔离工作台与禅道问题原生动作接口。
type IssueActionGateway interface {
	ExecuteIssueAction(context.Context, IssueActionRequest) error
}

// UnavailableIssueActionGateway 在原生接口未经配置或验证时显式失败，绝不伪造成功。
type UnavailableIssueActionGateway struct{ reason string }

func NewUnavailableIssueActionGateway(reason string) *UnavailableIssueActionGateway {
	return &UnavailableIssueActionGateway{reason: strings.TrimSpace(reason)}
}

func (g *UnavailableIssueActionGateway) ExecuteIssueAction(_ context.Context, req IssueActionRequest) error {
	if req.IssueID <= 0 || !validIssueAction(req.Action) {
		return fmt.Errorf("%w: invalid issue action request", ErrIssueActionUnavailable)
	}
	reason := "禅道原生问题操作接口尚未配置"
	if g != nil && g.reason != "" {
		reason = g.reason
	}
	return fmt.Errorf("%w: %s", ErrIssueActionUnavailable, reason)
}

func ValidIssueAction(action string) (IssueAction, bool) {
	value := IssueAction(strings.TrimSpace(strings.ToLower(action)))
	return value, validIssueAction(value)
}

func validIssueAction(action IssueAction) bool {
	return action == IssueActionResolve || action == IssueActionClose || action == IssueActionActivate
}
