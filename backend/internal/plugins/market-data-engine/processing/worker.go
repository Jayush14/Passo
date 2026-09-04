package processing

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
	"passo/backend/internal/plugins/market-data-engine/repository"
)

// WorkerPool manages a pool of workers that process job extractions
type WorkerPool struct {
	db                   *pgxpool.Pool
	extractor            *Extractor
	normalizer           *Normalizer
	jobRepo              *repository.JobRepository
	jobPostingRepo       *repository.JobPostingRepository
	jobDescriptionRepo   *repository.JobDescriptionRepository
	jobSkillRepo         *repository.JobSkillRepository
	jobCapabilityRepo    *repository.JobCapabilityRepository
	jobDomainRepo        *repository.JobDomainRepository
	jobSoftSkillRepo     *repository.JobSoftSkillRepository
	companyRepo          *repository.CompanyRepository
	locationRepo         *repository.LocationRepository
	skillRepo            *repository.SkillRepository
	skillAliasRepo       *repository.SkillAliasRepository
	jobTitleRepo         *repository.JobTitleRepository
	seniorityLevelRepo   *repository.SeniorityLevelRepository
	jobFamilyRepo        *repository.JobFamilyRepository
	employmentTypeRepo   *repository.EmploymentTypeRepository
	workModeRepo         *repository.WorkModeRepository
	capabilityRepo       *repository.CapabilityRepository
	domainRepo           *repository.DomainRepository
	normalizationRunRepo *repository.NormalizationRunRepository
	failedExtractionRepo *repository.FailedExtractionRepository

	config   WorkerConfig
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
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
		db:                   db,
		extractor:            NewExtractor(),
		normalizer:           NewNormalizer(db),
		jobRepo:              repository.NewJobRepository(db),
		jobPostingRepo:       repository.NewJobPostingRepository(db),
		jobDescriptionRepo:   repository.NewJobDescriptionRepository(db),
		jobSkillRepo:         repository.NewJobSkillRepository(db),
		jobCapabilityRepo:    repository.NewJobCapabilityRepository(db),
		jobDomainRepo:        repository.NewJobDomainRepository(db),
		jobSoftSkillRepo:     repository.NewJobSoftSkillRepository(db),
		companyRepo:          repository.NewCompanyRepository(db),
		locationRepo:         repository.NewLocationRepository(db),
		skillRepo:            repository.NewSkillRepository(db),
		skillAliasRepo:       repository.NewSkillAliasRepository(db),
		jobTitleRepo:         repository.NewJobTitleRepository(db),
		seniorityLevelRepo:   repository.NewSeniorityLevelRepository(db),
		jobFamilyRepo:        repository.NewJobFamilyRepository(db),
		employmentTypeRepo:   repository.NewEmploymentTypeRepository(db),
		workModeRepo:         repository.NewWorkModeRepository(db),
		capabilityRepo:       repository.NewCapabilityRepository(db),
		domainRepo:           repository.NewDomainRepository(db),
		normalizationRunRepo: repository.NewNormalizationRunRepository(db),
		failedExtractionRepo: repository.NewFailedExtractionRepository(db),
		config:               config,
		running:              false,
		stopChan:             make(chan struct{}),
	}
}

// Start begins the worker pool processing
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return nil
	}

	wp.running = true

	// Start workers
	for i := 0; i < wp.config.MaxConcurrentJobs; i++ {
		go wp.worker(ctx, i)
	}

	return nil
}

// Stop stops the worker pool processing
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return
	}

	close(wp.stopChan)
	wp.running = false
}

// IsRunning reports whether the worker pool is currently processing jobs.
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return wp.running
}

// worker is the main worker function that processes job extractions
func (wp *WorkerPool) worker(ctx context.Context, workerID int) {
	for {
		select {
		case <-wp.stopChan:
			return
		default:
			// Process a batch of job extractions
			if err := wp.processBatch(ctx); err != nil {
				// Log error but continue processing
				// In a real implementation, we would use proper logging
				continue
			}

			// Wait before processing the next batch
			select {
			case <-wp.stopChan:
				return
			case <-time.After(wp.config.JobPollInterval):
				// Continue to next iteration
			}
		}
	}
}

// processBatch processes a batch of job extractions
func (wp *WorkerPool) processBatch(ctx context.Context) error {
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
		return err
	}

	// Mark the run as completed when we're done
	defer func() {
		completedAt := time.Now().Format(time.RFC3339)
		wp.normalizationRunRepo.UpdateStatus(ctx, runID, "completed", &completedAt, nil)
	}()

	// Get pending job extractions using FOR UPDATE SKIP LOCKED for distributed locking
	rows, err := wp.db.Query(ctx, `
		SELECT id, job_id, raw_content, language, created_at
		FROM job_extractions
		WHERE mde_processed = FALSE
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`, wp.config.BatchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	var processedCount int
	for rows.Next() {
		var jobExtraction models.JobExtraction
		if err := rows.Scan(
			&jobExtraction.ID,
			&jobExtraction.JobID,
			&jobExtraction.RawContent,
			&jobExtraction.Language,
			&jobExtraction.CreatedAt,
		); err != nil {
			// Update the run with error and continue
			wp.normalizationRunRepo.UpdateProgress(ctx, runID, processedCount, 0, 1)
			wp.failedExtractionRepo.Insert(ctx, &models.FailedExtraction{
				ID:             uuid.New(),
				JobID:          jobExtraction.JobID,
				ExtractionData: jobExtraction.RawContent,
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

		// Process this job extraction
		if err := wp.processJobExtraction(ctx, &jobExtraction); err != nil {
			// Update the run with error and continue
			wp.normalizationRunRepo.UpdateProgress(ctx, runID, processedCount, 0, 1)
			wp.failedExtractionRepo.Insert(ctx, &models.FailedExtraction{
				ID:             uuid.New(),
				JobID:          jobExtraction.JobID,
				ExtractionData: jobExtraction.RawContent,
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

		processedCount++
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Update the run with final progress
	wp.normalizationRunRepo.UpdateProgress(ctx, runID, processedCount, processedCount, 0)

	return nil
}

// processJobExtraction processes a single job extraction
func (wp *WorkerPool) processJobExtraction(ctx context.Context, extraction *models.JobExtraction) error {
	// Extract job data using the LLM extractor
	extractedJob, err := wp.extractor.ExtractJobData(extraction.RawContent)
	if err != nil {
		return err
	}

	// Normalize the extracted job data
	normalizedJob := wp.normalizer.NormalizeJob(*extractedJob)

	// Start a transaction
	tx, err := wp.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	// Create or get the company
	var companyID uuid.UUID
	companyValue, _ := normalizedJob.Company.Value.(string)
	if companyValue != "" {
		company, err := wp.companyRepo.GetByCanonicalName(ctx, companyValue)
		if err != nil {
			// Create new company
			company.ID = uuid.New()
			company.CanonicalName = companyValue
			company.NormalizedName = companyValue
			company.Website = ""               // Would be extracted in a real implementation
			company.Industry = ""              // Would be extracted in a real implementation
			company.Size = ""                  // Would be extracted in a real implementation
			company.TaxonomyVersion = uuid.Nil // Would be looked up in a real implementation
			company.CreatedAt = time.Now()
			company.UpdatedAt = time.Now()

			companyID, err = wp.companyRepo.Insert(ctx, company)
			if err != nil {
				return err
			}
		} else {
			companyID = company.ID

			// Update company if needed
			// Note: Website, Industry, Size would come from extracted data in a real implementation
			company.UpdatedAt = time.Now()

			if err := wp.companyRepo.Update(ctx, company); err != nil {
				return err
			}
		}
	}

	// Create or get the location
	var locationID uuid.UUID
	if normalizedJob.Location.City != "" {
		// For now, always create a new location to avoid complications with lookup
		// TODO: Implement proper lookup by city/state/country to avoid duplicates
		location := &models.Location{
			ID:              uuid.New(),
			City:            normalizedJob.Location.City,
			State:           normalizedJob.Location.State,
			Country:         normalizedJob.Location.Country,
			CountryCode:     "",       // Would be extracted in a real implementation
			Lat:             0,        // Would be extracted in a real implementation
			Lon:             0,        // Would be extracted in a real implementation
			TaxonomyVersion: uuid.Nil, // Would be looked up in a real implementation
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		locationID, err = wp.locationRepo.Insert(ctx, location)
		if err != nil {
			return err
		}
	}

	// Create or get the job title
	titleValue := normalizedJob.Title.Value
	if titleValue != "" {
		// For now, always create a new job title to avoid complications with lookup
		// TODO: Implement proper lookup by title to avoid duplicates
		jobTitle := &models.JobTitle{
			ID:              uuid.New(),
			Title:           titleValue,
			NormalizedTitle: titleValue,
			TaxonomyVersion: uuid.Nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		_, err = wp.jobTitleRepo.Insert(ctx, jobTitle)
		if err != nil {
			return err
		}
	}

	// Create or get the seniority level
	seniorityValue := normalizedJob.Seniority.Value
	if seniorityValue != "" {
		// For now, always create a new seniority level to avoid complications with lookup
		// TODO: Implement proper lookup by level to avoid duplicates
		seniorityLevel := &models.SeniorityLevel{
			ID:              uuid.New(),
			Level:           seniorityValue,
			NormalizedLevel: seniorityValue,
			TaxonomyVersion: uuid.Nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		_, err = wp.seniorityLevelRepo.Insert(ctx, seniorityLevel)
		if err != nil {
			return err
		}
	}

	// Create or get the job family
	// TODO: JobFamily field is missing from NormalizedJobData struct
	// var jobFamilyID uuid.UUID
	// if normalizedJob.JobFamily.Family != "" {
	// 	jobFamily, err := wp.jobFamilyRepo.GetByID(ctx, normalizedJob.JobFamily.ID)
	// 	if err != nil {
	// 		// Create new job family
	// 		jobFamily.ID = uuid.New()
	// 		jobFamily.Family = normalizedJob.JobFamily.Family
	// 		jobFamily.NormalizedFamily = normalizedJob.JobFamily.NormalizedFamily
	// 		jobFamily.TaxonomyVersion = normalizedJob.JobFamily.TaxonomyVersion
	// 		jobFamily.CreatedAt = time.Now()
	// 		jobFamily.UpdatedAt = time.Now()
	//
	// 		jobFamilyID, err = wp.jobFamilyRepo.Insert(ctx, jobFamily)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	} else {
	// 		jobFamilyID = jobFamily.ID
	//
	// 		// Update job family if needed
	// 		jobFamily.Family = normalizedJob.JobFamily.Family
	// 		jobFamily.NormalizedFamily = normalizedJob.JobFamily.NormalizedFamily
	// 		jobFamily.TaxonomyVersion = normalizedJob.JobFamily.TaxonomyVersion
	// 		jobFamily.UpdatedAt = time.Now()
	//
	// 		if err := wp.jobFamilyRepo.Update(ctx, jobFamily); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	// Create or get the employment type
	var employmentTypeID uuid.UUID
	employmentTypeValue := normalizedJob.EmploymentType.Value
	if employmentTypeValue != "" {
		// For now, always create a new employment type to avoid complications with lookup
		// TODO: Implement proper lookup by type to avoid duplicates
		employmentType := &models.EmploymentType{
			ID:              uuid.New(),
			Type:            employmentTypeValue,
			NormalizedType:  employmentTypeValue,
			TaxonomyVersion: uuid.Nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		employmentTypeID, err = wp.employmentTypeRepo.Insert(ctx, employmentType)
		if err != nil {
			return err
		}
	}

	// Create or get the work mode
	var workModeID uuid.UUID
	workModeValue := normalizedJob.WorkMode.Value
	if workModeValue != "" {
		// For now, always create a new work mode to avoid complications with lookup
		// TODO: Implement proper lookup by mode to avoid duplicates
		workMode := &models.WorkMode{
			ID:              uuid.New(),
			Mode:            workModeValue,
			NormalizedMode:  workModeValue,
			TaxonomyVersion: uuid.Nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		workModeID, err = wp.workModeRepo.Insert(ctx, workMode)
		if err != nil {
			return err
		}
	}

	// Create the job
	job := &models.Job{
		ID:              uuid.New(),
		ExternalID:      extraction.JobID.String(),
		Source:          "unknown",
		SourceURL:       "",
		RawContent:      extraction.RawContent,
		DedupKey:        "", // Would be calculated from content in a real implementation
		Language:        extraction.Language,
		RetrievedAt:     time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		MDEProcessed:    true,
		MDEProcessedAt:  func() *string { s := time.Now().Format(time.RFC3339); return &s }(),
		MDEErrorMessage: nil,
	}

	_, err = wp.jobRepo.Insert(ctx, job)
	if err != nil {
		return err
	}

	// Create the job posting
	jobPosting := &models.JobPosting{
		ID:               uuid.New(),
		JobID:            job.ID,
		Title:            normalizedJob.Title.Value,
		CompanyID:        companyID,
		LocationID:       locationID,
		EmploymentTypeID: employmentTypeID,
		WorkModeID:       workModeID,
		SalaryMin:        0, // Would be extracted from normalized data in a real implementation
		SalaryMax:        0, // Would be extracted from normalized data in a real implementation
		SalaryCurrency:   "USD",
		SalaryPeriod:     "yearly",
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

	_, err = wp.jobPostingRepo.Insert(ctx, jobPosting)
	if err != nil {
		return err
	}

	// Create the job description
	jobDescription := &models.JobDescription{
		ID:          uuid.New(),
		JobID:       jobPosting.ID,
		Title:       jobPosting.Title,
		Description: jobPosting.Description,
	}

	_, err = wp.jobDescriptionRepo.Insert(ctx, jobDescription)
	if err != nil {
		return err
	}

	// Create job skills
	for _, skill := range normalizedJob.Skills {
		// Get or create the skill
		var skillID uuid.UUID
		dbSkill, err := wp.skillRepo.GetByCanonicalName(ctx, skill.NormalizedName)
		if err != nil {
			// Create new skill
			dbSkill = &models.Skill{
				ID:              uuid.New(),
				CanonicalName:   skill.NormalizedName,
				NormalizedName:  skill.NormalizedName,
				SkillTypeID:     uuid.Nil, // Would be looked up in a real implementation
				CategoryID:      uuid.Nil, // Would be looked up in a real implementation
				ParentSkillID:   nil,
				Description:     "", // Would be extracted in a real implementation
				Status:          "active",
				Version:         1,
				TaxonomyVersion: uuid.Nil, // Would be looked up in a real implementation
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			skillID, err = wp.skillRepo.Insert(ctx, dbSkill)
			if err != nil {
				return err
			}
		} else {
			skillID = dbSkill.ID

			// Update skill if needed
			// Note: Description, Version, TaxonomyVersion would come from extracted data in a real implementation
			dbSkill.UpdatedAt = time.Now()

			if err := wp.skillRepo.Update(ctx, dbSkill); err != nil {
				return err
			}
		}

		// Create job skill
		jobSkill := &models.JobSkill{
			ID:           uuid.New(),
			JobPostingID: jobPosting.ID,
			SkillID:      skillID,
			SkillLevel:   skill.RequirementLevel,
			Importance:   skill.Confidence,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err = wp.jobSkillRepo.Insert(ctx, jobSkill)
		if err != nil {
			return err
		}
	}

	// Create job capabilities (stub implementation)
	for _, skill := range normalizedJob.Skills {
		if skill.Capability == nil {
			continue
		}
		capability := skill.Capability
		// Get or create the capability
		var capabilityID uuid.UUID
		dbCapability, err := wp.capabilityRepo.GetByName(ctx, capability.CanonicalName)
		if err != nil {
			// Create new capability
			dbCapability = &models.Capability{
				ID:              uuid.New(),
				Name:            capability.CanonicalName,
				Category:        capability.Category,
				Description:     capability.Description,
				TaxonomyVersion: uuid.Nil,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			capabilityID, err = wp.capabilityRepo.Insert(ctx, dbCapability)
			if err != nil {
				return err
			}
		} else {
			capabilityID = dbCapability.ID

			// Update capability if needed
			dbCapability.Name = capability.CanonicalName
			dbCapability.Category = capability.Category
			dbCapability.Description = capability.Description
			dbCapability.UpdatedAt = time.Now()

			if err := wp.capabilityRepo.Update(ctx, dbCapability); err != nil {
				return err
			}
		}

		// Create job capability
		jobCapability := &models.JobCapability{
			ID:           uuid.New(),
			JobPostingID: jobPosting.ID,
			CapabilityID: capabilityID,
			Level:        "",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err = wp.jobCapabilityRepo.Insert(ctx, jobCapability)
		if err != nil {
			return err
		}
	}

	// Create job soft skills (stub implementation)
	for _, softSkill := range normalizedJob.SoftSkills {
		// Get or create the soft skill
		var softSkillID uuid.UUID
		dbSoftSkill, err := wp.skillRepo.GetByCanonicalName(ctx, softSkill.Name)
		if err != nil {
			// Create new soft skill
			dbSoftSkill = &models.Skill{
				ID:              uuid.New(),
				CanonicalName:   softSkill.Name,
				NormalizedName:  softSkill.Name,
				SkillTypeID:     uuid.Nil, // Would be looked up in a real implementation
				CategoryID:      uuid.Nil, // Would be looked up in a real implementation
				ParentSkillID:   nil,
				Description:     softSkill.Evidence,
				Status:          "active",
				Version:         1,
				TaxonomyVersion: uuid.Nil,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			softSkillID, err = wp.skillRepo.Insert(ctx, dbSoftSkill)
			if err != nil {
				return err
			}
		} else {
			softSkillID = dbSoftSkill.ID

			// Update soft skill if needed
			dbSoftSkill.Description = softSkill.Evidence
			dbSoftSkill.UpdatedAt = time.Now()

			if err := wp.skillRepo.Update(ctx, dbSoftSkill); err != nil {
				return err
			}
		}

		// Create job soft skill
		jobSoftSkill := &models.JobSoftSkill{
			ID:           uuid.New(),
			JobPostingID: jobPosting.ID,
			SoftSkillID:  softSkillID,
			Proficiency:  "",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if _, err := wp.jobSoftSkillRepo.Insert(ctx, jobSoftSkill); err != nil {
			return err
		}
	}

	// Mark the extraction as processed
	if err := wp.jobRepo.UpdateMDEProcessingStatus(ctx, extraction.JobID, true, nil, nil); err != nil {
		return err
	}

	return nil
}
