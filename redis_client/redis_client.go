package redisclient

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rudransh-shrivastava/minotaur/config"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(ctx context.Context) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Envs.RedisHost,
		Password: "",
		DB:       0,
	})

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	return &RedisClient{client: client}, nil
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, bool) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false
	}
	return val, true
}

func (r *RedisClient) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}
