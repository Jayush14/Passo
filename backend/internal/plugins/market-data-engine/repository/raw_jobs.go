package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// RawJobRepository handles storage and retrieval of raw market data
type RawJobRepository struct {
	db *pgxpool.Pool
}

// NewRawJobRepository creates a new RawJobRepository
func NewRawJobRepository(db *pgxpool.Pool) *RawJobRepository {
	return &RawJobRepository{db: db}
}

// StoreRawData stores raw market data and returns its ID
func (r *RawJobRepository) StoreRawData(ctx context.Context, data *models.RawMarketData) (uuid.UUID, error) {
	query := `
		INSERT INTO raw_market_data
		(id, data_type, raw_content, metadata, content_hash, ingestion_batch_id, received_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (content_hash) DO UPDATE SET
			status = EXCLUDED.status,
			received_at = EXCLUDED.received_at,
			updated_at = NOW()
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		data.ID,
		data.DataType,
		data.RawContent,
		data.Metadata,
		data.ContentHash,
		data.IngestionBatchID,
		data.ReceivedAt,
		data.Status,
	).Scan(&id)

	return id, err
}

// GetRawDataByID retrieves raw data by its ID
func (r *RawJobRepository) GetRawDataByID(ctx context.Context, id uuid.UUID) (*models.RawMarketData, error) {
	query := `
		SELECT id, data_type, raw_content, metadata, content_hash, ingestion_batch_id,
		       received_at, status, processed_at, error_message
		FROM raw_market_data
		WHERE id = $1
	`

	var data models.RawMarketData
	err := r.db.QueryRow(ctx, query, id).Scan(
		&data.ID, &data.DataType, &data.RawContent, &data.Metadata, &data.ContentHash,
		&data.IngestionBatchID, &data.ReceivedAt, &data.Status, &data.ProcessedAt, &data.ErrorMessage,
	)

	if err != nil {
		return nil, err
	}

	return &data, nil
}

// GetRawDataByContentHash retrieves raw data by its content hash (for idempotency)
func (r *RawJobRepository) GetRawDataByContentHash(ctx context.Context, contentHash string) (*models.RawMarketData, error) {
	query := `
		SELECT id, data_type, raw_content, metadata, content_hash, ingestion_batch_id,
		       received_at, status, processed_at, error_message
		FROM raw_market_data
		WHERE content_hash = $1
	`

	var data models.RawMarketData
	err := r.db.QueryRow(ctx, query, contentHash).Scan(
		&data.ID, &data.DataType, &data.RawContent, &data.Metadata, &data.ContentHash,
		&data.IngestionBatchID, &data.ReceivedAt, &data.Status, &data.ProcessedAt, &data.ErrorMessage,
	)

	if err != nil {
		return nil, err
	}

	return &data, nil
}

// UpdateRawDataStatus updates the processing status of raw data
func (r *RawJobRepository) UpdateRawDataStatus(ctx context.Context, id uuid.UUID, status string, processedAt *time.Time, errorMessage *string) error {
	query := `
		UPDATE raw_market_data
		SET status = $1, processed_at = $2, error_message = $3, updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, status, processedAt, errorMessage, id)
	return err
}

// ProcessingJobRepository handles storage and retrieval of processing jobs
type ProcessingJobRepository struct {
	db *pgxpool.Pool
}

// NewProcessingJobRepository creates a new ProcessingJobRepository
func NewProcessingJobRepository(db *pgxpool.Pool) *ProcessingJobRepository {
	return &ProcessingJobRepository{db: db}
}

// CreateProcessingJob creates a new processing job for raw data
func (r *ProcessingJobRepository) CreateProcessingJob(ctx context.Context, rawDataID uuid.UUID) (uuid.UUID, error) {
	query := `
		INSERT INTO processing_jobs (raw_data_id, status)
		VALUES ($1, 'PENDING')
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, rawDataID).Scan(&id)
	return id, err
}

// GetProcessingJobByID retrieves a processing job by its ID
func (r *ProcessingJobRepository) GetProcessingJobByID(ctx context.Context, id uuid.UUID) (*models.ProcessingJob, error) {
	query := `
		SELECT id, raw_data_id, status, attempts, last_error, started_at, completed_at, created_at, updated_at
		FROM processing_jobs
		WHERE id = $1
	`

	var job models.ProcessingJob
	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.RawDataID, &job.Status, &job.Attempts, &job.LastError,
		&job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}

// GetProcessingJobByRawDataID retrieves a processing job by its raw data ID
func (r *ProcessingJobRepository) GetProcessingJobByRawDataID(ctx context.Context, rawDataID uuid.UUID) (*models.ProcessingJob, error) {
	query := `
		SELECT id, raw_data_id, status, attempts, last_error, started_at, completed_at, created_at, updated_at
		FROM processing_jobs
		WHERE raw_data_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var job models.ProcessingJob
	err := r.db.QueryRow(ctx, query, rawDataID).Scan(
		&job.ID, &job.RawDataID, &job.Status, &job.Attempts, &job.LastError,
		&job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}

// UpdateProcessingJobStatus updates the status of a processing job
func (r *ProcessingJobRepository) UpdateProcessingJobStatus(ctx context.Context, jobID uuid.UUID, status string, errorMsg *string) error {
	query := `
		UPDATE processing_jobs
		SET status = $1, last_error = $2,
		    started_at = CASE WHEN $1 = 'PROCESSING' THEN NOW() ELSE started_at END,
		    completed_at = CASE WHEN $1 IN ('COMPLETED', 'FAILED', 'REVIEW_REQUIRED') THEN NOW() ELSE completed_at END,
		    attempts = attempts + 1,
		    updated_at = NOW()
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, status, errorMsg, jobID)
	return err
}

// ListProcessingJobsByStatus retrieves processing jobs by status
func (r *ProcessingJobRepository) ListProcessingJobsByStatus(ctx context.Context, status string, limit int) ([]models.ProcessingJob, error) {
	query := `
		SELECT id, raw_data_id, status, attempts, last_error, started_at, completed_at, created_at, updated_at
		FROM processing_jobs
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.ProcessingJob
	for rows.Next() {
		var job models.ProcessingJob
		if err := rows.Scan(
			&job.ID, &job.RawDataID, &job.Status, &job.Attempts, &job.LastError,
			&job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}