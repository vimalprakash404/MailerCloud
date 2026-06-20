package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"mailercloud-backend/model"
	"mailercloud-backend/store"
)

// StatsHandler handles GET /campaigns/{id}/stats.
//
// Depends on the store.StatsReader interface — not on *sql.DB.
// The handler contains zero SQL; all data access is in the repository.
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
