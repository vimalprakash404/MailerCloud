// Load generator CLI — simulates burst email campaign traffic.
// Scenario: up to 20,000 events/second at peak, in bursts —
// a single campaign send can fire 2 million sent events in under a minute.
// No events may be lost.
//
// Environment variables:
//   BACKEND_URL   — target URL (default: http://localhost:8080)
//   CAMPAIGN_ID   — campaign to send events for (default: camp-1)
//   TOTAL_EVENTS  — total events to send (default: 2000000)
//   CONCURRENCY   — target events per second / concurrent workers (default: 20000)

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	backendURL := getEnv("BACKEND_URL", "http://127.0.0.1:8080")
	campaignID := getEnv("CAMPAIGN_ID", "camp-1")
	totalEvents := envInt("TOTAL_EVENTS", 2000000)
	concurrency := envInt("CONCURRENCY", 20000)

	// Batch size equals concurrency (20,000 events per HTTP request)
	batchSize := concurrency

	// Calculate number of HTTP requests needed
	numBatches := (totalEvents + batchSize - 1) / batchSize

	// Use a reasonable number of goroutine workers (not 20K goroutines)
	// Cap workers at numBatches or 100, whichever is smaller
	numWorkers := numBatches
	if numWorkers > 100 {
		numWorkers = 100
	}

	log.Printf("╔══════════════════════════════════════════════════════╗")
	log.Printf("║  MailerCloud Load Generator — Burst Mode            ║")
	log.Printf("╠══════════════════════════════════════════════════════╣")
	log.Printf("║  Target:         %s", backendURL)
	log.Printf("║  Campaign:       %s", campaignID)
	log.Printf("║  Total Events:   %d", totalEvents)
	log.Printf("║  Events/Batch:   %d", batchSize)
	log.Printf("║  HTTP Requests:  %d", numBatches)
	log.Printf("║  Workers:        %d", numWorkers)
	log.Printf("╚══════════════════════════════════════════════════════╝")

	// HTTP client with connection pooling tuned for burst traffic
	maxConns := numWorkers
	if maxConns > 200 {
		maxConns = 200
	}
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        maxConns * 2,
			MaxIdleConnsPerHost: maxConns * 2,
			MaxConnsPerHost:     maxConns,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	var totalSent int64
	var totalErrors int64
	var wg sync.WaitGroup

	// Build all batches upfront
	batches := make([][]Event, 0, numBatches)
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
	ch := make(chan int, len(batches)) // send batch index
	for i := range batches {
		ch <- i
	}
	close(ch)

	start := time.Now()

	// Launch workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for batchIdx := range ch {
				batch := batches[batchIdx]
				batchReq := BatchRequest{Events: batch}
				body, _ := json.Marshal(batchReq)

				// Retry loop with exponential backoff
				var success bool
				for attempt := 0; attempt < 8; attempt++ {
					resp, err := client.Post(backendURL+"/events/batch", "application/json", bytes.NewReader(body))
					if err != nil {
						if attempt == 7 {
							log.Printf("ERROR: batch post failed after 8 attempts: %v", err)
						}
						jitter := time.Duration(rand.Intn(100)) * time.Millisecond
						time.Sleep(time.Duration(attempt+1)*200*time.Millisecond + jitter)
						continue
					}

					var result BatchResponse
					decodeErr := json.NewDecoder(resp.Body).Decode(&result)
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

					if decodeErr != nil {
						if attempt == 7 {
							log.Printf("ERROR: decode response failed after 8 attempts: %v", decodeErr)
						}
						jitter := time.Duration(rand.Intn(100)) * time.Millisecond
						time.Sleep(time.Duration(attempt+1)*200*time.Millisecond + jitter)
						continue
					}

					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						atomic.AddInt64(&totalSent, int64(result.Accepted))
						atomic.AddInt64(&totalErrors, int64(result.Dropped))
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
						jitter := time.Duration(rand.Intn(100)) * time.Millisecond
						time.Sleep(time.Duration(attempt+1)*200*time.Millisecond + jitter)
						continue
					}
					// Non-retryable
					break
				}
				if !success {
					atomic.AddInt64(&totalErrors, int64(len(batch)))
				}

				// Progress: show cumulative events dispatched in multiples of batchSize
				currentSent := atomic.LoadInt64(&totalSent)
				currentErrors := atomic.LoadInt64(&totalErrors)
				dispatched := currentSent + currentErrors
				elapsed := time.Since(start).Seconds()
				rate := float64(currentSent) / elapsed
				log.Printf("  Progress: %d/%d events (%.0f accepted/sec)",
					dispatched, totalEvents, rate)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	log.Printf("╔══════════════════════════════════════════════════════╗")
	log.Printf("║  Results                                            ║")
	log.Printf("╠══════════════════════════════════════════════════════╣")
	log.Printf("║  Sent:     %d", totalSent)
	log.Printf("║  Errors:   %d", totalErrors)
	log.Printf("║  Duration: %v", elapsed.Round(time.Millisecond))
	if elapsed.Seconds() > 0 {
		log.Printf("║  Rate:     %.0f events/sec", float64(totalSent)/elapsed.Seconds())
	}
	log.Printf("╚══════════════════════════════════════════════════════╝")
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

func init() {
	loadEnvFile(".env")
}

func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Strip quotes
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}
