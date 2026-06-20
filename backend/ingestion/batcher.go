package ingestion

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"mailercloud-backend/model"
	"mailercloud-backend/store"
)

// Batcher consumes events from a queue and flushes them to a store in
// bulk batches. Workers use blocking pop to wait for events, then
// drain the queue non-blockingly to fill a batch before flushing.
//
// Depends on store.EventConsumer (queue read-side) and store.EventStore
// (persistence) — both are interfaces, enabling testing with in-memory
// implementations.
type Batcher struct {
	consumer      store.EventConsumer
	eventStore    store.EventStore
	batchSize     int
	flushInterval time.Duration
	numWorkers    int
	wg            sync.WaitGroup
	stopCh        chan struct{}

	// Metrics (atomic)
	flushed int64
}

// Option configures a Batcher. Use With* functions to create options.
type Option func(*Batcher)

// WithBatchSize sets the maximum number of events per flush.
func WithBatchSize(n int) Option {
	return func(b *Batcher) { b.batchSize = n }
}

// WithWorkers sets the number of concurrent batch writer goroutines.
func WithWorkers(n int) Option {
	return func(b *Batcher) { b.numWorkers = n }
}

// WithFlushInterval sets the maximum time between flushes.
func WithFlushInterval(d time.Duration) Option {
	return func(b *Batcher) { b.flushInterval = d }
}

// NewBatcher creates a Batcher with functional options.
// Sensible defaults are applied; each option overrides one setting.
func NewBatcher(consumer store.EventConsumer, eventStore store.EventStore, opts ...Option) *Batcher {
	b := &Batcher{
		consumer:      consumer,
		eventStore:    eventStore,
		batchSize:     2000,
		flushInterval: 500 * time.Millisecond,
		numWorkers:    4,
		stopCh:        make(chan struct{}),
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Start launches the batch writer goroutines.
func (b *Batcher) Start() {
	for i := 0; i < b.numWorkers; i++ {
		b.wg.Add(1)
		go b.worker(i)
	}
	slog.Info("batcher started",
		"workers", b.numWorkers,
		"batch_size", b.batchSize,
		"flush_interval", b.flushInterval.String(),
	)
}

// Stop signals workers to drain and waits for completion.
func (b *Batcher) Stop() {
	close(b.stopCh)
	b.wg.Wait()
	slog.Info("batcher shutdown complete",
		"total_flushed", atomic.LoadInt64(&b.flushed),
	)
}

// worker consumes events from the queue and flushes batches to the store.
// Uses an event-driven pattern:
//  1. BlockingPop blocks until at least one event is available (no CPU waste)
//  2. Drain loop rapidly pops remaining events up to batchSize
//  3. Flush the batch via EventStore
func (b *Batcher) worker(id int) {
	defer b.wg.Done()
	ctx := context.Background()

	for {
		select {
		case <-b.stopCh:
			// Drain any remaining events before exiting
			b.drainAndFlush(id, ctx)
			return
		default:
		}

		// Step 1: Block-wait for the first event (up to flushInterval)
		event, ok := b.consumer.BlockingPop(ctx, b.flushInterval)
		if !ok {
			continue // timeout, loop back to check stopCh
		}

		// Step 2: Got one event — drain more non-blockingly up to batchSize
		batch := make([]model.Event, 0, b.batchSize)
		batch = append(batch, event)

		if b.batchSize > 1 {
			more := b.consumer.DrainBatch(ctx, b.batchSize-1)
			batch = append(batch, more...)
		}

		// Step 3: Flush to store
		b.flush(id, ctx, batch)
	}
}

// drainAndFlush drains all remaining events from the queue before shutdown.
func (b *Batcher) drainAndFlush(id int, ctx context.Context) {
	for {
		events := b.consumer.DrainBatch(ctx, b.batchSize)
		if len(events) == 0 {
			return
		}
		b.flush(id, ctx, events)
	}
}

// flush writes a batch of events to the store. Retries up to 3 times on failure.
func (b *Batcher) flush(workerID int, ctx context.Context, batch []model.Event) {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		err = b.eventStore.FlushBatch(ctx, batch)
		if err == nil {
			atomic.AddInt64(&b.flushed, int64(len(batch)))
			return
		}
		slog.Error("flush failed",
			"worker", workerID,
			"attempt", attempt,
			"batch_size", len(batch),
			"error", err,
		)
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}
	slog.Error("dead letter: events lost after retries",
		"worker", workerID,
		"events_lost", len(batch),
		"last_error", err,
	)
}
