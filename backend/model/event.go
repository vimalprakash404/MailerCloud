package model

import (
	"fmt"
	"time"
)

// ValidEventTypes enumerates the allowed engagement event types.
var ValidEventTypes = map[string]bool{
	"sent":    true,
	"opened":  true,
	"clicked": true,
	"bounced": true,
}

// Event represents a single email engagement event.
type Event struct {
	EventID    string `json:"event_id"`
	CampaignID string `json:"campaign_id"`
	Type       string `json:"type"`
	Timestamp  string `json:"timestamp"`

	// ParsedTime is populated after validation — not serialized.
	ParsedTime time.Time `json:"-"`
}

// Validate checks all fields and parses the timestamp.
func (e *Event) Validate() error {
	if e.EventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if len(e.EventID) > 64 {
		return fmt.Errorf("event_id must be at most 64 characters")
	}
	if e.CampaignID == "" {
		return fmt.Errorf("campaign_id is required")
	}
	if len(e.CampaignID) > 64 {
		return fmt.Errorf("campaign_id must be at most 64 characters")
	}
	if !ValidEventTypes[e.Type] {
		return fmt.Errorf("type must be one of: sent, opened, clicked, bounced")
	}
	if e.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}

	t, err := time.Parse(time.RFC3339, e.Timestamp)
	if err != nil {
		return fmt.Errorf("timestamp must be a valid RFC3339 string: %w", err)
	}
	e.ParsedTime = t
	return nil
}

// CampaignStats holds the aggregated counts for a single campaign.
type CampaignStats struct {
	Sent    int    `json:"sent"`
	Opened  int    `json:"opened"`
	Clicked int    `json:"clicked"`
	Bounced int    `json:"bounced"`
}

// ErrorResponse is the standard JSON error body.
type ErrorResponse struct {
	Error string `json:"error"`
}
