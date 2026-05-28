package limiter

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/go-redis/redis/v8"
)

//go:embed lua/rate_limit.lua
var rateLimitScript string

type RedisLimiter struct {
	client         *redis.Client
	limitScriptSHA string
	mu             sync.RWMutex
}

var (
	instance *RedisLimiter
	once     sync.Once
)

func New(ctx context.Context, r *redis.Client) *RedisLimiter {
	once.Do(func() {
		// 预加载脚本
		limitSHA, err := r.ScriptLoad(ctx, rateLimitScript).Result()
		if err != nil {
			common.SysLog(fmt.Sprintf("Failed to load rate limit script: %v", err))
		}
		instance = &RedisLimiter{
			client:         r,
			limitScriptSHA: limitSHA,
		}
	})

	return instance
}

func (rl *RedisLimiter) scriptSHA() string {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.limitScriptSHA
}

func (rl *RedisLimiter) reloadScript(ctx context.Context) error {
	limitSHA, err := rl.client.ScriptLoad(ctx, rateLimitScript).Result()
	if err != nil {
		return err
	}

	rl.mu.Lock()
	rl.limitScriptSHA = limitSHA
	rl.mu.Unlock()
	return nil
}

func isNoScriptError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NOSCRIPT")
}

func (rl *RedisLimiter) evalSha(ctx context.Context, key string, config *Config) (int, error) {
	return rl.client.EvalSha(
		ctx,
		rl.scriptSHA(),
		[]string{key},
		config.Requested,
		config.Rate,
		config.Capacity,
	).Int()
}

func (rl *RedisLimiter) Allow(ctx context.Context, key string, opts ...Option) (bool, error) {
	// 默认配置
	config := &Config{
		Capacity:  10,
		Rate:      1,
		Requested: 1,
	}

	// 应用选项模式
	for _, opt := range opts {
		opt(config)
	}

	// 执行限流
	result, err := rl.evalSha(ctx, key, config)
	if isNoScriptError(err) {
		if reloadErr := rl.reloadScript(ctx); reloadErr != nil {
			return false, fmt.Errorf("reload rate limit script failed: %w", reloadErr)
		}
		result, err = rl.evalSha(ctx, key, config)
	}

	if err != nil {
		return false, fmt.Errorf("rate limit failed: %w", err)
	}
	return result == 1, nil
}

// Config 配置选项模式
type Config struct {
	Capacity  int64
	Rate      int64
	Requested int64
}

type Option func(*Config)

func WithCapacity(c int64) Option {
	return func(cfg *Config) { cfg.Capacity = c }
}

func WithRate(r int64) Option {
	return func(cfg *Config) { cfg.Rate = r }
}

func WithRequested(n int64) Option {
	return func(cfg *Config) { cfg.Requested = n }
}
