// =============================================================================
// 文件: internal/module/login/repo.go
// 模块: 登录
// 类型: action
// 职责: 封装认证相关用户查询与登录信息更新操作。
// 依赖: internal/model
// =============================================================================

package login

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// Repo 封装认证相关数据访问。
type Repo struct {
	db          *gorm.DB
	columnCache sync.Map // key: table.column, value: bool
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindUserByAccount 按账号查询用户。未找到返回 (nil, nil)。
func (r *Repo) FindUserByAccount(ctx context.Context, account string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("account = ? AND deleted = ?", account, "0").
		First(&user).Error
	if err == nil {
		return &user, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, err
}

// RecordFailure 写入一次登录失败记录。
func (r *Repo) RecordFailure(ctx context.Context, account, ip string) error {
	accountCol := r.resolveAccountColumn(ctx, "zt_login_failures")
	payload := map[string]any{
		accountCol: account,
		"ip":       ip,
		"failedAt": time.Now(),
	}
	return r.db.WithContext(ctx).Table("zt_login_failures").Create(payload).Error
}

// CleanOldFailures 清理早于指定时间的失败记录。
func (r *Repo) CleanOldFailures(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("failedAt < ?", before).
		Delete(&model.LoginFailure{}).
		Error
}

// InsertLoginLog 写入登录日志。
func (r *Repo) InsertLoginLog(ctx context.Context, log *model.LoginLog) error {
	if log == nil {
		return errors.New("login log is nil")
	}
	accountCol := r.resolveAccountColumn(ctx, "zt_login_logs")
	payload := map[string]any{
		accountCol:   log.Account,
		"userId":     log.UserID,
		"ip":         log.IP,
		"userAgent":  log.UserAgent,
		"success":    log.Success,
		"failReason": log.FailReason,
	}
	if !log.CreatedAt.IsZero() {
		payload["createdDate"] = log.CreatedAt
	}
	return r.db.WithContext(ctx).Table("zt_login_logs").Create(payload).Error
}

func (r *Repo) resolveAccountColumn(ctx context.Context, table string) string {
	if r.hasColumn(ctx, table, "account") {
		return "account"
	}
	if r.hasColumn(ctx, table, "username") {
		return "username"
	}
	return "account"
}

func (r *Repo) hasColumn(ctx context.Context, table, column string) bool {
	cacheKey := table + "." + column
	if v, ok := r.columnCache.Load(cacheKey); ok {
		return v.(bool)
	}
	ok := r.db.WithContext(ctx).Migrator().HasColumn(table, column)
	r.columnCache.Store(cacheKey, ok)
	return ok
}
