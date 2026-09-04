package handlers

import (
	"encoding/json"
	"net/http"

	"passo/backend/internal/plugins/market-data-engine/processing"
)

// ProcessingHandler handles HTTP requests for market data engine processing
type ProcessingHandler struct {
	workerPool *processing.WorkerPool
}

// NewProcessingHandler creates a new ProcessingHandler
func NewProcessingHandler(workerPool *processing.WorkerPool) *ProcessingHandler {
	return &ProcessingHandler{
		workerPool: workerPool,
	}
}

// StartProcessing starts the worker pool processing
// @Summary Start MDE processing
// @Description Start the worker pool to process pending job extractions
// @Tags processing
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mde/process/start [post]
func (h *ProcessingHandler) StartProcessing(w http.ResponseWriter, r *http.Request) {
	if err := h.workerPool.Start(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "MDE processing started"})
}

// StopProcessing stops the worker pool processing
// @Summary Stop MDE processing
// @Description Stop the worker pool processing
// @Tags processing
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/mde/process/stop [post]
func (h *ProcessingHandler) StopProcessing(w http.ResponseWriter, _ *http.Request) {
	h.workerPool.Stop()
	writeJSON(w, http.StatusOK, map[string]string{"message": "MDE processing stopped"})
}

// GetProcessingStatus returns the current processing status
// @Summary Get MDE processing status
// @Description Get the current status of the MDE worker pool
// @Tags processing
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/mde/process/status [get]
func (h *ProcessingHandler) GetProcessingStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"running": h.workerPool.IsRunning(),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
