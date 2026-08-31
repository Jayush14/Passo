package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"passo/backend/internal/plugins/market-data-engine/config"
	"passo/backend/internal/plugins/market-data-engine/ingestion"
)

// IngestionHandler handles HTTP requests for data ingestion
type IngestionHandler struct {
	service *ingestion.IngestionService
	cfg     *config.Config
}

// NewIngestionHandler creates a new IngestionHandler
func NewIngestionHandler(service *ingestion.IngestionService, cfg *config.Config) *IngestionHandler {
	return &IngestionHandler{service: service, cfg: cfg}
}

// HealthCheck returns the health status of the service
func (h *IngestionHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": fmtTimeNow(),
		"version":   "1.0.0",
	})
}

// IngestRawData handles the ingestion of a single piece of raw data
func (h *IngestionHandler) IngestRawData(c *gin.Context) {
	var req ingestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert metadata to interface{} for the service
	var metadata interface{} = req.Metadata
	if req.Metadata == nil {
		metadata = map[string]interface{}{}
	}

	rawData, err := h.service.IngestRawData(c.Request.Context(), req.DataType, req.Content, metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert UUID to string for JSON response
	resp := ingestResponse{
		ID:          rawData.ID.String(),
		Status:      rawData.Status,
		ContentHash: rawData.ContentHash,
	}

	existing := rawData.Status == "RECEIVED"
	if existing {
		c.JSON(http.StatusCreated, resp)
	} else {
		c.JSON(http.StatusOK, resp)
	}
}

// IngestBatchRawData handles the ingestion of multiple pieces of raw data
func (h *IngestionHandler) IngestBatchRawData(c *gin.Context) {
	var req ingestBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert items to the format expected by the service
	items := make([]map[string]interface{}, len(req.Items))
	for i, item := range req.Items {
		items[i] = map[string]interface{}{
			"content":  item.Content,
			"metadata": item.Metadata,
		}
	}

	// Convert batch metadata to interface{}
	var batchMetadata interface{} = req.BatchMetadata
	if req.BatchMetadata == nil {
		batchMetadata = map[string]interface{}{}
	}

	rawData, err := h.service.IngestBatchRawData(c.Request.Context(), req.DataType, items, batchMetadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// For batch processing, we would typically return a batch ID
	// For now, we'll return the last processed item's info
	resp := ingestResponse{
		ID:          rawData.ID.String(),
		Status:      rawData.Status,
		ContentHash: rawData.ContentHash,
	}

	c.JSON(http.StatusAccepted, resp)
}

// GetProcessingStatus returns the processing status for a raw data item
func (h *IngestionHandler) GetProcessingStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID format"})
		return
	}

	procJob, err := h.service.ProcJobRepo().GetProcessingJobByRawDataID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if procJob == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "processing job not found"})
		return
	}

	resp := processingStatusResponse{
		ProcessingID: procJob.ID.String(),
		RawDataID:    procJob.RawDataID.String(),
		Status:       procJob.Status,
		Attempts:     procJob.Attempts,
		LastError:    procJob.LastError,
		StartedAt:    procJob.StartedAt,
		CompletedAt:  procJob.CompletedAt,
		CreatedAt:    procJob.CreatedAt,
		UpdatedAt:    procJob.UpdatedAt,
	}

	c.JSON(http.StatusOK, resp)
}

// Helper functions
func fmtTimeNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Request structures
type ingestRequest struct {
	DataType string      `json:"data_type" binding:"required"`
	Content  string      `json:"content" binding:"required"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type ingestResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	ContentHash string `json:"content_hash"`
}

type ingestBatchRequest struct {
	DataType      string            `json:"data_type" binding:"required"`
	Items         []ingestBatchItem `json:"items" binding:"required,dive"`
	BatchMetadata interface{}       `json:"batch_metadata,omitempty"`
}

type ingestBatchItem struct {
	Content  string      `json:"content" binding:"required"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type processingStatusResponse struct {
	ProcessingID string     `json:"processing_id"`
	RawDataID    string     `json:"raw_data_id"`
	Status       string     `json:"status"`
	Attempts     int        `json:"attempts"`
	LastError    *string    `json:"last_error,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
