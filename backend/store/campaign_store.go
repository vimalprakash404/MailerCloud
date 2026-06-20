package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"mailercloud-backend/model"
)

// CampaignStore implements StatsReader and EventStore using MySQL.
// All raw SQL lives here — handlers and batcher never touch SQL directly.
type CampaignStore struct {
	db *sql.DB
}

// NewCampaignStore creates a new repository backed by the given DB connection.
func NewCampaignStore(db *sql.DB) *CampaignStore {
	return &CampaignStore{db: db}
}

// GetCampaignStats returns aggregated counts for a campaign.
// Returns zero-value stats (not an error) if the campaign has no events.
func (s *CampaignStore) GetCampaignStats(ctx context.Context, campaignID string) (model.CampaignStats, error) {
	var stats model.CampaignStats
	err := s.db.QueryRowContext(ctx, `
		SELECT sent_count, opened_count, clicked_count, bounced_count
		FROM campaign_stats
		WHERE campaign_id = ?
	`, campaignID).Scan(&stats.Sent, &stats.Opened, &stats.Clicked, &stats.Bounced)

	if err == sql.ErrNoRows {
		return model.CampaignStats{}, nil
	}
	if err != nil {
		return model.CampaignStats{}, fmt.Errorf("query campaign_stats: %w", err)
	}
	return stats, nil
}

// FlushBatch writes a batch of events to MySQL using INSERT IGNORE and updates
// campaign_stats atomically in a single transaction.
func (s *CampaignStore) FlushBatch(ctx context.Context, batch []model.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// ── 1. Deduplicate in-memory ────────────────────────────────
	seen := make(map[string]bool, len(batch))
	deduped := make([]model.Event, 0, len(batch))
	for _, e := range batch {
		if !seen[e.EventID] {
			seen[e.EventID] = true
			deduped = append(deduped, e)
		}
	}

	// ── 2. Bulk INSERT IGNORE ───────────────────────────────────
	placeholders := make([]string, len(deduped))
	args := make([]interface{}, 0, len(deduped)*4)
	for i, e := range deduped {
		placeholders[i] = "(?, ?, ?, ?)"
		args = append(args, e.EventID, e.CampaignID, e.Type, e.ParsedTime)
	}

	query := fmt.Sprintf(
		"INSERT IGNORE INTO events (event_id, campaign_id, type, timestamp) VALUES %s",
		strings.Join(placeholders, ", "),
	)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert events: %w", err)
	}

	rowsInserted, _ := result.RowsAffected()
	if rowsInserted == 0 {
		return tx.Commit() // all duplicates — nothing to update
	}

	slog.Debug("batch inserted",
		"deduped", len(deduped),
		"rows_inserted", rowsInserted,
	)

	// ── 3. Aggregate deltas per campaign ────────────────────────
	type delta struct {
		sent, opened, clicked, bounced int
	}
	deltas := make(map[string]*delta)

	for _, e := range deduped {
		d, ok := deltas[e.CampaignID]
		if !ok {
			d = &delta{}
			deltas[e.CampaignID] = d
		}
		switch e.Type {
		case "sent":
			d.sent++
		case "opened":
			d.opened++
		case "clicked":
			d.clicked++
		case "bounced":
			d.bounced++
		}
	}

	// ── 4. Single stats UPSERT per campaign ─────────────────────
	for campaignID, d := range deltas {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO campaign_stats (campaign_id, sent_count, opened_count, clicked_count, bounced_count)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				sent_count    = sent_count    + VALUES(sent_count),
				opened_count  = opened_count  + VALUES(opened_count),
				clicked_count = clicked_count + VALUES(clicked_count),
				bounced_count = bounced_count + VALUES(bounced_count)
		`, campaignID, d.sent, d.opened, d.clicked, d.bounced)
		if err != nil {
			return fmt.Errorf("update stats for campaign %s: %w", campaignID, err)
		}
	}

	return tx.Commit()
}
