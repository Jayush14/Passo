package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"log"
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
	log.Printf("[INFO] HandleStartProcessing: received request")
	if err := h.workerPool.Start(); err != nil {
		log.Printf("[ERROR] HandleStartProcessing: failed to start worker pool: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("[INFO] HandleStartProcessing: worker pool started successfully")
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
	log.Printf("[INFO] HandleStopProcessing: received request")
	h.workerPool.Stop()
	log.Printf("[INFO] HandleStopProcessing: worker pool stopping")
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
	log.Printf("[DEBUG] HandleGetProcessingStatus: received request")
	running := h.workerPool.IsRunning()
	log.Printf("[DEBUG] HandleGetProcessingStatus: worker pool running=%v", running)
	writeJSON(w, http.StatusOK, map[string]bool{
		"running": running,
	})
}

// HandleNormalizeBatch processes a batch of pending extractions on demand.
// @Summary Process a batch of MDE extractions
// @Description Triggers the worker pool to process a batch of pending job extractions and returns the number processed.
// @Tags processing
// @Accept json
// @Produce json
// @Produce json
// @Param batchSize query int false "Batch size (optional, defaults to config)"
// @Success 200 {object} map[string]int
// @Failure 500 {object} map[string]string
// @Router /mde/process/normalize [post]
func (h *ProcessingHandler) HandleNormalizeBatch(w http.ResponseWriter, r *http.Request) {
	batchSize := h.workerPool.Config().BatchSize
	if v := r.URL.Query().Get("batchSize"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			batchSize = n
		}
	}
	log.Printf("[INFO] HandleNormalizeBatch: received request (batchSize=%d)", batchSize)
	processed, err := h.workerPool.ProcessPendingExtractions(r.Context(), batchSize)
	if err != nil {
		log.Printf("[ERROR] HandleNormalizeBatch: failed to process batch: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	log.Printf("[INFO] HandleNormalizeBatch: processed %d extractions", processed)
	writeJSON(w, http.StatusOK, map[string]int{"processed": processed})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
