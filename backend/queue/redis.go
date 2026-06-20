package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"mailercloud-backend/config"
	"mailercloud-backend/model"
)

const queueKey = "mailercloud:events"

// RedisQueue provides a high-throughput event queue backed by a Redis List.
// LPUSH is O(1) and handles 100k+ ops/sec — HTTP handlers never block.
// Workers consume via BRPOP (blocking pop) for efficient batching.
//
// Implements store.EventQueue and store.EventConsumer interfaces.
type RedisQueue struct {
	client *redis.Client
}

// NewRedisQueue connects to Redis using the provided configuration.
// Retries the connection up to 30 times to handle Docker startup races.
func NewRedisQueue(cfg config.RedisConfig) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx := context.Background()
	for i := 0; i < 30; i++ {
		if err := client.Ping(ctx).Err(); err == nil {
			slog.Info("redis connected",
				"addr", cfg.Addr,
				"pool_size", cfg.PoolSize,
			)
			return &RedisQueue{client: client}, nil
		}
		slog.Warn("waiting for redis",
			"attempt", i+1,
			"max_attempts", 30,
		)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to Redis after 30 retries")
}

// Push adds a single event to the queue. O(1), never blocks.
func (q *RedisQueue) Push(ctx context.Context, event model.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return q.client.LPush(ctx, queueKey, data).Err()
}

// PushBatch adds multiple events using a single LPush command (variadic).
// Returns the number of events successfully pushed.
func (q *RedisQueue) PushBatch(ctx context.Context, events []model.Event) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	values := make([]interface{}, 0, len(events))
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		values = append(values, data)
	}

	if len(values) == 0 {
		return 0, nil
	}

	err := q.client.LPush(ctx, queueKey, values...).Err()
	if err != nil {
		return 0, fmt.Errorf("lpush batch: %w", err)
	}

	return len(values), nil
}

// BlockingPop waits up to `timeout` for one event from the queue.
// Returns the event and true, or zero-value and false on timeout.
func (q *RedisQueue) BlockingPop(ctx context.Context, timeout time.Duration) (model.Event, bool) {
	result, err := q.client.BRPop(ctx, timeout, queueKey).Result()
	if err != nil {
		return model.Event{}, false
	}
	// result[0] = key name, result[1] = value
	var event model.Event
	if err := json.Unmarshal([]byte(result[1]), &event); err != nil {
		return model.Event{}, false
	}
	event.Validate() // re-populate ParsedTime
	return event, true
}

// DrainBatch pops up to `count` events non-blockingly using RPopCount.
func (q *RedisQueue) DrainBatch(ctx context.Context, count int) []model.Event {
	res, err := q.client.RPopCount(ctx, queueKey, count).Result()
	if err != nil {
		return nil
	}

	events := make([]model.Event, 0, len(res))
	for _, data := range res {
		var event model.Event
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		event.Validate()
		events = append(events, event)
	}
	return events
}

// Len returns the approximate queue length.
func (q *RedisQueue) Len(ctx context.Context) int64 {
	val, _ := q.client.LLen(ctx, queueKey).Result()
	return val
}

// Close closes the Redis connection.
func (q *RedisQueue) Close() error {
	return q.client.Close()
}
