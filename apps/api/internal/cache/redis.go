package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/agungardiyanta/UrlShorter/apps/api/internal/domain"
	"github.com/redis/go-redis/v9"
)

var ErrMiss = errors.New("cache miss")

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(ctx context.Context, addr, password string, db int, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisCache{client: client, ttl: ttl}, nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) GetLink(ctx context.Context, code string) (domain.Link, error) {
	payload, err := c.client.Get(ctx, key(code)).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.Link{}, ErrMiss
	}
	if err != nil {
		return domain.Link{}, err
	}
	var link domain.Link
	if err := json.Unmarshal(payload, &link); err != nil {
		return domain.Link{}, err
	}
	return link, nil
}

func (c *RedisCache) SetLink(ctx context.Context, link domain.Link) error {
	if link.DeletedAt != nil {
		return c.DeleteLink(ctx, link.Code)
	}
	payload, err := json.Marshal(link)
	if err != nil {
		return err
	}
	ttl := c.ttl
	if link.ExpiresAt != nil {
		untilExpiry := time.Until(*link.ExpiresAt)
		if untilExpiry <= 0 {
			return c.DeleteLink(ctx, link.Code)
		}
		if untilExpiry < ttl {
			ttl = untilExpiry
		}
	}
	return c.client.Set(ctx, key(link.Code), payload, ttl).Err()
}

func (c *RedisCache) DeleteLink(ctx context.Context, code string) error {
	return c.client.Del(ctx, key(code)).Err()
}

func key(code string) string {
	return "urlshorter:link:" + code
}
