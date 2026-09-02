// =============================================================================
// 文件: internal/pkg/ratelimit/ratelimit.go
// 模块: 基础设施
// 类型: infra
// 职责: 封装进程内限流器初始化与访问控制。
// 依赖: 无
// =============================================================================

package ratelimit

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"workbench/internal/pkg/render"
)

const (
	defaultCleanupInterval = time.Minute
	defaultStaleDuration   = 10 * time.Minute
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Limiter 提供进程内按 key 限流能力，并定期清理长期未使用条目。
type Limiter struct {
	mu              sync.Mutex
	items           map[string]*limiterEntry
	rps             rate.Limit
	burst           int
	cleanupInterval time.Duration
	staleDuration   time.Duration
}

// New 创建一个按 key 限流器。
func New(rps float64, burst int) *Limiter {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = 1
	}
	l := &Limiter{
		items:           make(map[string]*limiterEntry),
		rps:             rate.Limit(rps),
		burst:           burst,
		cleanupInterval: defaultCleanupInterval,
		staleDuration:   defaultStaleDuration,
	}
	go l.cleanupLoop()
	return l
}

// Allow 判断指定 key 当前是否允许通过。
func (l *Limiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	k := strings.TrimSpace(key)
	if k == "" {
		k = "anonymous"
	}

	now := time.Now()
	l.mu.Lock()
	entry, ok := l.items[k]
	if !ok {
		entry = &limiterEntry{
			limiter:  rate.NewLimiter(l.rps, l.burst),
			lastSeen: now,
		}
		l.items[k] = entry
	} else {
		entry.lastSeen = now
	}
	allow := entry.limiter.Allow()
	l.mu.Unlock()
	return allow
}

// KeyByIPAndAccount 生成用于限流的复合 key。
func KeyByIPAndAccount(ip, account string) string {
	normalizedIP := strings.TrimSpace(ip)
	if normalizedIP == "" {
		normalizedIP = "unknown"
	}
	normalizedAccount := strings.TrimSpace(strings.ToLower(account))
	if normalizedAccount == "" {
		normalizedAccount = "anonymous"
	}
	return normalizedIP + "|" + normalizedAccount
}

func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanupExpired()
	}
}

func (l *Limiter) cleanupExpired() {
	now := time.Now()
	l.mu.Lock()
	for key, entry := range l.items {
		if now.Sub(entry.lastSeen) > l.staleDuration {
			delete(l.items, key)
		}
	}
	l.mu.Unlock()
}

// NewGlobalLimiter 创建全局共享令牌桶限流中间件。
// rps <= 0 时使用默认值 1，避免配置错误导致无限放行。
func NewGlobalLimiter(rps int) gin.HandlerFunc {
	if rps <= 0 {
		rps = 1
	}
	limiter := rate.NewLimiter(rate.Limit(rps), rps)
	return func(c *gin.Context) {
		if limiter.Allow() {
			c.Next()
			return
		}
		render.Error(c, http.StatusTooManyRequests, "请求过于频繁，请稍后重试", nil)
		c.Abort()
	}
}
