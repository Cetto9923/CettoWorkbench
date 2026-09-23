// =============================================================================
// 文件: internal/module/login/service.go
// 模块: 登录
// 类型: action
// 职责: 实现登录与登出业务逻辑，并管理会话状态。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package login

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/pkg/encode"
	"workbench/internal/pkg/errorx"
)

// Service 处理认证业务逻辑。
type Service struct {
	repo       authRepo
	sessionMgr sessionManager
	logger     *zap.Logger
	viewCache  zentaoViewCache // 可选：登录成功清禅道可见范围缓存
}

type authRepo interface {
	FindUserByAccount(ctx context.Context, account string) (*model.User, error)
	RecordFailure(ctx context.Context, account, ip string) error
	InsertLoginLog(ctx context.Context, log *model.LoginLog) error
}

type sessionManager interface {
	RenewToken(ctx context.Context) error
	Put(ctx context.Context, key string, val interface{})
	Destroy(ctx context.Context) error
}

// zentaoViewCache 登录后失效禅道可见范围短缓存（由 user.Service 实现）。
type zentaoViewCache interface {
	InvalidateZentaoView(account string)
}

// NewService 创建 Service。viewCache 可为 nil。
func NewService(repo *Repo, sessionMgr *scs.SessionManager, logger *zap.Logger, viewCache zentaoViewCache) *Service {
	return NewServiceWithDeps(repo, sessionMgr, logger, viewCache)
}

// NewServiceWithDeps 使用可替换依赖创建 Service（供测试）。
func NewServiceWithDeps(repo authRepo, sessionMgr sessionManager, logger *zap.Logger, viewCache zentaoViewCache) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:       repo,
		sessionMgr: sessionMgr,
		logger:     logger,
		viewCache:  viewCache,
	}
}

// LoginResp 登录响应。
type LoginResp struct {
	User *model.User
}

// Login 执行登录。
func (s *Service) Login(ctx context.Context, req LoginReq) (LoginResp, error) {
	req.normalizeLoginAccount()
	account := req.Account
	if reason, err := s.checkLockout(ctx, account); err != nil {
		s.recordLoginLog(ctx, req, sql.NullInt64{}, false, reason)
		s.logger.Warn("login failed", zap.String("account", account), zap.String("reason", reason))
		return LoginResp{}, err
	}

	user, err := s.repo.FindUserByAccount(ctx, account)
	if err != nil {
		return LoginResp{}, err
	}
	if user == nil {
		return LoginResp{}, s.failLogin(ctx, req, sql.NullInt64{}, "user_not_found", "auth.login.failed", "账号或密码错误")
	}
	if !user.IsActiveDB {
		return LoginResp{}, s.failLogin(ctx, req, sql.NullInt64{Int64: user.ID, Valid: true}, "user_disabled", "auth.login.disabled", "账号已被禁用")
	}
	if user.PasswordHash != encode.MD5(req.Password) {
		return LoginResp{}, s.failLogin(ctx, req, sql.NullInt64{Int64: user.ID, Valid: true}, "password_mismatch", "auth.login.failed", "账号或密码错误")
	}

	if err := s.sessionMgr.RenewToken(ctx); err != nil {
		return LoginResp{}, err
	}
	s.sessionMgr.Put(ctx, "userID", user.ID)

	// 登录成功后清禅道可见范围缓存，下次业务请求重新 GET /user。
	if s.viewCache != nil {
		s.viewCache.InvalidateZentaoView(account)
	}

	s.recordLoginLog(ctx, req, sql.NullInt64{Int64: user.ID, Valid: true}, true, "")
	s.logger.Info("login success", zap.String("account", account), zap.Int64("userID", user.ID))

	return LoginResp{User: user}, nil
}

// checkLockout 检查账号是否处于临时锁定（zt_user.locked >= now）。
// 返回 (reason, err)：reason 仅在 err != nil 时有意义，用于写登录日志。
func (s *Service) checkLockout(ctx context.Context, account string) (reason string, err error) {
	user, err := s.repo.FindUserByAccount(ctx, account)
	if err != nil {
		return "", err
	}
	// 账号不存在时不在此拦截，交给后续统一「账号或密码错误」路径。
	if user == nil {
		return "", nil
	}
	// locked 表示锁定截止时间：未过期（>= now）则禁止登录。
	if user.Locked != nil && !user.Locked.Before(time.Now()) {
		return "account_locked", errorx.New("auth.login.locked", "账号已被临时锁定，请 15 分钟后再试")
	}

	return "", nil
}

// Logout 执行登出。
func (s *Service) Logout(ctx context.Context, actor *model.User) error {
	if actor != nil && s.viewCache != nil {
		s.viewCache.InvalidateZentaoView(actor.Account)
	}
	return s.sessionMgr.Destroy(ctx)
}

func (s *Service) failLogin(ctx context.Context, req LoginReq, userID sql.NullInt64, reason, code, msg string) error {
	req.normalizeLoginAccount()
	account := req.Account
	if recErr := s.repo.RecordFailure(ctx, account, req.IP); recErr != nil {
		s.logger.Warn("record login failure failed", zap.String("account", account), zap.Error(recErr))
	}
	s.recordLoginLog(ctx, req, userID, false, reason)
	s.logger.Warn("login failed", zap.String("account", account), zap.String("reason", reason))
	return errorx.New(code, msg)
}

func (s *Service) recordLoginLog(ctx context.Context, req LoginReq, userID sql.NullInt64, success bool, failReason string) {
	req.normalizeLoginAccount()
	loginLog := &model.LoginLog{
		Account:    req.Account,
		UserID:     userID,
		IP:         strings.TrimSpace(req.IP),
		UserAgent:  strings.TrimSpace(req.UserAgent),
		Success:    success,
		FailReason: strings.TrimSpace(failReason),
	}
	if err := s.repo.InsertLoginLog(ctx, loginLog); err != nil {
		s.logger.Warn("insert login log failed",
			zap.String("account", loginLog.Account),
			zap.Bool("success", success),
			zap.String("reason", failReason),
			zap.Error(err),
		)
	}
}
