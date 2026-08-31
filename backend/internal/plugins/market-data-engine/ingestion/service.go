package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"passo/backend/internal/plugins/market-data-engine/models"
	"passo/backend/internal/plugins/market-data-engine/repository"
)

// IngestionService handles the ingestion of raw market data
type IngestionService struct {
	rawJobRepo  *repository.RawJobRepository
	procJobRepo *repository.ProcessingJobRepository
}

// NewIngestionService creates a new IngestionService
func NewIngestionService(rawJobRepo *repository.RawJobRepository, procJobRepo *repository.ProcessingJobRepository) *IngestionService {
	return &IngestionService{
		rawJobRepo:  rawJobRepo,
		procJobRepo: procJobRepo,
	}
}

// IngestRawData ingests a single piece of raw data
func (s *IngestionService) IngestRawData(ctx context.Context, dataType string, content string, metadata interface{}) (*models.RawMarketData, error) {
	// Generate content hash for idempotency
	hash := sha256.Sum256([]byte(content))
	contentHash := hex.EncodeToString(hash[:])

	// Check if we already have this data
	existing, err := s.rawJobRepo.GetRawDataByContentHash(ctx, contentHash)
	if err == nil && existing != nil {
		// Data already exists, return existing record
		return existing, nil
	}

	// Create new raw data record
	rawData := &models.RawMarketData{
		DataType:    dataType,
		RawContent:  content,
		Metadata:    metadata,
		ContentHash: contentHash,
		Status:      "RECEIVED",
	}

	id, err := s.rawJobRepo.StoreRawData(ctx, rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to store raw data: %w", err)
	}

	rawData.ID = id

	// Create processing job
	_, err = s.procJobRepo.CreateProcessingJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing job: %w", err)
	}

	return rawData, nil
}

// IngestBatchRawData ingests multiple pieces of raw data
func (s *IngestionService) IngestBatchRawData(ctx context.Context, dataType string, items []map[string]interface{}, batchMetadata interface{}) (*models.RawMarketData, error) {
	// For batch processing, we might want to create a batch record
	// For now, we'll process each item individually
	var lastError error
	var lastData *models.RawMarketData

	for _, item := range items {
		content, _ := item["content"].(string)
		metadata := item["metadata"]
		if metadata == nil {
			metadata = batchMetadata
		}

		data, err := s.IngestRawData(ctx, dataType, content, metadata)
		if err != nil {
			lastError = err
			continue
		}
		lastData = data
	}

	if lastError != nil && lastData == nil {
		return nil, fmt.Errorf("failed to ingest batch items: %w", lastError)
	}

	return lastData, lastError
}

// RawJobRepo returns the raw job repository
func (s *IngestionService) RawJobRepo() *repository.RawJobRepository {
	return s.rawJobRepo
}

// ProcJobRepo returns the processing job repository
func (s *IngestionService) ProcJobRepo() *repository.ProcessingJobRepository {
	return s.procJobRepo
}
