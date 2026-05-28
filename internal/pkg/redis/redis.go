// =============================================================================
// 文件: internal/pkg/redis/redis.go
// 模块: 基础设施
// 类型: infra
// 职责: 初始化 Redis 连接并提供关闭能力。
// 依赖: internal/config
// =============================================================================

package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"workbench/internal/config"
)

const (
	dialTimeout  = 5 * time.Second
	readTimeout  = 3 * time.Second
	writeTimeout = 3 * time.Second
	poolSize     = 10
	minIdleConns = 2
)

// Clients 业务 Redis 客户端集合：Source 与 Clean 分别对应配置中的 source_db、clean_db。
type Clients struct {
	Source *goredis.Client
	Clean  *goredis.Client
}

// New 根据配置初始化 Redis 连接并 Ping 校验。
func New(cfg *config.Config, logger *zap.Logger) (*Clients, error) {
	source, err := newClient(cfg, cfg.Redis.SourceDB)
	if err != nil {
		return nil, fmt.Errorf("redis source: %w", err)
	}
	clean, err := newClient(cfg, cfg.Redis.CleanDB)
	if err != nil {
		_ = source.Close()
		return nil, fmt.Errorf("redis clean: %w", err)
	}
	if logger != nil {
		logger.Info("redis connected",
			zap.String("addr", addr(cfg)),
			zap.Int("source_db", cfg.Redis.SourceDB),
			zap.Int("clean_db", cfg.Redis.CleanDB),
		)
	}
	return &Clients{Source: source, Clean: clean}, nil
}

// Close 关闭所有 Redis 连接。
func Close(c *Clients) error {
	if c == nil {
		return nil
	}
	var errs []error
	if c.Source != nil {
		if err := c.Source.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.Clean != nil {
		if err := c.Clean.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("close redis: %v", errs)
}

func newClient(cfg *config.Config, db int) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:         addr(cfg),
		Password:     cfg.Redis.Password,
		DB:           db,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	})
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping db %d: %w", db, err)
	}
	return client, nil
}

func addr(cfg *config.Config) string {
	return fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
}
