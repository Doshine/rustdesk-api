package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"strings"
	"time"
)

var ctx = context.Background()

type RedisCache struct {
	rdb       *redis.Client
	keyPrefix string
}

func RedisCacheInit(conf *redis.Options) *RedisCache {
	return RedisCacheInitWithPrefix(conf, "")
}

func RedisCacheInitWithPrefix(conf *redis.Options, keyPrefix string) *RedisCache {
	c := &RedisCache{keyPrefix: strings.TrimSpace(keyPrefix)}
	c.rdb = redis.NewClient(conf)
	return c
}

func (c *RedisCache) key(key string) string {
	return c.keyPrefix + key
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *RedisCache) Close() error {
	return c.rdb.Close()
}

func (c *RedisCache) Get(key string, value interface{}) error {
	data, err := c.rdb.Get(ctx, c.key(key)).Result()
	if err != nil {
		return err
	}
	err1 := DecodeValue(data, value)
	return err1
}

func (c *RedisCache) Set(key string, value interface{}, exp int) error {
	str, err := EncodeValue(value)
	if err != nil {
		return err
	}
	if exp <= 0 {
		exp = MaxTimeOut
	}
	_, err1 := c.rdb.Set(ctx, c.key(key), str, time.Duration(exp)*time.Second).Result()
	return err1
}

func (c *RedisCache) Delete(key string) error {
	return c.rdb.Del(ctx, c.key(key)).Err()
}

func (c *RedisCache) Increment(key string, exp int) (int64, error) {
	if exp <= 0 {
		exp = MaxTimeOut
	}
	result, err := c.rdb.Eval(ctx, `
local value = redis.call('INCR', KEYS[1])
if value == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
return value
`, []string{c.key(key)}, exp).Int64()
	return result, err
}

func (c *RedisCache) Decrement(key string) error {
	_, err := c.rdb.Eval(ctx, `
local value = redis.call('DECR', KEYS[1])
if value <= 0 then redis.call('DEL', KEYS[1]) end
return value
`, []string{c.key(key)}).Result()
	return err
}

func (c *RedisCache) GetAndDelete(key string, value interface{}) error {
	data, err := c.rdb.Eval(ctx, `
local value = redis.call('GET', KEYS[1])
if value then redis.call('DEL', KEYS[1]) end
return value
`, []string{c.key(key)}).Text()
	if err != nil {
		return err
	}
	return DecodeValue(data, value)
}

func (c *RedisCache) Gc() error {
	return nil
}

func NewRedis(conf *redis.Options) *RedisCache {
	return RedisCacheInit(conf)
}

func NewRedisWithPrefix(conf *redis.Options, keyPrefix string) *RedisCache {
	return RedisCacheInitWithPrefix(conf, keyPrefix)
}
