package processing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
	"passo/backend/internal/plugins/market-data-engine/repository"
)

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
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		log.Print("[INFO] MDE worker pool already running")
		return nil
	}

	wp.running = true

	// Start workers
	for i := 0; i < wp.config.MaxConcurrentJobs; i++ {
		go wp.worker(ctx, i)
		log.Printf("[INFO] Started MDE worker %d", i)
	}

	log.Print("[INFO] MDE worker pool started")
	return nil
}

// Stop stops the worker pool processing
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		log.Print("[INFO] MDE worker pool not running")
		return
	}

	close(wp.stopChan)
	wp.running = false
	log.Print("[INFO] MDE worker pool stopping...")
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
func (wp *WorkerPool) worker(ctx context.Context, workerID int) {
	log.Printf("[INFO] workerID=%d: started", workerID)
	defer log.Printf("[INFO] workerID=%d: stopped", workerID)

	for {
		select {
		case <-wp.stopChan:
			log.Printf("[INFO] workerID=%d: received stop signal", workerID)
			return
		default:
			// Process a batch of job extractions
			if count, err := wp.processBatch(ctx, wp.config.BatchSize); err != nil {
				// Log error but continue processing
				log.Printf("[ERROR] workerID=%d: error processing batch: %v", workerID, err)
				continue
			} else {
				// Optionally log count for debugging
				if count > 0 {
					log.Printf("[INFO] workerID=%d: processed %d extractions in batch", workerID, count)
				}
			}

			// Wait before processing the next batch
			select {
			case <-wp.stopChan:
				log.Printf("[INFO] workerID=%d: received stop signal during wait", workerID)
				return
			case <-time.After(wp.config.JobPollInterval):
				// Continue to next iteration
			}
		}
	}
}

// processBatch processes a batch of job extractions
func (wp *WorkerPool) processBatch(ctx context.Context, batchSize int) (int, error) {
	log.Printf("[DEBUG] processBatch: starting batch (size=%d)", batchSize)
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
		log.Printf("[ERROR] processBatch: failed to insert normalization run: %v", err)
		return 0, err
	}

	// Mark the run as completed when we're done
	defer func() {
		completedAt := time.Now().Format(time.RFC3339)
		if err := wp.normalizationRunRepo.UpdateStatus(ctx, runID, "completed", &completedAt, nil); err != nil {
			log.Printf("[ERROR] processBatch: failed to update normalization run status: %v", err)
		}
		log.Printf("[DEBUG] processBatch: normalization run %s completed", runID)
	}()

	// Get pending job extractions using FOR UPDATE SKIP LOCKED for distributed locking
	rows, err := wp.db.Query(ctx, `
		SELECT id, job_id, raw_response, created_at
		FROM job_extractions
		WHERE mde_processed = FALSE
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		$1
	`, batchSize)
	if err != nil {
		log.Printf("[ERROR] processBatch: failed to query job_extractions: %v", err)
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
			log.Printf("[ERROR] processBatch: failed to scan job extraction: %v", err)
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
			continue
		}

		log.Printf("[DEBUG] processBatch: processing extraction %s (jobID=%s)", jobExtraction.ID, jobExtraction.JobID)

		// Process this job extraction
		if err := wp.processJobExtraction(ctx, &jobExtraction); err != nil {
			// Update the run with error and continue
			log.Printf("[ERROR] processBatch: failed to process job extraction %s: %v", jobExtraction.ID, err)
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
			continue
		}

		log.Printf("[DEBUG] processBatch: successfully processed extraction %s", jobExtraction.ID)
		processedCount++
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] processBatch: error after iterating rows: %v", err)
		return processedCount, err
	}

	// Update the run with final progress
	if err := wp.normalizationRunRepo.UpdateProgress(ctx, runID, processedCount, processedCount, 0); err != nil {
		log.Printf("[ERROR] processBatch: failed to update progress: %v", err)
	}

	log.Printf("[INFO] processBatch: completed batch, processed=%d", processedCount)
	return processedCount, nil
}

// processJobExtraction processes a single job extraction
func (wp *WorkerPool) processJobExtraction(ctx context.Context, extraction *models.JobExtraction) error {
	// Unmarshal raw_response into ExtractedJobData
	var extracted models.ExtractedJobData
	if err := json.Unmarshal(extraction.RawResponse, &extracted); err != nil {
		return fmt.Errorf("failed to unmarshal raw_response: %w", err)
	}

	log := log.Printf // shortcut for logging

	// Normalize the extracted job data
	normalizedJob := wp.normalizer.NormalizeJob(extracted)

	// Start a transaction
	tx, err := wp.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log("[ERROR] processJobExtraction: rollback failed after error: %v", rbErr)
			}
		} else {
			if cmErr := tx.Commit(ctx); cmErr != nil {
				log("[ERROR] processJobExtraction: commit failed: %v", cmErr)
				err = cmErr
			}
		}
	}()

	// Create or get the company
	var companyID uuid.UUID
	companyValue, _ := normalizedJob.Company.Value.(string)
	if companyValue != "" {
		company, err := wp.companyRepo.GetByCanonicalName(ctx, companyValue)
		if err != nil || company == nil {
			// unknown -> review queue
			log("[INFO] processJobExtraction: unknown company '%s' (extraction %s) -> review queue", companyValue, extraction.ID)
			wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
				ID:           uuid.New(),
				EntityType:   "company",
				EntityID:     uuid.Nil,
				FieldName:    "company",
				FieldValue:   companyValue,
				Priority:     0,
				Attempts:     0,
				Status:       "pending_review",
				ErrorMessage: nil,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
			// continue without creating a company placeholder
		} else {
			companyID = company.ID
			log("[DEBUG] processJobExtraction: found company '%s' (ID=%s)", companyValue, companyID)
		}
	}

	// Create or get the location
	var locationID uuid.UUID
	if normalizedJob.Location.City != "" {
		location, err := wp.locationRepo.GetByCityStateCountry(ctx, normalizedJob.Location.City, normalizedJob.Location.State, normalizedJob.Location.Country)

		if err != nil || location == nil {
			// unknown -> review queue
			locStr := normalizedJob.Location.City + ", " + normalizedJob.Location.State + ", " + normalizedJob.Location.Country
			log("[INFO] processJobExtraction: unknown location '%s' (extraction %s) -> review queue", locStr, extraction.ID)
			wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
				ID:           uuid.New(),
				EntityType:   "location",
				EntityID:     uuid.Nil,
				FieldName:    "location",
				FieldValue:   locStr,
				Priority:     0,
				Attempts:     0,
				Status:       "pending_review",
				ErrorMessage: nil,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		} else {
			locationID = location.ID
			log("[DEBUG] processJobExtraction: found location '%s' (ID=%s)", locationID)
		}
	}

	// Create or get the job title
	var titleID uuid.UUID
	if normalizedJob.Title.Value != "" {
		title, err := wp.jobTitleRepo.GetByCanonicalName(ctx, normalizedJob.Title.Value)
		if err != nil || title == nil {
			// unknown -> review queue
			log("[INFO] processJobExtraction: unknown title '%s' (extraction %s) -> review queue", normalizedJob.Title.Value, extraction.ID)
			wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
				ID:           uuid.New(),
				EntityType:   "title",
				EntityID:     uuid.Nil,
				FieldName:    "title",
				FieldValue:   normalizedJob.Title.Value,
				Priority:     0,
				Attempts:     0,
				Status:       "pending_review",
				ErrorMessage: nil,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		} else {
			titleID = title.ID
			log("[DEBUG] processJobExtraction: found title '%s' (ID=%s)", normalizedJob.Title.Value, titleID)
		}
	}

	// Create or get the seniority level
	var seniorityID uuid.UUID
	if normalizedJob.Seniority.Value != "" {
		seniority, err := wp.seniorityLevelRepo.GetByCanonicalName(ctx, normalizedJob.Seniority.Value)
		if err != nil || seniority == nil {
			// unknown -> review queue
			log("[INFO] processJobExtraction: unknown seniority '%s' (extraction %s) -> review queue", normalizedJob.Seniority.Value, extraction.ID)
			wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
				ID:           uuid.New(),
				EntityType:   "seniority",
				EntityID:     uuid.Nil,
				FieldName:    "seniority",
				FieldValue:   normalizedJob.Seniority.Value,
				Priority:     0,
				Attempts:     0,
				Status:       "pending_review",
				ErrorMessage: nil,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		} else {
			seniorityID = seniority.ID
			log("[DEBUG] processJobExtraction: found seniority '%s' (ID=%s)", normalizedJob.Seniority.Value, seniorityID)
		}
	}

	// Create or get the employment type
	var employmentTypeID uuid.UUID
	if normalizedJob.EmploymentType.Value != "" {
		employmentType, err := wp.employmentTypeRepo.GetByCanonicalName(ctx, normalizedJob.EmploymentType.Value)
		if err != nil || employmentType == nil {
			// unknown -> review queue
			log("[INFO] processJobExtraction: unknown employment_type '%s' (extraction %s) -> review queue", normalizedJob.EmploymentType.Value, extraction.ID)
			wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
				ID:           uuid.New(),
				EntityType:   "employment_type",
				EntityID:     uuid.Nil,
				FieldName:    "employment_type",
				FieldValue:   normalizedJob.EmploymentType.Value,
				Priority:     0,
				Attempts:     0,
				Status:       "pending_review",
				ErrorMessage: nil,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		} else {
			employmentTypeID = employmentType.ID
			log("[DEBUG] processJobExtraction: found employment_type '%s' (ID=%s)", normalizedJob.EmploymentType.Value, employmentTypeID)
		}
	}

	// Create or get the work mode
	var workModeID uuid.UUID
	if normalizedJob.WorkMode.Value != "" {
		workMode, err := wp.workModeRepo.GetByCanonicalName(ctx, normalizedJob.WorkMode.Value)
		if err != nil || workMode == nil {
			// unknown -> review queue
			log("[INFO] processJobExtraction: unknown work_mode '%s' (extraction %s) -> review queue", normalizedJob.WorkMode.Value, extraction.ID)
			wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
				ID:           uuid.New(),
				EntityType:   "work_mode",
				EntityID:     uuid.Nil,
				FieldName:    "work_mode",
				FieldValue:   normalizedJob.WorkMode.Value,
				Priority:     0,
				Attempts:     0,
				Status:       "pending_review",
				ErrorMessage: nil,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		} else {
			workModeID = workMode.ID
			log("[DEBUG] processJobExtraction: found work_mode '%s' (ID=%s)", normalizedJob.WorkMode.Value, workModeID)
		}
	}

	// Create the job
	job := &models.Job{
		ID:              uuid.New(),
		ExternalID:      extraction.JobID.String(),
		Source:          "unknown",
		SourceURL:       "",
		RawContent:      extraction.RawContent,
		Language:        extraction.Language,
		RetrievedAt:     time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MDEProcessed:    true,
		MDEProcessedAt:  func() *string { s := time.Now().Format(time.RFC3339); return &s }(),
		MDEErrorMessage: nil,
		// normalization_version will be set from config or taxonomy version later
	}

	jobID, err := wp.jobRepo.Insert(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}
	log("[DEBUG] processJobExtraction: inserted job ID=%s", jobID)

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
			log("[INFO] processJobExtraction: duplicate posting detected (dedup_key=%s) for extraction %s -> skipping persistence", dedupKey, extraction.ID)
			wp.jobRepo.UpdateMDEProcessingStatus(ctx, extraction.JobID, true, nil, nil)
			return nil
		}
		return fmt.Errorf("failed to insert job posting: %w", err)
	}
	log("[DEBUG] processJobExtraction: inserted job posting ID=%s", postingID)

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
	log("[DEBUG] processJobExtraction: inserted job description ID=%s", descID)

	// Create job skills
	for _, skill := range normalizedJob.Skills {
		// Get or create the skill
		var skillModel *models.Skill
		var err error
		if skill.NormalizedName != "" {
			skillModel, err = wp.skillRepo.GetByCanonicalName(ctx, skill.NormalizedName)
			if err != nil || skillModel == nil {
				// unknown -> review queue
				log("[INFO] processJobExtraction: unknown skill (normalized) '%s' (extraction %s) -> review queue", skill.NormalizedName, extraction.ID)
				wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
					ID:           uuid.New(),
					EntityType:   "skill",
					EntityID:     uuid.Nil,
					FieldName:    "skill",
					FieldValue:   skill.NormalizedName,
					Priority:     0,
					Attempts:     0,
					Status:       "pending_review",
					ErrorMessage: nil,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				})
				continue
			}
		} else {
			// fallback to raw name
			skillModel, err = wp.skillRepo.GetByCanonicalName(ctx, skill.RawName)
			if err != nil || skillModel == nil {
				// unknown -> review queue
				log("[INFO] processJobExtraction: unknown skill (raw) '%s' (extraction %s) -> review queue", skill.RawName, extraction.ID)
				wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
					ID: uuid.New(),

					EntityType:   "skill",
					EntityID:     uuid.Nil,
					FieldName:    "skill",
					FieldValue:   skill.RawName,
					Priority:     0,
					Attempts:     0,
					Status:       "pending_review",
					ErrorMessage: nil,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				})
				continue
			}
		}
		skillID := skillModel.ID
		log("[DEBUG] processJobExtraction: found skill '%s' (ID=%s)", skill.NormalizedName, skillID)

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
			log("[ERROR] processJobExtraction: failed to insert job skill for skill ID %s: %v", skillID, err)
			continue
		}
		log("[DEBUG] processJobExtraction: inserted job skill for skill ID=%s", skillID)
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
			log("[ERROR] processJobExtraction: failed to insert job capability for capability ID %s: %v", capabilityID, err)
			continue
		}
		log("[DEBUG] processJobExtraction: inserted job capability for capability ID=%s", capabilityID)
	}

	// Create job soft skills (stub implementation)
	for _, softSkill := range normalizedJob.SoftSkills {
		var softSkillModel *models.Skill
		var err error
		if softSkill.Name != "" {
			softSkillModel, err = wp.skillRepo.GetByCanonicalName(ctx, softSkill.Name)
			if err != nil || softSkillModel == nil {
				// unknown -> review queue
				log("[INFO] processJobExtraction: unknown soft skill '%s' (extraction %s) -> review queue", softSkill.Name, extraction.ID)
				wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
					ID:           uuid.New(),
					EntityType:   "soft_skill",
					EntityID:     uuid.Nil,
					FieldName:    "soft_skill",
					FieldValue:   softSkill.Name,
					Priority:     0,
					Attempts:     0,
					Status:       "pending_review",
					ErrorMessage: nil,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				})
				continue
			}
		} else {
			// fallback
			softSkillModel, err = wp.skillRepo.GetByCanonicalName(ctx, softSkill.Evidence) // not ideal
			if err != nil || softSkillModel == nil {
				// unknown -> review queue
				log("[INFO] processJobExtraction: unknown soft skill (by evidence) '%s' (extraction %s) -> review queue", softSkill.Evidence, extraction.ID)
				wp.entityResolutionQueueRepo.Insert(ctx, &models.EntityResolutionQueueItem{
					ID:           uuid.New(),
					EntityType:   "soft_skill",
					EntityID:     uuid.Nil,
					FieldName:    "soft_skill",
					FieldValue:   softSkill.Evidence,
					Priority:     0,
					Attempts:     0,
					Status:       "pending_review",
					ErrorMessage: nil,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				})
				continue
			}
		}
		softSkillID := softSkillModel.ID
		log("[DEBUG] processJobExtraction: found soft skill '%s' (ID=%s)", softSkill.Name, softSkillID)

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
			log("[ERROR] processJobExtraction: failed to insert job soft skill for soft skill ID %s: %v", softSkillID, err)
			continue
		}
		log("[DEBUG] processJobExtraction: inserted job soft skill for soft skill ID=%s", softSkillID)
	}

	// Mark the extraction as processed
	if err := wp.jobRepo.UpdateMDEProcessingStatus(ctx, extraction.JobID, true, nil, nil); err != nil {
		return fmt.Errorf("failed to update extraction processing status: %w", err)
	}
	log("[INFO] processJobExtraction: extraction %s marked as processed", extraction.ID)

	return nil
}
