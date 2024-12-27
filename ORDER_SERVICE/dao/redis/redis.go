package redis

import (
	"ORDER_SERVICE/config"
	"context"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	Rs *redsync.Redsync
)

func Init(cfg *config.RedisConfig) error {
	rc := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	//判断redis是否连接成功,ping一下
	err := rc.Ping(context.Background()).Err()
	zap.L().Info("我连上了")
	if err != nil {
		return err
	}

	// 基于redis的连接创建redsync obj
	pool := goredis.NewPool(rc)

	// Create an instance of redisync to be used to obtain a mutual exclusion
	// lock.
	Rs = redsync.New(pool)
	return nil
}
