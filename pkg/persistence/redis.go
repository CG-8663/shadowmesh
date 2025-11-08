package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shadowmesh/shadowmesh/pkg/discovery"
)

// RedisCache handles Redis caching operations
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
	ttl    time.Duration
}

// RedisCacheConfig holds Redis configuration
type RedisCacheConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      time.Duration // Cache TTL (default: 5 minutes)
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(config RedisCacheConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = 5 * time.Minute // Default TTL
	}

	log.Println("Redis connection established")
	return &RedisCache{
		client: client,
		ctx:    ctx,
		ttl:    ttl,
	}, nil
}

// CachePeer caches a peer in Redis
func (rc *RedisCache) CachePeer(peer *discovery.PeerInfo) error {
	key := fmt.Sprintf("peer:%s", peer.PeerID)

	data, err := json.Marshal(peer)
	if err != nil {
		return fmt.Errorf("failed to marshal peer: %w", err)
	}

	return rc.client.Set(rc.ctx, key, data, rc.ttl).Err()
}

// GetCachedPeer retrieves a peer from cache
func (rc *RedisCache) GetCachedPeer(peerID string) (*discovery.PeerInfo, error) {
	key := fmt.Sprintf("peer:%s", peerID)

	data, err := rc.client.Get(rc.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("peer not in cache")
	}
	if err != nil {
		return nil, err
	}

	var peer discovery.PeerInfo
	if err := json.Unmarshal([]byte(data), &peer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal peer: %w", err)
	}

	return &peer, nil
}

// InvalidatePeer removes a peer from cache
func (rc *RedisCache) InvalidatePeer(peerID string) error {
	key := fmt.Sprintf("peer:%s", peerID)
	return rc.client.Del(rc.ctx, key).Err()
}

// CacheSession caches a session token
func (rc *RedisCache) CacheSession(token, peerID string, expiresAt time.Time) error {
	key := fmt.Sprintf("session:%s", token)
	data := map[string]interface{}{
		"peer_id":    peerID,
		"expires_at": expiresAt.Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Set TTL to match session expiry
	ttl := time.Until(expiresAt)
	return rc.client.Set(rc.ctx, key, jsonData, ttl).Err()
}

// GetCachedSession retrieves a session from cache
func (rc *RedisCache) GetCachedSession(token string) (peerID string, expiresAt time.Time, err error) {
	key := fmt.Sprintf("session:%s", token)

	data, err := rc.client.Get(rc.ctx, key).Result()
	if err == redis.Nil {
		return "", time.Time{}, fmt.Errorf("session not in cache")
	}
	if err != nil {
		return "", time.Time{}, err
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
		return "", time.Time{}, err
	}

	peerID = sessionData["peer_id"].(string)
	expiresAtUnix := int64(sessionData["expires_at"].(float64))
	expiresAt = time.Unix(expiresAtUnix, 0)

	return peerID, expiresAt, nil
}

// InvalidateSession removes a session from cache
func (rc *RedisCache) InvalidateSession(token string) error {
	key := fmt.Sprintf("session:%s", token)
	return rc.client.Del(rc.ctx, key).Err()
}

// CachePublicPeers caches the list of public peers
func (rc *RedisCache) CachePublicPeers(peers []*discovery.PeerInfo) error {
	key := "public_peers"

	data, err := json.Marshal(peers)
	if err != nil {
		return fmt.Errorf("failed to marshal public peers: %w", err)
	}

	// Cache for 1 minute (frequently changing)
	return rc.client.Set(rc.ctx, key, data, 1*time.Minute).Err()
}

// GetCachedPublicPeers retrieves cached public peers
func (rc *RedisCache) GetCachedPublicPeers() ([]*discovery.PeerInfo, error) {
	key := "public_peers"

	data, err := rc.client.Get(rc.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("public peers not in cache")
	}
	if err != nil {
		return nil, err
	}

	var peers []*discovery.PeerInfo
	if err := json.Unmarshal([]byte(data), &peers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal public peers: %w", err)
	}

	return peers, nil
}

// CacheClosestPeers caches the result of a FindClosest query
func (rc *RedisCache) CacheClosestPeers(targetID string, peers []*discovery.PeerInfo) error {
	key := fmt.Sprintf("closest:%s", targetID)

	data, err := json.Marshal(peers)
	if err != nil {
		return fmt.Errorf("failed to marshal closest peers: %w", err)
	}

	// Cache for 30 seconds (very dynamic)
	return rc.client.Set(rc.ctx, key, data, 30*time.Second).Err()
}

// GetCachedClosestPeers retrieves cached closest peers
func (rc *RedisCache) GetCachedClosestPeers(targetID string) ([]*discovery.PeerInfo, error) {
	key := fmt.Sprintf("closest:%s", targetID)

	data, err := rc.client.Get(rc.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("closest peers not in cache")
	}
	if err != nil {
		return nil, err
	}

	var peers []*discovery.PeerInfo
	if err := json.Unmarshal([]byte(data), &peers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal closest peers: %w", err)
	}

	return peers, nil
}

// IncrementCounter increments a counter (for metrics)
func (rc *RedisCache) IncrementCounter(name string) error {
	key := fmt.Sprintf("counter:%s", name)
	return rc.client.Incr(rc.ctx, key).Err()
}

// GetCounter retrieves a counter value
func (rc *RedisCache) GetCounter(name string) (int64, error) {
	key := fmt.Sprintf("counter:%s", name)
	return rc.client.Get(rc.ctx, key).Int64()
}

// SetExpiry sets an expiry on a key
func (rc *RedisCache) SetExpiry(key string, duration time.Duration) error {
	return rc.client.Expire(rc.ctx, key, duration).Err()
}

// FlushAll clears all cache (use with caution)
func (rc *RedisCache) FlushAll() error {
	return rc.client.FlushAll(rc.ctx).Err()
}

// GetStats returns Redis cache statistics
func (rc *RedisCache) GetStats() (map[string]interface{}, error) {
	info := rc.client.Info(rc.ctx, "stats")
	if info.Err() != nil {
		return nil, info.Err()
	}

	// Get key counts
	peerCount, _ := rc.client.Keys(rc.ctx, "peer:*").Result()
	sessionCount, _ := rc.client.Keys(rc.ctx, "session:*").Result()

	return map[string]interface{}{
		"cached_peers":    len(peerCount),
		"cached_sessions": len(sessionCount),
		"info":            info.Val(),
	}, nil
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	log.Println("Closing Redis connection")
	return rc.client.Close()
}

// Health checks if Redis is healthy
func (rc *RedisCache) Health() error {
	return rc.client.Ping(rc.ctx).Err()
}
