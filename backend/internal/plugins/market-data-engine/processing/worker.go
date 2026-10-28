package processing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
	"passo/backend/internal/plugins/market-data-engine/repository"
)

func logWithTS(format string, a ...interface{}) {
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, a...)
	fmt.Println(ts, msg)
}

// WorkerPool manages a pool of workers that process job extractions
type WorkerPool struct {
	db                        *pgxpool.Pool
	normalizer                *Normalizer
	jobRepo                   *repository.JobRepository
	jobPostingRepo            *repository.JobPostingRepository
	jobDescriptionRepo        *repository.JobDescriptionRepository
	jobSkillRepo              *repository.JobSkillRepository
	jobCapabilityRepo         *repository.JobCapabilityRepository
	jobDomainRepo             *repository.JobDomainRepository
	jobSoftSkillRepo          *repository.JobSoftSkillRepository
	companyRepo               *repository.CompanyRepository
	locationRepo              *repository.LocationRepository
	skillRepo                 *repository.SkillRepository
	skillAliasRepo            *repository.SkillAliasRepository
	jobTitleRepo              *repository.JobTitleRepository
	seniorityLevelRepo        *repository.SeniorityLevelRepository
	jobFamilyRepo             *repository.JobFamilyRepository
	employmentTypeRepo        *repository.EmploymentTypeRepository
	workModeRepo              *repository.WorkModeRepository
	capabilityRepo            *repository.CapabilityRepository
	domainRepo                *repository.DomainRepository
	normalizationRunRepo      *repository.NormalizationRunRepository
	failedExtractionRepo      *repository.FailedExtractionRepository
	entityResolutionQueueRepo *repository.EntityResolutionQueueRepository

	config   WorkerConfig
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

// Helper functions

func computeDedupKey(normalizedJob models.NormalizedJobData, jobID uuid.UUID) string {
	// We'll construct a string from source_job_id (JobID), url (empty), location_id, and extracted_json (we don't have here)
	// For simplicity, we'll use JobID, location_id, and a hash of the normalized job's strings.
	// In a real implementation, we would store the raw_response JSONB and compute over its relevant fields.
	// We'll use the job's external ID (JobID) and location ID.
	locID := normalizedJob.Location.City + "," + normalizedJob.Location.State + "," + normalizedJob.Location.Country
	// We'll also include the title and company for dedup.
	title := normalizedJob.Title.Value
	company := ""
	if val, ok := normalizedJob.Company.Value.(string); ok {
		company = val
	}
	data := jobID.String() + "|" + company + "|" + locID + "|" + title
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// WorkerConfig holds configuration for the worker pool
type WorkerConfig struct {
	JobPollInterval                  time.Duration
	MaxConcurrentJobs                int
	BatchSize                        int
	MaxRetryAttempts                 int
	FuzzyDistanceLimit               int
	NormalizationConfidenceThreshold float64
	EnableSyncNormalization          bool
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(db *pgxpool.Pool, config WorkerConfig) *WorkerPool {
	return &WorkerPool{
		db:                        db,
		normalizer:                NewNormalizer(db),
		jobRepo:                   repository.NewJobRepository(db),
		jobPostingRepo:            repository.NewJobPostingRepository(db),
		jobDescriptionRepo:        repository.NewJobDescriptionRepository(db),
		jobSkillRepo:              repository.NewJobSkillRepository(db),
		jobCapabilityRepo:         repository.NewJobCapabilityRepository(db),
		jobDomainRepo:             repository.NewJobDomainRepository(db),
		jobSoftSkillRepo:          repository.NewJobSoftSkillRepository(db),
		companyRepo:               repository.NewCompanyRepository(db),
		locationRepo:              repository.NewLocationRepository(db),
		skillRepo:                 repository.NewSkillRepository(db),
		skillAliasRepo:            repository.NewSkillAliasRepository(db),
		jobTitleRepo:              repository.NewJobTitleRepository(db),
		seniorityLevelRepo:        repository.NewSeniorityLevelRepository(db),
		jobFamilyRepo:             repository.NewJobFamilyRepository(db),
		employmentTypeRepo:        repository.NewEmploymentTypeRepository(db),
		workModeRepo:              repository.NewWorkModeRepository(db),
		capabilityRepo:            repository.NewCapabilityRepository(db),
		domainRepo:                repository.NewDomainRepository(db),
		normalizationRunRepo:      repository.NewNormalizationRunRepository(db),
		failedExtractionRepo:      repository.NewFailedExtractionRepository(db),
		entityResolutionQueueRepo: repository.NewEntityResolutionQueueRepository(db),
		config:                    config,
		running:                   false,
		stopChan:                  make(chan struct{}),
	}
}

// Config returns a copy of the worker pool configuration.
func (wp *WorkerPool) Config() WorkerConfig {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.config
}

// Start begins the worker pool processing
func (wp *WorkerPool) Start() error {

	logWithTS("[INFO] Starting MDE worker pool with config:", wp.config)

	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		logWithTS("[INFO] MDE worker pool already running")
		return nil
	}

	wp.running = true

	// Start workers
	for i := 0; i < wp.config.MaxConcurrentJobs; i++ {
		go wp.worker(i)
		logWithTS("[INFO] Started MDE worker %d", i)
	}

	logWithTS("[INFO] MDE worker pool started")
	return nil
}

// Stop stops the worker pool processing
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		logWithTS("[INFO] MDE worker pool not running")
		return
	}

	close(wp.stopChan)
	wp.running = false
	logWithTS("[INFO] MDE worker pool stopping...")
}

// IsRunning reports whether the worker pool is currently processing jobs.
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return wp.running
}

// ProcessPendingExtractions processes a single batch of pending extractions and returns the number processed.
func (wp *WorkerPool) ProcessPendingExtractions(ctx context.Context, batchSize int) (int, error) {
	return wp.processBatch(ctx, batchSize)
}

// worker is the main worker function that processes job extractions
func (wp *WorkerPool) worker(workerID int) {
	logWithTS("[INFO] workerID=%d: started", workerID)
	defer logWithTS("[INFO] workerID=%d: stopped", workerID)

	for {
		select {
		case <-wp.stopChan:
			logWithTS("[INFO] workerID=%d: received stop signal", workerID)
			return
		default:
			// Process a batch of job extractions
			if count, err := wp.processBatch(context.Background(), wp.config.BatchSize); err != nil {
				// Log error but continue processing
				logWithTS("[ERROR] workerID=%d: error processing batch: %v", workerID, err)
				continue
			} else {
				// Optionally log count for debugging
				if count > 0 {
					logWithTS("[INFO] workerID=%d: processed %d extractions in batch", workerID, count)
				}
			}

			// Wait before processing the next batch
			select {
			case <-wp.stopChan:
				logWithTS("[INFO] workerID=%d: received stop signal during wait", workerID)
				return
			case <-time.After(wp.config.JobPollInterval):
				// Continue to next iteration
			}
		}
	}
}

// processBatch processes a batch of job extractions
func (wp *WorkerPool) processBatch(ctx context.Context, batchSize int) (int, error) {
	logWithTS("[DEBUG] processBatch: starting batch (size=%d)", batchSize)
	// Start a normalization run to track this batch
	runID, err := wp.normalizationRunRepo.Insert(ctx, &models.NormalizationRun{
		ID:        uuid.New(),
		RunType:   "worker",
		Status:    "running",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		logWithTS("[ERROR] processBatch: failed to insert normalization run: %v", err)
		return 0, err
	}

	logWithTS("[DEBUG] processBatch: normalization run %s started", runID)
	// Mark the run as completed when we're done
	defer func() {
		completedAt := time.Now().Format(time.RFC3339)
		if err := wp.normalizationRunRepo.UpdateStatus(ctx, runID, "completed", &completedAt, nil); err != nil {
			logWithTS("[ERROR] processBatch: failed to update normalization run status: %v", err)
		}
		logWithTS("[DEBUG] processBatch: normalization run %s completed", runID)
	}()

	// Get pending job extractions using FOR UPDATE SKIP LOCKED for distributed locking
	rows, err := wp.db.Query(ctx, `
		SELECT id, job_id, raw_response, created_at
		FROM job_extractions
		WHERE mde_processed = FALSE
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`, batchSize)
	if err != nil {
		logWithTS("[ERROR] processBatch: failed to query job_extractions: %v", err)
		return 0, err
	}

	defer rows.Close()

	var processedCount int
	for rows.Next() {
		var jobExtraction models.JobExtraction
		if err := rows.Scan(
			&jobExtraction.ID,
			&jobExtraction.JobID,
			&jobExtraction.RawResponse,
			&jobExtraction.CreatedAt,
		); err != nil {
			// Update the run with error and continue
			logWithTS("[ERROR] processBatch: failed to scan job extraction: %v", err)
			wp.normalizationRunRepo.UpdateProgress(ctx, runID, processedCount, 0, 1)
			wp.failedExtractionRepo.Insert(ctx, &models.FailedExtraction{
				ID:             uuid.New(),
				JobID:          jobExtraction.JobID,
				ExtractionData: string(jobExtraction.RawResponse),
				ErrorMessage:   err.Error(),
				StackTrace:     "", // In a real implementation, we would capture the stack trace
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			})
			// Mark the extraction as processed with error
			processedAt := time.Now().Format(time.RFC3339)
			errorMsg := err.Error()
			wp.jobRepo.UpdateMDEProcessingStatus(ctx, jobExtraction.JobID, true, &processedAt, &errorMsg)

			// Mark the job extraction as processed in the job_extractions table to avoid reprocessing
			if _, err := wp.db.Exec(ctx, `
				UPDATE job_extractions
				SET mde_processed = true,
				    mde_processed_at = NOW(),
				    mde_error_message = $1
				WHERE id = $2
			`, err.Error(), jobExtraction.ID); err != nil {
				logWithTS("[ERROR] processBatch: failed to update job extraction %s as processed: %v", jobExtraction.ID, err)
			}

			continue
		}

		logWithTS("[DEBUG] processBatch: processing extraction %s (jobID=%s)", jobExtraction.ID, jobExtraction.JobID)

		// Process this job extraction
		if err := wp.processJobExtraction(ctx, &jobExtraction); err != nil {
			// Update the run with error and continue
			logWithTS("[ERROR] processBatch: failed to process job extraction %s: %v", jobExtraction.ID, err)
			wp.normalizationRunRepo.UpdateProgress(ctx, runID, processedCount, 0, 1)
			wp.failedExtractionRepo.Insert(ctx, &models.FailedExtraction{
				ID:             uuid.New(),
				JobID:          jobExtraction.JobID,
				ExtractionData: string(jobExtraction.RawResponse),
				ErrorMessage:   err.Error(),
				StackTrace:     "", // In a real implementation, we would capture the stack trace
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			})
			// Mark the extraction as processed with error
			processedAt := time.Now().Format(time.RFC3339)
			errorMsg := err.Error()
			wp.jobRepo.UpdateMDEProcessingStatus(ctx, jobExtraction.JobID, true, &processedAt, &errorMsg)

			// Mark the job extraction as processed in the job_extractions table to avoid reprocessing
			if _, err := wp.db.Exec(ctx, `
				UPDATE job_extractions
				SET mde_processed = true,
				    mde_processed_at = NOW(),
				    mde_error_message = $1
				WHERE id = $2
			`, err.Error(), jobExtraction.ID); err != nil {
				logWithTS("[ERROR] processBatch: failed to update job extraction %s as processed: %v", jobExtraction.ID, err)
			}

			continue
		}

		logWithTS("[DEBUG] processBatch: successfully processed extraction %s", jobExtraction.ID)
		processedCount++

		// Mark the job extraction as processed in the job_extractions table to avoid reprocessing
		if _, err := wp.db.Exec(ctx, `
			UPDATE job_extractions
			SET mde_processed = true,
			    mde_processed_at = NOW(),
			    mde_error_message = NULL
			WHERE id = $1
		`, jobExtraction.ID); err != nil {
			logWithTS("[ERROR] processBatch: failed to update job extraction %s as processed: %v", jobExtraction.ID, err)
		}
	}

	if err := rows.Err(); err != nil {
		logWithTS("[ERROR] processBatch: error after iterating rows: %v", err)
		return processedCount, err
	}

	// Update the run with final progress
	if err := wp.normalizationRunRepo.UpdateProgress(ctx, runID, processedCount, processedCount, 0); err != nil {
		logWithTS("[ERROR] processBatch: failed to update progress: %v", err)
	}

	logWithTS("[INFO] processBatch: completed batch, processed=%d", processedCount)
	return processedCount, nil
}

// processJobExtraction processes a single job extraction
func (wp *WorkerPool) processJobExtraction(ctx context.Context, extraction *models.JobExtraction) error {
	// Unmarshal raw_response into ExtractedJobData
	logWithTS("[DEBUG] processJobExtraction: unmarshaling raw_response for extraction %s", extraction.ID)
	var extracted models.ExtractedJobData
	if err := json.Unmarshal(extraction.RawResponse, &extracted); err != nil {
		return fmt.Errorf("failed to unmarshal raw_response: %w", err)
	}

	// shortcut for logging removed; using logWithTS directly

	// Normalize the extracted job data
	normalizedJob := wp.normalizer.NormalizeJob(extracted)

	logWithTS("[DEBUG] processJobExtraction: normalized job data for extraction %s: %+v", extraction.ID, normalizedJob)

	// Start a transaction
	tx, err := wp.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				logWithTS("[ERROR] processJobExtraction: rollback failed after error: %v", rbErr)
			}
		} else {
			if cmErr := tx.Commit(ctx); cmErr != nil {
				logWithTS("[ERROR] processJobExtraction: commit failed: %v", cmErr)
				err = cmErr
			}
		}
	}()

	logWithTS("[DEBUG] processJobExtraction: transaction started for extraction %s", extraction.ID)
	// Create or get the company
	logWithTS("[TRACE] processJobExtraction: entering company block")
	var companyID *uuid.UUID
	companyValue, _ := normalizedJob.Company.Value.(string)
	if companyValue != "" {
		company, err := wp.companyRepo.GetByCanonicalName(ctx, companyValue)
		logWithTS("[DEBUG] processJobExtraction: retrieved company '%s' for extraction %s", companyValue, extraction.ID)
		if err != nil || company == nil {
			// unknown -> use nil UUID
			logWithTS("[WARN] processJobExtraction: unknown company '%s' (extraction %s) -> using nil UUID", companyValue, extraction.ID)
			// companyID remains nil
		} else {
			companyID = &company.ID
			logWithTS("[DEBUG] processJobExtraction: found company '%s' (ID=%s)", companyValue, *companyID)
		}
	}

	// Create or get the location
	logWithTS("[DEBUG] processJobExtraction: after company lookup, before location lookup")
	var locationID uuid.UUID
	if normalizedJob.Location.City != "" {
		logWithTS("[DEBUG] processJobExtraction: about to call location repo")
		location, err := wp.locationRepo.GetByCityStateCountry(ctx, normalizedJob.Location.City, normalizedJob.Location.State, normalizedJob.Location.Country)
		logWithTS("[DEBUG] processJobExtraction: location repo returned")

		if err != nil || location == nil {
			// unknown -> use nil UUID
			locStr := normalizedJob.Location.City + ", " + normalizedJob.Location.State + ", " + normalizedJob.Location.Country
			logWithTS("[WARN] processJobExtraction: unknown location '%s' (extraction %s) -> using nil UUID", locStr, extraction.ID)
			locationID = uuid.Nil
		} else {
			locationID = location.ID
			logWithTS("[DEBUG] processJobExtraction: found location '%s' (ID=%s)", locationID)
		}
	}
	logWithTS("[DEBUG] processJobExtraction: after location block")

	// Create or get the job title
	var titleID *uuid.UUID
	if normalizedJob.Title.Value != "" {
		title, err := wp.jobTitleRepo.GetByCanonicalName(ctx, normalizedJob.Title.Value)
		if err != nil || title == nil {
			// unknown -> use nil UUID
			logWithTS("[WARN] processJobExtraction: unknown title '%s' (extraction %s) -> using nil UUID", normalizedJob.Title.Value, extraction.ID)
			titleID = nil
		} else {
			titleID = &title.ID
			logWithTS("[DEBUG] processJobExtraction: found title '%s' (ID=%s)", normalizedJob.Title.Value, *titleID)
		}
	}

	// Create or get the seniority level
	var seniorityID *uuid.UUID
	if normalizedJob.Seniority.Value != "" {
		seniority, err := wp.seniorityLevelRepo.GetByCanonicalName(ctx, normalizedJob.Seniority.Value)
		if err != nil || seniority == nil {
			// unknown -> use nil UUID
			logWithTS("[WARN] processJobExtraction: unknown seniority '%s' (extraction %s) -> using nil UUID", normalizedJob.Seniority.Value, extraction.ID)
			seniorityID = nil
		} else {
			seniorityID = &seniority.ID
			logWithTS("[DEBUG] processJobExtraction: found seniority '%s' (ID=%s)", normalizedJob.Seniority.Value, *seniorityID)
		}
	}

	// Create or get the employment type
	var employmentTypeID uuid.UUID
	if normalizedJob.EmploymentType.Value != "" {
		employmentType, err := wp.employmentTypeRepo.GetByCanonicalName(ctx, normalizedJob.EmploymentType.Value)
		if err != nil || employmentType == nil {
			// unknown -> use nil UUID
			logWithTS("[WARN] processJobExtraction: unknown employment_type '%s' (extraction %s) -> using nil UUID", normalizedJob.EmploymentType.Value, extraction.ID)
			employmentTypeID = uuid.Nil
		} else {
			employmentTypeID = employmentType.ID
			logWithTS("[DEBUG] processJobExtraction: found employment_type '%s' (ID=%s)", normalizedJob.EmploymentType.Value, employmentTypeID)
		}
	}

	// Create or get the work mode
	var workModeID uuid.UUID
	if normalizedJob.WorkMode.Value != "" {
		workMode, err := wp.workModeRepo.GetByCanonicalName(ctx, normalizedJob.WorkMode.Value)
		if err != nil || workMode == nil {
			// unknown -> use nil UUID
			logWithTS("[WARN] processJobExtraction: unknown work_mode '%s' (extraction %s) -> using nil UUID", normalizedJob.WorkMode.Value, extraction.ID)
			workModeID = uuid.Nil
		} else {
			workModeID = workMode.ID
			logWithTS("[DEBUG] processJobExtraction: found work_mode '%s' (ID=%s)", normalizedJob.WorkMode.Value, workModeID)
		}
	}

	logWithTS("[DEBUG] processJobExtraction: creating job record for extraction %s", extraction.ID)

	// Create or get the job family
	var jobFamilyID *uuid.UUID
	// TODO: implement job family lookup
	// For now, leave as nil
	jobFamilyID = nil

	// Create or get the normalization version
	var normalizationVersionID *uuid.UUID
	// TODO: implement normalization version lookup
	// For now, leave as nil
	normalizationVersionID = nil

	// Create the job
	job := &models.Job{
		ID:                   uuid.New(),
		ExternalID:           extraction.JobID.String(),
		Source:               "unknown",
		SourceURL:            "",
		RawContent:           extraction.RawContent,
		Language:             extraction.Language,
		TitleID:              titleID,
		CompanyID:            companyID,
		SeniorityID:          seniorityID,
		JobFamilyID:          jobFamilyID,
		NormalizationVersion: normalizationVersionID,
		RetrievedAt:          time.Now(),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		MDEProcessed:         true,
		MDEProcessedAt:       func() *string { s := time.Now().Format(time.RFC3339); return &s }(),
		MDEErrorMessage:      nil,
	}

	jobID, err := wp.jobRepo.Insert(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}
	logWithTS("[DEBUG] processJobExtraction: inserted job ID=%s", jobID)

	// Create the job posting with deduplication
	dedupKey := computeDedupKey(normalizedJob, extraction.JobID)
	jobPosting := &models.JobPosting{
		ID:               uuid.New(),
		JobID:            jobID,
		Title:            normalizedJob.Title.Value,
		CompanyID:        companyID,
		LocationID:       locationID,
		EmploymentTypeID: employmentTypeID,
		WorkModeID:       workModeID,
		SalaryMin:        0, // Would be extracted from normalized data in a real implementation
		SalaryMax:        0, // Would be extracted from normalized data in a real implementation
		SalaryCurrency:   "USD",
		SalaryPeriod:     "yearly",
		DedupKey:         dedupKey,
		Description: func() string {
			if len(normalizedJob.Responsibilities) > 0 {
				var desc strings.Builder
				for i, resp := range normalizedJob.Responsibilities {
					if i > 0 {
						desc.WriteString("\n\n")
					}
					desc.WriteString(resp.Description)
				}
				return desc.String()
			}
			return ""
		}(),
		ApplyURL:           "",
		PostedAt:           time.Now(),
		ExpiresAt:          time.Time{},
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		MatchedSkillsCount: len(normalizedJob.Skills),
		TotalSkillsCount:   len(normalizedJob.Skills),
		MatchScore:         1.0,
		Status:             "active",
	}

	postingID, err := wp.jobPostingRepo.Insert(ctx, jobPosting)
	if err != nil {
		// Check if it's a unique violation (dedup key)
		if isUniqueViolation(err) {
			// Duplicate posting: we can still link skills to the existing posting if desired.
			// For simplicity, we'll just mark the extraction as processed and skip further persistence.
			logWithTS("[INFO] processJobExtraction: duplicate posting detected (dedup_key=%s) for extraction %s -> skipping persistence", dedupKey, extraction.ID)
			wp.jobRepo.UpdateMDEProcessingStatus(ctx, extraction.JobID, true, nil, nil)
			// Mark the extraction as processed to avoid re‑processing the same row
			processedAt := time.Now().Format(time.RFC3339)
			if _, err := wp.db.Exec(ctx, `
				UPDATE job_extractions
				SET mde_processed = true,
				    mde_processed_at = $1,
				    mde_error_message = NULL
				WHERE id = $2
			`, processedAt, extraction.ID); err != nil {
				logWithTS("[ERROR] processJobExtraction: failed to mark extraction %s as processed after duplicate: %v", extraction.ID, err)
			}
			return nil
		}
		return fmt.Errorf("failed to insert job posting: %w", err)
	}
	logWithTS("[DEBUG] processJobExtraction: inserted job posting ID=%s", postingID)

	// Create the job description
	jobDescription := &models.JobDescription{
		ID:          uuid.New(),
		JobID:       postingID,
		Title:       jobPosting.Title,
		Description: jobPosting.Description,
	}

	descID, err := wp.jobDescriptionRepo.Insert(ctx, jobDescription)
	if err != nil {
		return fmt.Errorf("failed to insert job description: %w", err)
	}
	logWithTS("[DEBUG] processJobExtraction: inserted job description ID=%s", descID)

	// Create job skills
	for _, skill := range normalizedJob.Skills {
		var skillID uuid.UUID
		var found bool

		if skill.NormalizedName != "" {
			skillModel, err := wp.skillRepo.GetByCanonicalName(ctx, skill.NormalizedName)
			if err == nil && skillModel != nil {
				skillID = skillModel.ID
				found = true
				logWithTS("[DEBUG] processJobExtraction: found skill (normalized) '%s' (ID=%s)", skill.NormalizedName, skillID)
			} else {
				logWithTS("[WARN] processJobExtraction: unknown skill (normalized) '%s' (extraction %s) -> using nil UUID", skill.NormalizedName, extraction.ID)
			}
		} else if skill.RawName != "" {
			// fallback to raw name
			skillModel, err := wp.skillRepo.GetByCanonicalName(ctx, skill.RawName)
			if err == nil && skillModel != nil {
				skillID = skillModel.ID
				found = true
				logWithTS("[DEBUG] processJobExtraction: found skill (raw) '%s' (ID=%s)", skill.RawName, skillID)
			} else {
				logWithTS("[WARN] processJobExtraction: unknown skill (raw) '%s' (extraction %s) -> using nil UUID", skill.RawName, extraction.ID)
			}
		}

		if !found {
			skillID = uuid.Nil
		}

		// Create job skill
		jobSkill := &models.JobSkill{
			ID:              uuid.New(),
			JobID:           jobID,
			SkillID:         skillID,
			RequirementType: skill.RequirementLevel,
			Confidence:      skill.Confidence,
			Evidence:        skill.Evidence,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if _, err := wp.jobSkillRepo.Insert(ctx, jobSkill); err != nil {
			logWithTS("[ERROR] processJobExtraction: failed to insert job skill: %v", err)
			continue
		}
		logWithTS("[DEBUG] processJobExtraction: inserted job skill with skillID=%s", skillID)
	}

	// Create job capabilities from skills
	processedCapabilities := make(map[uuid.UUID]bool)
	for _, skill := range normalizedJob.Skills {
		if skill.Capability == nil {
			continue
		}
		capability := skill.Capability
		capabilityID := capability.ID

		// Skip if we've already processed this capability
		if processedCapabilities[capabilityID] {
			continue
		}
		processedCapabilities[capabilityID] = true

		// Create job capability
		jobCapability := &models.JobCapability{
			ID:    uuid.New(),
			JobID: jobID,
			// CapabilityID: capabilityID,
			Importance: 0.0, // placeholder
			Evidence:   "",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if capabilityID != uuid.Nil {
			jobCapability.CapabilityID = &capabilityID
		}
		if _, err := wp.jobCapabilityRepo.Insert(ctx, jobCapability); err != nil {
			logWithTS("[ERROR] processJobExtraction: failed to insert job capability for capability ID %s: %v", capabilityID, err)
			continue
		}
		logWithTS("[DEBUG] processJobExtraction: inserted job capability for capability ID=%s", capabilityID)
	}

	// Create job soft skills (stub implementation - skipped for now due to schema mismatch)
	// TODO: Fix job_soft_skills table schema to match model expectations
	/* for _, softSkill := range normalizedJob.SoftSkills {
		var softSkillID uuid.UUID
		var found bool

		if softSkill.Name != "" {
			softSkillModel, err := wp.skillRepo.GetByCanonicalName(ctx, softSkill.Name)
			if err != nil || softSkillModel == nil {
				// unknown -> use nil UUID
				logWithTS("[WARN] processJobExtraction: unknown soft skill '%s' (extraction %s) -> using nil UUID", softSkill.Name, extraction.ID)
				continue
			} else {
				// fallback
				softSkillModel, err = wp.skillRepo.GetByCanonicalName(ctx, softSkill.Evidence) // not ideal
				if err != nil || softSkillModel == nil {
					// unknown -> use nil UUID
					logWithTS("[WARN] processJobExtraction: unknown soft skill (by evidence) '%s' (extraction %s) -> using nil UUID", softSkill.Evidence, extraction.ID)
					continue
				} else {
					softSkillID := softSkillModel.ID
					logWithTS("[DEBUG] processJobExtraction: found soft skill '%s' (ID=%s)", softSkill.Name, softSkillID)

					// Create job soft skill
					jobSoftSkill := &models.JobSoftSkill{
						ID:          uuid.New(),
						JobID:       jobID,
						SoftSkillID: softSkillID,
						Proficiency: "",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}

					if _, err := wp.jobSoftSkillRepo.Insert(ctx, jobSoftSkill); err != nil {
						logWithTS("[ERROR] processJobExtraction: failed to insert job soft skill for soft skill ID %s: %v", softSkillID, err)
						continue
					}
					logWithTS("[DEBUG] processJobExtraction: inserted job soft skill for soft skill ID=%s", softSkillID)
				}
			}
		} else if softSkill.Evidence != "" {
			// fallback to evidence
			softSkillModel, err := wp.skillRepo.GetByCanonicalName(ctx, softSkill.Evidence)
			if err != nil || softSkillModel == nil {
				// unknown -> use nil UUID
				logWithTS("[WARN] processJobExtraction: unknown soft skill (by evidence) '%s' (extraction %s) -> using nil UUID", softSkill.Evidence, extraction.ID)
				continue
			} else {
				softSkillID := softSkillModel.ID
				logWithTS("[DEBUG] processJobExtraction: found soft skill (by evidence) '%s' (ID=%s)", softSkill.Evidence, softSkillID)

				// Create job soft skill
				jobSoftSkill := &models.JobSoftSkill{
					ID:          uuid.New(),
					JobID:       jobID,
					SoftSkillID: softSkillID,
					Proficiency: "",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}

				if _, err := wp.jobSoftSkillRepo.Insert(ctx, jobSoftSkill); err != nil {
					logWithTS("[ERROR] processJobExtraction: failed to insert job soft skill for soft skill ID %s: %v", softSkillID, err)
					continue
				}
				logWithTS("[DEBUG] processJobExtraction: inserted job soft skill for soft skill ID=%s", softSkillID)
			}
		}
	} */

	// Mark the extraction as processed
	if err := wp.jobRepo.UpdateMDEProcessingStatus(ctx, extraction.JobID, true, nil, nil); err != nil {
		return fmt.Errorf("failed to update extraction processing status: %w", err)
	}
	logWithTS("[INFO] processJobExtraction: extraction %s marked as processed", extraction.ID)

	return nil
}
