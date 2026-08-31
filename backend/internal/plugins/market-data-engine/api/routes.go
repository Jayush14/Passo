package api

import (
	"github.com/gin-gonic/gin"
	"passo/backend/internal/plugins/market-data-engine/api/handlers"
)

// SetupRoutes sets up all API routes
func SetupRoutes(router *gin.Engine, ingestionHandler *handlers.IngestionHandler) {
	// Health check
	router.GET("/health", ingestionHandler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Ingestion endpoints
		v1.POST("/raw-data", ingestionHandler.IngestRawData)
		v1.POST("/raw-data/batch", ingestionHandler.IngestBatchRawData)
		v1.GET("/processing/:id", ingestionHandler.GetProcessingStatus)

		// Additional endpoints will be added in later phases
	}
}