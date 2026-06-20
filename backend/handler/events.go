package handler

import (
	"encoding/json"
	"net/http"

	"mailercloud-backend/model"
	"mailercloud-backend/store"
)

// EventsHandler handles POST /events and POST /events/batch.
// Pushes events to the queue (O(1), never blocks) and returns 202 immediately.
//
// Depends on the store.EventQueue interface — not on a concrete Redis type.
// This enables unit testing with an in-memory queue implementation.
type EventsHandler struct {
	queue store.EventQueue
}

// NewEventsHandler creates a new EventsHandler backed by any EventQueue.
func NewEventsHandler(q store.EventQueue) *EventsHandler {
	return &EventsHandler{queue: q}
}

// ServeHTTP processes a single event ingestion request.
func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{Error: "method not allowed"})
		return
	}

	var event model.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	if err := event.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	// Use the request context so work is cancelled if the client disconnects.
	if err := h.queue.Push(r.Context(), event); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "queue error: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

// BatchRequest is the JSON payload for POST /events/batch.
type BatchRequest struct {
	Events []model.Event `json:"events"`
}

// BatchResponse reports how many events were accepted vs dropped.
type BatchResponse struct {
	Accepted int `json:"accepted"`
	Dropped  int `json:"dropped"`
	Total    int `json:"total"`
}

// ServeBatchHTTP processes a batch of events in a single request.
// Uses Redis pipeline to push all events in a single round-trip (~1ms for 200 events).
func (h *EventsHandler) ServeBatchHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{Error: "method not allowed"})
		return
	}

	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	if len(req.Events) == 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "events array is empty"})
		return
	}

	// Validate all events
	valid := make([]model.Event, 0, len(req.Events))
	for i := range req.Events {
		if err := req.Events[i].Validate(); err != nil {
			continue
		}
		valid = append(valid, req.Events[i])
	}

	if len(valid) == 0 {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "no valid events in batch"})
		return
	}

	// Push all events — use request context for cancellation.
	accepted, err := h.queue.PushBatch(r.Context(), valid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: "queue error: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, BatchResponse{
		Accepted: accepted,
		Dropped:  len(valid) - accepted,
		Total:    len(req.Events),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
