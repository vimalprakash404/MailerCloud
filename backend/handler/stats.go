package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"mailercloud-backend/model"
	"mailercloud-backend/store"
)

// WebSocket upgrader — allows all origins (CORS handled at middleware level).
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	// Time allowed to write a message to the peer.
	wsWriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	wsPongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	wsPingPeriod = 30 * time.Second

	// How often to push fresh stats to the client.
	wsStatsPushInterval = 5 * time.Second
)

// StatsHandler handles GET /campaigns/{id}/stats and WS campaigns/{id}/stats/ws.
type StatsHandler struct {
	reader store.StatsReader
}

// NewStatsHandler creates a new StatsHandler backed by any StatsReader.
func NewStatsHandler(reader store.StatsReader) *StatsHandler {
	return &StatsHandler{reader: reader}
}

// ServeHTTP returns aggregated campaign statistics.
func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "campaign id is required"})
		return
	}

	stats, err := h.reader.GetCampaignStats(r.Context(), campaignID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "database error"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// ServeWSHTTP upgrades the connection to WebSocket and streams live stats.
func (h *StatsHandler) ServeWSHTTP(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	if campaignID == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "campaign id is required"})
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "campaign_id", campaignID, "error", err)
		return
	}
	defer conn.Close()

	slog.Info("websocket client connected", "campaign_id", campaignID)

	// Configure read deadline and pong handler for keep-alive.
	conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	// Channel to signal client disconnect.
	done := make(chan struct{})

	// Read pump — detects client disconnects and close frames.
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					slog.Warn("websocket unexpected close", "campaign_id", campaignID, "error", err)
				}
				return
			}
		}
	}()

	statsTicker := time.NewTicker(wsStatsPushInterval)
	defer statsTicker.Stop()

	pingTicker := time.NewTicker(wsPingPeriod)
	defer pingTicker.Stop()

	// Send initial stats immediately.
	stats, err := h.reader.GetCampaignStats(r.Context(), campaignID)
	if err == nil {
		conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
		if wErr := conn.WriteJSON(stats); wErr != nil {
			slog.Error("websocket initial write failed", "campaign_id", campaignID, "error", wErr)
			return
		}
	}

	for {
		select {
		case <-done:
			slog.Info("websocket client disconnected", "campaign_id", campaignID)
			return

		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("websocket ping failed", "campaign_id", campaignID, "error", err)
				return
			}

		case <-statsTicker.C:
			stats, err := h.reader.GetCampaignStats(r.Context(), campaignID)
			if err != nil {
				slog.Error("failed to get campaign stats for websocket", "campaign_id", campaignID, "error", err)
				// Send error frame to client
				errPayload, _ := json.Marshal(model.ErrorResponse{Error: "database error"})
				conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
				conn.WriteMessage(websocket.TextMessage, errPayload)
				continue
			}

			conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteJSON(stats); err != nil {
				slog.Error("websocket write failed", "campaign_id", campaignID, "error", err)
				return
			}

		case <-r.Context().Done():
			slog.Info("websocket request context cancelled", "campaign_id", campaignID)
			// Send close message before returning.
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server shutting down"))
			return
		}
	}
}
