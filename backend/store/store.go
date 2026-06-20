// Package store defines the interfaces for data access.
// Interfaces are defined at the consumer side (Go idiom: "accept interfaces,
// return structs"). Concrete implementations live in queue/ and store/.
package store

import (
	"context"
	"time"

	"mailercloud-backend/model"
)

// EventQueue is the write-side of the event queue.
// Implemented by queue.RedisQueue; consumed by handler.EventsHandler.
type EventQueue interface {
	Push(ctx context.Context, event model.Event) error
	PushBatch(ctx context.Context, events []model.Event) (int, error)
}

// EventConsumer is the read-side of the event queue.
// Implemented by queue.RedisQueue; consumed by ingestion.Batcher.
type EventConsumer interface {
	BlockingPop(ctx context.Context, timeout time.Duration) (model.Event, bool)
	DrainBatch(ctx context.Context, count int) []model.Event
}

// StatsReader reads aggregated campaign statistics.
// Implemented by store.CampaignStore; consumed by handler.StatsHandler.
type StatsReader interface {
	GetCampaignStats(ctx context.Context, campaignID string) (model.CampaignStats, error)
}

// EventStore persists batches of events and updates aggregate stats.
// Implemented by store.CampaignStore; consumed by ingestion.Batcher.
type EventStore interface {
	FlushBatch(ctx context.Context, events []model.Event) error
}
