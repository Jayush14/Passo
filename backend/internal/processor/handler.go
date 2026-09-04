package processor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"passo/backend/internal/ai/provider"
	"passo/backend/internal/apikey"
	"passo/backend/internal/db"
	"passo/backend/internal/prompts"

	"github.com/google/uuid"
)

// db is the shared DB instance for this package.
var dbInst *db.DB
var batchCancels sync.Map

// SetDB sets the database instance for the processor handlers.
func SetDB(database *db.DB) {
	dbInst = database
}

// ProcessJobRequest represents the incoming job processing request.
type ProcessJobRequest struct {
	JobDescription string `json:"job_description"`
	// Optional: allow specifying provider/model, otherwise use defaults
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

// ProcessJobResponse represents the job processing response.
type ProcessJobResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	JobID   string      `json:"job_id,omitempty"` // Could be a hash or timestamp
	Data    interface{} `json:"data,omitempty"`   // The processed job data
}

// HandleProcessJob processes a job description using AI provider directly
// to extract and normalize job data.
// It expects a JSON body with a "job_description" field, optional "provider", and optional "model".
func HandleProcessJob(w http.ResponseWriter, r *http.Request) {
	// Decode request body
	var req ProcessJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.JobDescription == "" {
		http.Error(w, "job_description is required", http.StatusBadRequest)
		return
	}

	// Set defaults for provider and model if not specified
	if req.Provider == "" {
		req.Provider = "gemini" // default provider
	}
	if req.Model == "" {
		req.Model = "gemini-2.0-flash" // default model
	}

	// Retrieve user ID from context (set by auth middleware).
	userIDVal := r.Context().Value("userID")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get encrypted API key from database for the specific provider.
	if dbInst == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}
	encryptedKey, err := dbInst.GetAPIKey(r.Context(), userID, req.Provider)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "API key not found for user and provider", http.StatusUnauthorized)
			return
		}
		http.Error(w, "could not retrieve API key", http.StatusInternalServerError)
		return
	}

	// Decrypt the API key.
	apiKey, err := apikey.Decrypt(encryptedKey)
	if err != nil {
		http.Error(w, "decryption failed", http.StatusInternalServerError)
		return
	}

	// Create AI provider
	gen, err := provider.Factory(r.Context(), req.Provider, apiKey, req.Model)
	if err != nil {
		http.Error(w, fmt.Sprintf("unsupported provider %s: %v", req.Provider, err), http.StatusBadGateway)
		return
	}

	// Build prompt for job description extraction using template
	promptData := map[string]string{
		"JobDescription": req.JobDescription,
	}
	prompt, err := prompts.RenderTemplate("job_extraction", promptData)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to render job extraction template: %v", err), http.StatusInternalServerError)
		return
	}

	// TEMP TEST MODE: fake the LLM response to delay and return a valid JSON payload
	// without calling the provider again.

	generatedText, err := gen.GenerateText(r.Context(), prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate text: %v", err), http.StatusInternalServerError)
		return
	}

	// Use a single stable job ID for this extraction lifecycle.
	jobID := uuid.New().String()

	// Save the raw response to the database in the background.
	go func() {
		// Use background context to avoid request cancellation issues.
		ctx := context.Background()
		_, err := dbInst.ExecContext(ctx, `
			INSERT INTO job_extractions (id, job_id, provider, model, prompt_version, raw_response, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, jobID, jobID, req.Provider, req.Model, "v1", generatedText, time.Now())
		if err != nil {
			// Log the error but don't fail the request.
			fmt.Printf("failed to save job extraction: %v\n", err)
		}
	}()

	// Extract JSON object from the model output (trim any surrounding text)
	jsonStr := strings.TrimSpace(generatedText)
	startIdx := strings.Index(jsonStr, "{")
	endIdx := strings.LastIndex(jsonStr, "}")
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		http.Error(w, "invalid response from provider: no JSON object found", http.StatusInternalServerError)
		return
	}
	jsonStr = jsonStr[startIdx : endIdx+1]

	// Parse the extracted job data
	var extractedJob map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &extractedJob); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse provider response as JSON: %v", err), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := ProcessJobResponse{
		Success: true,
		Message: "Job description processed successfully",
		JobID:   jobID,
		Data:    extractedJob,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}

// BatchProcessJobRequest represents the payload for starting a batch job.
type BatchProcessJobRequest struct {
	Jobs []ProcessJobRequest `json:"jobs"`
}

// batchJob is the internal unit of work sent to workers.
type batchJob struct {
	JobDescription string
	Provider       string
	Model          string
	UserID         int64
	BatchID        string // UUID of the batch
	JobID          string // UUID for this specific job
}

// BatchStatusResponse is the payload returned by GET /batch/{batch_id}.
type BatchStatusResponse struct {
	BatchID        string    `json:"batch_id"`
	UserID         int64     `json:"user_id"`
	TotalJobs      int       `json:"total_jobs"`
	ProcessedOK    int       `json:"processed_ok"`
	ProcessedFail  int       `json:"processed_fail"`
	JobsInProgress int       `json:"jobs_in_progress"`
	JobsWaiting    int       `json:"jobs_waiting"`
	Status         string    `json:"status"` // pending | processing | completed | stopped
	CreatedAt      time.Time `json:"created_at"`
}

// processSingleJob contains the core logic of processing a single job.
// It returns nil on success, or an error if any step fails (including DB insert).
func processSingleJob(ctx context.Context, job batchJob, dbInst *db.DB) (string, error) {
	// Get encrypted API key for the user/provider
	encryptedKey, err := dbInst.GetAPIKey(ctx, job.UserID, job.Provider)
	if err != nil {
		return "", err
	}
	// Decrypt the key
	apiKey, err := apikey.Decrypt(encryptedKey)
	if err != nil {
		return "", err
	}
	// Create AI provider
	gen, err := provider.Factory(ctx, job.Provider, apiKey, job.Model)
	if err != nil {
		return "", err
	}
	// Build prompt from template
	promptData := map[string]string{"JobDescription": job.JobDescription}
	prompt, err := prompts.RenderTemplate("job_extraction", promptData)
	if err != nil {
		return "", err
	}
	// Call the LLM
	generatedText, err := gen.GenerateText(ctx, prompt)
	if err != nil {
		return "", err
	}
	// Extract JSON object from the LLM output to validate
	jsonStr := strings.TrimSpace(generatedText)
	startIdx := strings.Index(jsonStr, "{")
	endIdx := strings.LastIndex(jsonStr, "}")
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return "", fmt.Errorf("no JSON object found in provider response")
	}
	jsonStr = jsonStr[startIdx : endIdx+1]
	// Parse JSON – we don't store it, just validate that it's valid
	var extracted map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &extracted); err != nil {
		return "", fmt.Errorf("failed to parse provider response as JSON: %w", err)
	}
	// Success: return the raw generated text
	return generatedText, nil
}

// saveFailedRaw stores the raw job description when a job fails to be processed.
func saveFailedRaw(ctx context.Context, job batchJob, batchID string, userID int64, dbInst *db.DB) error {
	// Reuse the same stable job ID that was created when the staging row was inserted.
	_, err := dbInst.ExecContext(ctx, `
		INSERT INTO job_failed_raw
			(id, job_id, batch_id, user_id, provider, model, job_description, retry_count)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, job.JobID, job.JobID, batchID, userID, job.Provider, job.Model, job.JobDescription, 0)
	return err
}

// updateBatchProgress increments processed_ok by okDelta and processed_fail by failDelta,
// updateBatchProgress increments processed_ok by okDelta and processed_fail by failDelta,
// then updates status based on the new totals.
// If the batch becomes completed, removes its cancel function.
func updateBatchProgress(ctx context.Context, batchID string, okDelta, failDelta int, dbInst *db.DB) error {
	// Log the delta for debugging
	log.Printf("updateBatchProgress: batchID=%s, okDelta=%d, failDelta=%d", batchID, okDelta, failDelta)

	// Get current values for logging
	var currentOK, currentFail, totalJobs, currentIP, currentW int64
	errQuery := dbInst.QueryRowContext(ctx, `
		SELECT processed_ok, processed_fail, total_jobs, jobs_in_progress, jobs_waiting
		FROM job_batches WHERE batch_id = $1
	`, batchID).Scan(&currentOK, &currentFail, &totalJobs, &currentIP, &currentW)
	if errQuery != nil {
		log.Printf("failed to query current batch state: %v", errQuery)
	} else {
		log.Printf("batch %s: before update - OK=%d, Fail=%d, IP=%d, Wait=%d, Total=%d",
			batchID, currentOK, currentFail, currentIP, currentW, totalJobs)
	}

	_, err := dbInst.ExecContext(ctx, `
		UPDATE job_batches
		SET processed_ok = processed_ok + $1,
		    processed_fail = processed_fail + $2,
		    status = CASE
				WHEN (processed_ok + $1 + processed_fail + $2) = total_jobs THEN 'completed'
				ELSE status
			 END,
		    jobs_in_progress = CASE
				WHEN (processed_ok + $1 + processed_fail + $2) = total_jobs THEN 0
				ELSE jobs_in_progress
			 END,
		    jobs_waiting = CASE
				WHEN (processed_ok + $1 + processed_fail + $2) = total_jobs THEN 0
				ELSE jobs_waiting
			 END,
		    updated_at = now()
		WHERE batch_id = $3
	`, okDelta, failDelta, batchID)
	if err != nil {
		return err
	}

	// Get updated values for logging
	var newOK, newFail, newIP, newW int64
	var newStatus string
	errQuery = dbInst.QueryRowContext(ctx, `
		SELECT processed_ok, processed_fail, jobs_in_progress, jobs_waiting, status
		FROM job_batches WHERE batch_id = $1
	`, batchID).Scan(&newOK, &newFail, &newIP, &newW, &newStatus)
	if errQuery != nil {
		log.Printf("failed to query updated batch state: %v", errQuery)
	} else {
		log.Printf("batch %s: after update - OK=%d, Fail=%d, IP=%d, Wait=%d, Status=%s",
			batchID, newOK, newFail, newIP, newW, newStatus)
	}

	// If batch completed, remove cancel function to clean up
	var status string
	err = dbInst.QueryRowContext(ctx, "SELECT status FROM job_batches WHERE batch_id = $1", batchID).Scan(&status)
	if err == nil && status == "completed" {
		log.Printf("batch %s: batch completed, removing cancel function", batchID)
		batchCancels.Delete(batchID)
	}
	return nil
}

// updateBatchJobCounts updates the in-progress and waiting job counts
func updateBatchJobCounts(ctx context.Context, batchID string, inProgressDelta, waitingDelta int, dbInst *db.DB) error {
	log.Printf("updateBatchJobCounts: batchID=%s, inProgressDelta=%d, waitingDelta=%d", batchID, inProgressDelta, waitingDelta)

	// Get current values for logging
	var currentIP, currentW int64
	errQuery := dbInst.QueryRowContext(ctx, `
		SELECT jobs_in_progress, jobs_waiting
		FROM job_batches WHERE batch_id = $1
	`, batchID).Scan(&currentIP, &currentW)
	if errQuery != nil {
		log.Printf("failed to query current job counts: %v", errQuery)
	} else {
		log.Printf("batch %s: before job count update - IP=%d, Wait=%d",
			batchID, currentIP, currentW)
	}

	_, err := dbInst.ExecContext(ctx, `
		UPDATE job_batches
		SET jobs_in_progress = jobs_in_progress + $1,
		    jobs_waiting = jobs_waiting + $2,
		    updated_at = now()
		WHERE batch_id = $3
	`, inProgressDelta, waitingDelta, batchID)
	if err != nil {
		return err
	}

	// Get updated values for logging
	var newIP, newW int64
	errQuery = dbInst.QueryRowContext(ctx, `
		SELECT jobs_in_progress, jobs_waiting
		FROM job_batches WHERE batch_id = $1
	`, batchID).Scan(&newIP, &newW)
	if errQuery != nil {
		log.Printf("failed to query updated job counts: %v", errQuery)
	} else {
		log.Printf("batch %s: after job count update - IP=%d, Wait=%d",
			batchID, newIP, newW)
	}

	return err
}

// updateJobStagingStatus updates the status of a job in the staging table
func updateJobStagingStatus(ctx context.Context, jobID string, batchID string, status string, dbInst *db.DB) error {
	_, err := dbInst.ExecContext(ctx, `
		UPDATE job_staging
		SET status = $1, updated_at = now()
		WHERE id = $2 AND batch_id = $3
	`, status, jobID, batchID)
	return err
}

// updateJobStagingDescription updates the job_description field of a job in the staging table
func updateJobStagingDescription(ctx context.Context, jobID string, batchID string, jobDescription string, dbInst *db.DB) error {
	_, err := dbInst.ExecContext(ctx, `
		UPDATE job_staging
		SET job_description = $1, updated_at = now()
		WHERE id = $2 AND batch_id = $3
	`, jobDescription, jobID, batchID)
	return err
}

// updateJobStagingStatusAndDescription updates both status and job_description of a job in the staging table
func updateJobStagingStatusAndDescription(ctx context.Context, jobID string, batchID string, status string, jobDescription string, dbInst *db.DB) error {
	_, err := dbInst.ExecContext(ctx, `
		UPDATE job_staging
		SET status = $1, job_description = $2, updated_at = now()
		WHERE id = $3 AND batch_id = $4
	`, status, jobDescription, jobID, batchID)
	return err
}

// updateJobStagingResponse stores the LLM response in the raw_json field of a job in the staging table
func updateJobStagingResponse(ctx context.Context, jobID string, batchID string, response string, dbInst *db.DB) error {
	_, err := dbInst.ExecContext(ctx, `
		UPDATE job_staging
		SET raw_json = $1::jsonb, updated_at = now()
		WHERE id = $2 AND batch_id = $3
	`, response, jobID, batchID)
	return err
}

// updateJobStagingStatusAndResponse updates both status and raw_json (LLM response) of a job in the staging table
func updateJobStagingStatusAndResponse(ctx context.Context, jobID string, batchID string, status string, response string, dbInst *db.DB) error {
	_, err := dbInst.ExecContext(ctx, `
		UPDATE job_staging
		SET status = $1, raw_json = $2::jsonb, updated_at = now()
		WHERE id = $3 AND batch_id = $4
	`, status, response, jobID, batchID)
	return err
}

// maybeUpdateBatchStatus checks if all jobs for a batch are done and updates the batch status to completed.
func maybeUpdateBatchStatus(ctx context.Context, batchID string, userID int64, dbInst *db.DB) error {
	var pending, inprog int
	err := dbInst.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending'),
			COUNT(*) FILTER (WHERE status = 'inprogress')
		FROM job_staging
		WHERE batch_id = $1
	`, batchID).Scan(&pending, &inprog)
	if err != nil {
		return err
	}
	if pending == 0 && inprog == 0 {
		_, err := dbInst.ExecContext(ctx, `
			UPDATE job_batches
			SET status = 'completed', updated_at = now()
			WHERE batch_id = $1 AND user_id = $2
		`, batchID, userID)
		return err
	}
	return nil
}

// workerFn processes jobs from the channel, respecting the LLM semaphore.
func workerFn(
	jobs <-chan batchJob,
	batchID string,
	userID int64,
	dbInst *db.DB,
	llmLimiter chan struct{}, // shared semaphore limiting LLM calls
) {
	for job := range jobs {
		// Fetch the actual job data from the staging table
		var jobData struct {
			JobDescription string
			Provider       string
			Model          string
		}
		err := dbInst.QueryRowContext(context.Background(), `
			SELECT job_description, provider, model
			FROM job_staging
			WHERE id = $1 AND batch_id = $2
		`, job.JobID, batchID).Scan(&jobData.JobDescription, &jobData.Provider, &jobData.Model)
		if err != nil {
			log.Printf("batch %s job %s: failed to fetch job data from staging: %v", batchID, job.JobID, err)
			// Update job staging status to failed
			if err := updateJobStagingStatus(context.Background(), job.JobID, batchID, "failed", dbInst); err != nil {
				log.Printf("failed to update job staging status to failed: %v", err)
			}
			continue
		}

		// log.Printf("batch %s job %s: moving from waiting to in progress", batchID, job.JobID)
		// Move job from waiting to in progress in staging table
		if err := updateJobStagingStatus(context.Background(), job.JobID, batchID, "inprogress", dbInst); err != nil {
			log.Printf("failed to update job staging status: %v", err)
		}

		// Acquire a token from the LLM semaphore (blocks if none available)
		// log.Printf("batch %s job %s: waiting for LLM token", batchID, job.JobID)
		llmLimiter <- struct{}{}
		// log.Printf("batch %s job %s: acquired LLM token, starting processing", batchID, job.JobID)

		// Create a temporary batchJob with the actual data from the database
		dbJob := batchJob{
			JobDescription: jobData.JobDescription,
			Provider:       jobData.Provider,
			Model:          jobData.Model,
			UserID:         userID,
			BatchID:        batchID,
			JobID:          job.JobID,
		}

		generatedText, err := processSingleJob(context.Background(), dbJob, dbInst)
		// log.Printf("batch %s job %s: finished LLM processing", batchID, job.JobID)
		<-llmLimiter // release token
		// log.Printf("batch %s job %s: released LLM token", batchID, job.JobID)

		if err != nil {
			log.Printf("batch %s job %s failed: %v", batchID, job.JobID, err)
			// Update job staging status to failed
			if err := updateJobStagingStatus(context.Background(), job.JobID, batchID, "failed", dbInst); err != nil {
				log.Printf("failed to update job staging status to failed: %v", err)
			}
			// Optionally store error info in raw_json
			if updateErr := updateJobStagingResponse(context.Background(), job.JobID, batchID, fmt.Sprintf(`{"error":"%v"}`, err), dbInst); updateErr != nil {
				log.Printf("failed to update job staging raw_json with error: %v", updateErr)
			}
			// Check if batch is done after this failure
			if err := maybeUpdateBatchStatus(context.Background(), batchID, userID, dbInst); err != nil {
				log.Printf("failed to maybe update batch status: %v", err)
			}
			continue
		}
		// Success path – insert raw response in background and update staging
		// log.Printf("batch %s job %s: saving raw response to DB", batchID, job.JobID)
		go func() {
			ctx := context.Background()
			_, err := dbInst.ExecContext(ctx, `
				INSERT INTO job_extractions
					(id, job_id, provider, model, prompt_version, raw_response, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, dbJob.JobID, dbJob.JobID, dbJob.Provider, dbJob.Model, "v1", generatedText, time.Now())
			if err != nil {
				log.Printf("failed to save job extraction: %v", err)
			}
		}()
		// Update job staging status to completed and store raw response
		if err := updateJobStagingStatusAndResponse(context.Background(), job.JobID, batchID, "completed", generatedText, dbInst); err != nil {
			log.Printf("failed to update job staging status and raw_json: %v", err)
		}
		// Check if batch is done after this success
		if err := maybeUpdateBatchStatus(context.Background(), batchID, userID, dbInst); err != nil {
			log.Printf("failed to maybe update batch status: %v", err)
		}
	}
}

// HandleProcessJobBatch starts a batch processing job.
// It expects a JSON payload with an array of job objects.
func HandleProcessJobBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchProcessJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Jobs) == 0 {
		http.Error(w, "jobs array must not be empty", http.StatusBadRequest)
		return
	}
	for _, j := range req.Jobs {
		if strings.TrimSpace(j.JobDescription) == "" {
			http.Error(w, "job_description is required for each job", http.StatusBadRequest)
			return
		}
	}

	// Auth – userID from context (set by auth middleware)
	userIDVal := r.Context().Value("userID")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("batch request: userID=%d, jobs=%d", userID, len(req.Jobs))

	if dbInst == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	// Create batch record with essential fields only
	batchID := uuid.New().String()
	_, err := dbInst.ExecContext(r.Context(), `
		INSERT INTO job_batches
			(batch_id, user_id, total_jobs, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, batchID, userID, len(req.Jobs), "pending", time.Now(), time.Now())
	if err != nil {
		log.Printf("failed to create batch record: %v", err)
		http.Error(w, "failed to create batch record", http.StatusInternalServerError)
		return
	}

	// Create staging entries for each job and keep the database IDs for the sender loop.
	stagedJobIDs := make([]string, 0, len(req.Jobs))
	for _, j := range req.Jobs {
		jobID := uuid.New().String()
		// Set default provider and model if not specified
		provider := j.Provider
		if provider == "" {
			provider = "gemini"
		}
		model := j.Model
		if model == "" {
			model = "gemini-2.0-flash"
		}
		_, err := dbInst.ExecContext(r.Context(), `
			INSERT INTO job_staging
				(id, batch_id, user_id, provider, model, job_description, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, jobID, batchID, userID, provider, model, j.JobDescription, "pending", time.Now(), time.Now())
		if err != nil {
			log.Printf("failed to create staging entry for job: %v", err)
			continue
		}
		stagedJobIDs = append(stagedJobIDs, jobID)
	}

	// Create a cancellable context for this batch
	ctx, cancel := context.WithCancel(context.Background())
	// Store the cancel function so we can stop the batch later
	batchCancels.Store(batchID, cancel)

	// Configure worker pool & LLM concurrency (env overridable)
	workerCount := runtime.GOMAXPROCS(0)
	if env := os.Getenv("WORKER_COUNT"); env != "" {
		wc, err := strconv.Atoi(env)
		if err == nil && wc > 0 {
			workerCount = wc
		}
	}
	llmConcurrency := 5
	if env := os.Getenv("LLM_MAX_CONCURRENCY"); env != "" {
		lc, err := strconv.Atoi(env)
		if err == nil && lc > 0 {
			llmConcurrency = lc
		}
	}

	jobChan := make(chan batchJob, 1)
	llmLimiter := make(chan struct{}, llmConcurrency)

	// Start workers
	for i := 0; i < workerCount; i++ {
		go workerFn(jobChan, batchID, userID, dbInst, llmLimiter)
	}

	// Sender goroutine: sends jobs unless stop is requested
	go func() {
		defer close(jobChan)
		for i, j := range req.Jobs {
			if i >= len(stagedJobIDs) {
				log.Printf("batch %s: missing staging ID for job index %d", batchID, i)
				continue
			}

			jobID := stagedJobIDs[i]
			select {
			case <-ctx.Done():
				log.Printf("batch %s: stop requested, marking %d remaining jobs as stopped", batchID, len(req.Jobs)-i)
				for _, remainingJobID := range stagedJobIDs[i:] {
					if err := updateJobStagingStatus(context.Background(), remainingJobID, batchID, "stopped", dbInst); err != nil {
						log.Printf("failed to update job staging status to stopped: %v", err)
					}
				}
				return
			default:
				jobChan <- batchJob{
					JobDescription: j.JobDescription,
					Provider:       j.Provider,
					Model:          j.Model,
					UserID:         userID,
					BatchID:        batchID,
					JobID:          jobID,
				}
			}
		}
	}()

	// Respond with the batch ID immediately (202 Accepted)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"batch_id": batchID})
}

// HandleStopBatch stops a batch processing job by canceling its context.
// It expects a POST request to /batch/stop with JSON body {"batch_id": "..."}.
func HandleStopBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BatchID string `json:"batch_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	batchID := strings.TrimSpace(req.BatchID)
	if batchID == "" {
		http.Error(w, "missing batch_id", http.StatusBadRequest)
		return
	}

	// Auth – userID from context (set by auth middleware)
	userIDVal := r.Context().Value("userID")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if dbInst == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	// Verify batch belongs to user and get its status
	var status string
	err := dbInst.QueryRowContext(r.Context(), `
		SELECT status FROM job_batches WHERE batch_id = $1 AND user_id = $2
	`, batchID, userID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "batch not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to verify batch", http.StatusInternalServerError)
		return
	}

	// If batch already completed, stopping is not needed
	if status == "completed" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "batch already completed"})
		return
	}

	// Retrieve cancel function
	if cancelFunc, ok := batchCancels.Load(batchID); ok {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			cancel() // signal cancellation

			// Update batch status to stopped
			if _, err := dbInst.ExecContext(r.Context(), `
				UPDATE job_batches
				SET status = 'stopped',
				    updated_at = now()
				WHERE batch_id = $1 AND user_id = $2
			`, batchID, userID); err != nil {
				log.Printf("failed to update batch status to stopped: %v", err)
				// Continue anyway since we already cancelled
			}

			// Update job staging status to stopped for pending and inprogress jobs
			if _, err := dbInst.ExecContext(r.Context(), `
				UPDATE job_staging
				SET status = 'stopped', updated_at = now()
				WHERE batch_id = $1 AND status IN ('pending', 'inprogress')
			`, batchID); err != nil {
				log.Printf("failed to update job staging status to stopped: %v", err)
			}
			log.Printf("stop requested for batch %s", batchID)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "stop requested",
			})
			return
		}
	}
	// If no cancel function found but batch exists and not completed,
	// it may have just started but sender already finished (unlikely)
	// or workers all exited. Treat as not stoppable.
	http.Error(w, "batch cannot be stopped (no active cancel)", http.StatusBadRequest)
}

// GetBatchStatus returns the current status and progress for a batch.
func GetBatchStatus(w http.ResponseWriter, r *http.Request) {
	// Extract batch_id from path: /batch/<uuid>

	batchID := strings.TrimPrefix(r.URL.Path, "/batch/")
	fmt.Printf("GetBatchStatus called for batchID=%s\n", batchID)
	if batchID == "" {
		http.Error(w, "missing batch_id", http.StatusBadRequest)
		return
	}

	// Auth – same userID extraction used elsewhere
	userIDVal := r.Context().Value("userID")
	if userIDVal == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if dbInst == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	var resp BatchStatusResponse
	// Get basic batch info from job_batches
	err := dbInst.QueryRowContext(r.Context(), `
		SELECT batch_id, user_id, total_jobs, status, created_at
		FROM job_batches
		WHERE batch_id = $1 AND user_id = $2
	`, batchID, userID).Scan(
		&resp.BatchID,
		&resp.UserID,
		&resp.TotalJobs,
		&resp.Status,
		&resp.CreatedAt,
	)

	fmt.Println("response after scanning batch info:", resp, "error:", err)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "batch not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to read batch status", http.StatusInternalServerError)
		return
	}

	// Get job counts from job_staging table
	var pendingCount, inprogressCount, completedCount, failedCount int
	countsErr := dbInst.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'inprogress') as inprogress,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed
		FROM job_staging
		WHERE batch_id = $1
	`, batchID).Scan(&pendingCount, &inprogressCount, &completedCount, &failedCount)
	if countsErr != nil {
		log.Printf("failed to get job counts from staging table: %v", countsErr)
		// Set defaults to avoid breaking the response
		pendingCount = 0
		inprogressCount = 0
		completedCount = 0
		failedCount = 0
	}

	fmt.Printf("batch %s: counts - pending=%d, inprogress=%d, completed=%d, failed=%d\n",
		batchID, pendingCount, inprogressCount, completedCount, failedCount)

	// Set response values
	resp.ProcessedOK = completedCount
	resp.ProcessedFail = failedCount
	resp.JobsInProgress = inprogressCount
	resp.JobsWaiting = pendingCount
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "batch not found or access denied", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to read batch status", http.StatusInternalServerError)
		return
	}

	fmt.Printf("GetBatchStatus: returning response: %+v\n", resp)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
