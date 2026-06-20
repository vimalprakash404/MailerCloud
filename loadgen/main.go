// Load generator CLI — fires configurable bursts using batch POST /events/batch.
// Uses async event-driven architecture: batches events client-side and sends
// them in bulk to minimize HTTP round-trip overhead.
//
// Environment variables:
//   BACKEND_URL   — target URL (default: http://localhost:8080)
//   CAMPAIGN_ID   — campaign to send events for (default: camp-1)
//   TOTAL_EVENTS  — total events to send (default: 10000)
//   CONCURRENCY   — parallel goroutines (default: 50)
//   BATCH_SIZE    — events per HTTP request (default: 200)

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Event struct {
	EventID    string `json:"event_id"`
	CampaignID string `json:"campaign_id"`
	Type       string `json:"type"`
	Timestamp  string `json:"timestamp"`
}

type BatchRequest struct {
	Events []Event `json:"events"`
}

type BatchResponse struct {
	Accepted int `json:"accepted"`
	Dropped  int `json:"dropped"`
	Total    int `json:"total"`
}

var eventTypes = []string{"sent", "opened", "clicked", "bounced"}

func main() {
	backendURL := getEnv("BACKEND_URL", "http://localhost:8080")
	campaignID := getEnv("CAMPAIGN_ID", "camp-1")
	totalEvents := envInt("TOTAL_EVENTS", 10000)
	concurrency := envInt("CONCURRENCY", 50)
	batchSize := envInt("BATCH_SIZE", 200)

	log.Printf("╔══════════════════════════════════════════╗")
	log.Printf("║  MailerCloud Load Generator (Batch)      ║")
	log.Printf("╠══════════════════════════════════════════╣")
	log.Printf("║  Target:      %s", backendURL)
	log.Printf("║  Campaign:    %s", campaignID)
	log.Printf("║  Events:      %d", totalEvents)
	log.Printf("║  Concurrency: %d", concurrency)
	log.Printf("║  Batch Size:  %d", batchSize)
	log.Printf("║  HTTP Reqs:   ~%d", (totalEvents+batchSize-1)/batchSize)
	log.Printf("╚══════════════════════════════════════════╝")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency * 2,
			MaxIdleConnsPerHost: concurrency * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var sent int64
	var errors int64
	var wg sync.WaitGroup

	// Build all batches upfront — split totalEvents into chunks of batchSize
	var batches [][]Event
	for offset := 0; offset < totalEvents; offset += batchSize {
		end := offset + batchSize
		if end > totalEvents {
			end = totalEvents
		}

		batch := make([]Event, 0, end-offset)
		for i := offset; i < end; i++ {
			batch = append(batch, Event{
				EventID:    fmt.Sprintf("load-%s-%d-%d", campaignID, time.Now().UnixNano(), i),
				CampaignID: campaignID,
				Type:       eventTypes[rand.Intn(len(eventTypes))],
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
			})
		}
		batches = append(batches, batch)
	}

	// Feed batches into a work channel
	ch := make(chan []Event, len(batches))
	for _, b := range batches {
		ch <- b
	}
	close(ch)

	start := time.Now()

	// Launch workers — each consumes batch work items from the channel
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range ch {
				batchReq := BatchRequest{Events: batch}
				body, _ := json.Marshal(batchReq)

				// Retry loop with exponential backoff
				var success bool
				for attempt := 0; attempt < 8; attempt++ {
					resp, err := client.Post(backendURL+"/events/batch", "application/json", bytes.NewReader(body))
					if err != nil {
						time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
						continue
					}

					var result BatchResponse
					json.NewDecoder(resp.Body).Decode(&result)
					resp.Body.Close()

					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						atomic.AddInt64(&sent, int64(result.Accepted))
						atomic.AddInt64(&errors, int64(result.Dropped))
						success = true
						break
					}
					if resp.StatusCode == 429 || resp.StatusCode == 503 {
						base := time.Duration(attempt+1) * 500 * time.Millisecond
						jitter := time.Duration(rand.Intn(200)) * time.Millisecond
						time.Sleep(base + jitter)
						continue
					}
					if resp.StatusCode >= 500 {
						time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
						continue
					}
					// Non-retryable error
					break
				}
				if !success {
					atomic.AddInt64(&errors, int64(len(batch)))
				}

				current := atomic.LoadInt64(&sent) + atomic.LoadInt64(&errors)
				if current%(int64(batchSize)*5) < int64(batchSize) {
					elapsed := time.Since(start).Seconds()
					rate := float64(atomic.LoadInt64(&sent)) / elapsed
					log.Printf("  Progress: %d/%d events (%.0f accepted/sec)",
						current, totalEvents, rate)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	log.Printf("╔══════════════════════════════════════════╗")
	log.Printf("║  Results                                 ║")
	log.Printf("╠══════════════════════════════════════════╣")
	log.Printf("║  Sent:     %d", sent)
	log.Printf("║  Errors:   %d", errors)
	log.Printf("║  Duration: %v", elapsed.Round(time.Millisecond))
	if elapsed.Seconds() > 0 {
		log.Printf("║  Rate:     %.0f events/sec", float64(sent)/elapsed.Seconds())
	}
	log.Printf("╚══════════════════════════════════════════╝")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
