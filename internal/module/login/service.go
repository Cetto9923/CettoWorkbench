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
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/encode"
)

// 登录失败锁定策略：
//   - loginFailureWindow：统计失败次数的时间滑动窗口。
//   - accountLockThreshold：同一账号在窗口内失败达此次数后锁定账号，防止密码爆破。
//   - ipLockThreshold：同一 IP 在窗口内失败达此次数后锁定，防止分布式枚举账号。
//     IP 阈值（20）高于单账号阈值（5）：因为一个 IP 后面可能有多个合法账号同时使用。
const (
	loginFailureWindow         = 15 * time.Minute
	accountLockThreshold int64 = 5
	ipLockThreshold      int64 = 20
)

// Service 处理认证业务逻辑。
type Service struct {
	repo       authRepo
	sessionMgr sessionManager
	logger     *zap.Logger
}

type authRepo interface {
	CountFailuresByAccount(ctx context.Context, account string, since time.Time) (int64, error)
	CountFailuresByIP(ctx context.Context, ip string, since time.Time) (int64, error)
	FindUserByAccount(ctx context.Context, account string) (*model.User, error)
	RecordFailure(ctx context.Context, account, ip string) error
	UpdateLastLogin(ctx context.Context, userID int64, ip string) error
	InsertLoginLog(ctx context.Context, log *model.LoginLog) error
}

type sessionManager interface {
	RenewToken(ctx context.Context) error
	Put(ctx context.Context, key string, val interface{})
	Destroy(ctx context.Context) error
}

// NewService 创建 Service。
func NewService(repo *Repo, sessionMgr *scs.SessionManager, logger *zap.Logger) *Service {
	return NewServiceWithDeps(repo, sessionMgr, logger)
}

// NewServiceWithDeps 使用可替换依赖创建 Service（供测试）。
func NewServiceWithDeps(repo authRepo, sessionMgr sessionManager, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:       repo,
		sessionMgr: sessionMgr,
		logger:     logger,
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
	if reason, err := s.checkLockout(ctx, account, req.IP); err != nil {
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

	if err := s.repo.UpdateLastLogin(ctx, user.ID, req.IP); err != nil {
		s.logger.Warn("update last login failed", zap.Int64("userID", user.ID), zap.Error(err))
	}

	s.recordLoginLog(ctx, req, sql.NullInt64{Int64: user.ID, Valid: true}, true, "")
	s.logger.Info("login success", zap.String("account", account), zap.Int64("userID", user.ID))

	return LoginResp{User: user}, nil
}

// checkLockout 检查账号和 IP 是否触发登录失败锁定。
// 返回 (reason, err)：reason 仅在 err != nil 时有意义，用于写登录日志。
func (s *Service) checkLockout(ctx context.Context, account, ip string) (reason string, err error) {
	since := time.Now().Add(-loginFailureWindow)

	accountCount, err := s.repo.CountFailuresByAccount(ctx, account, since)
	if err != nil {
		return "", err
	}
	if accountCount >= accountLockThreshold {
		return "account_locked", errorx.New("auth.login.locked", "账号已被临时锁定，请 15 分钟后再试")
	}

	ipCount, err := s.repo.CountFailuresByIP(ctx, ip, since)
	if err != nil {
		return "", err
	}
	if ipCount >= ipLockThreshold {
		return "ip_locked", errorx.New("auth.login.iplocked", "当前 IP 尝试次数过多，请 15 分钟后再试")
	}

	return "", nil
}

// Logout 执行登出。
func (s *Service) Logout(ctx context.Context, actor *model.User) error {
	// actor 当前未使用：第一期只销毁 Session。
	// 保留此参数是遵循规范“action 类型写操作必须接收 actor”，
	// 后续可在此记录登出日志（actor.ID、IP 等审计信息）。
	_ = actor
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
		TenantID:   0, // 多租户预埋：第一期固定为 0，第三期从 ctx 中读取真实 tenantID。
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
