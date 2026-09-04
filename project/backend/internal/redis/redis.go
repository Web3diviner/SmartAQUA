package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type memoryEntry struct {
	data      []byte
	expiresAt *time.Time
}

// Client wraps the Redis client with in-memory fallback functionality
type Client struct {
	client    *redis.Client
	useMemory bool
	memMap    sync.Map
}

// New creates a new Redis client with in-memory fallback for local development
func New(addr, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  1 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[redis] Remote Redis not reachable at %s (%v). Using built-in in-memory cache for local execution.", addr, err)
		return &Client{useMemory: true}, nil
	}

	return &Client{client: rdb}, nil
}

// Set stores a key-value pair with optional expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if c.useMemory {
		var expiresAt *time.Time
		if expiration > 0 {
			t := time.Now().Add(expiration)
			expiresAt = &t
		}
		c.memMap.Store(key, memoryEntry{data: data, expiresAt: expiresAt})
		return nil
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// Get retrieves a value by key and unmarshals it into the provided interface
func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	if c.useMemory {
		val, ok := c.memMap.Load(key)
		if !ok {
			return fmt.Errorf("key not found: %s", key)
		}
		entry := val.(memoryEntry)
		if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
			c.memMap.Delete(key)
			return fmt.Errorf("key not found: %s", key)
		}
		if err := json.Unmarshal(entry.data, dest); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}
		return nil
	}

	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete removes a key from Redis
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if c.useMemory {
		for _, k := range keys {
			c.memMap.Delete(k)
		}
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in Redis
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if c.useMemory {
		val, ok := c.memMap.Load(key)
		if !ok {
			return false, nil
		}
		entry := val.(memoryEntry)
		if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
			c.memMap.Delete(key)
			return false, nil
		}
		return true, nil
	}
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Increment increments a numeric value
func (c *Client) Increment(ctx context.Context, key string) (int64, error) {
	if c.useMemory {
		var current int64
		val, ok := c.memMap.Load(key)
		if ok {
			entry := val.(memoryEntry)
			_ = json.Unmarshal(entry.data, &current)
		}
		current++
		data, _ := json.Marshal(strconv.FormatInt(current, 10))
		c.memMap.Store(key, memoryEntry{data: data})
		return current, nil
	}
	return c.client.Incr(ctx, key).Result()
}

// SetExpiration sets expiration for an existing key
func (c *Client) SetExpiration(ctx context.Context, key string, expiration time.Duration) error {
	if c.useMemory {
		val, ok := c.memMap.Load(key)
		if !ok {
			return nil
		}
		entry := val.(memoryEntry)
		t := time.Now().Add(expiration)
		entry.expiresAt = &t
		c.memMap.Store(key, entry)
		return nil
	}
	return c.client.Expire(ctx, key, expiration).Err()
}

// HealthCheck checks if Redis is accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	if c.useMemory {
		return nil
	}
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.useMemory {
		return nil
	}
	return c.client.Close()
}

// GetClient returns the underlying Redis client for advanced operations
func (c *Client) GetClient() *redis.Client {
	return c.client
}
