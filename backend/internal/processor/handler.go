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
	JobDescription string `json:"description"` // The description to send to LLM for skills extraction
	// Pre-populated basic job facts (won't be sent to LLM)
	Title          string `json:"title,omitempty"`
	Company        string `json:"company,omitempty"`
	Location       string `json:"location,omitempty"`
	EmploymentType string `json:"employment_type,omitempty"`
	WorkMode       string `json:"work_mode,omitempty"`
	Seniority      string `json:"seniority,omitempty"`
	// Experience as min_years and max_years
	ExperienceMinYears *int `json:"min_years,omitempty"`
	ExperienceMaxYears *int `json:"max_years,omitempty"`
	// Source-specific IDs for duplicate detection
	LinkedinJobID string `json:"linkedin_job_id,omitempty"`
	NaukriJobID   string `json:"naukri_job_id,omitempty"`
	Source        string `json:"source,omitempty"` // e.g., "linkedin", "naukri"
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

// HandleProcessJob processes job data, extracting only skills/responsibilities/etc. from description
// via LLM, while checking for duplicates using source-specific IDs.
func HandleProcessJob(w http.ResponseWriter, r *http.Request) {
	log.Printf("HandleProcessJob: START - Processing job request")

	// Decode request body
	var req ProcessJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("HandleProcessJob: ERROR - Invalid JSON: %v", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.JobDescription == "" {
		log.Printf("HandleProcessJob: ERROR - Job description is required")
		http.Error(w, "description is required for skills extraction", http.StatusBadRequest)
		return
	}

	// Set defaults for provider and model if not specified
	if req.Provider == "" {
		req.Provider = "gemini" // default provider
	}
	if req.Model == "" {
		req.Model = "gemini-2.0-flash" // default model
	}
	log.Printf("HandleProcessJob: Request details - Provider: %s, Model: %s, Source: %s, LinkedIn ID: %s, Naukri ID: %s",
		req.Provider, req.Model, req.Source, req.LinkedinJobID, req.NaukriJobID)

	// Retrieve user ID from context (set by auth middleware).
	userIDVal := r.Context().Value("userID")
	if userIDVal == nil {
		log.Printf("HandleProcessJob: ERROR - Unauthorized request")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		log.Printf("HandleProcessJob: ERROR - Invalid user ID in context")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get encrypted API key from database for the specific provider.
	if dbInst == nil {
		log.Printf("HandleProcessJob: ERROR - Database not initialized")
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}
	encryptedKey, err := dbInst.GetAPIKey(r.Context(), userID, req.Provider)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("HandleProcessJob: ERROR - API key not found for user %d and provider %s", userID, req.Provider)
			http.Error(w, "API key not found for user and provider", http.StatusUnauthorized)
			return
		}
		log.Printf("HandleProcessJob: ERROR - Could not retrieve API key: %v", err)
		http.Error(w, "could not retrieve API key", http.StatusInternalServerError)
		return
	}

	// Decrypt the API key.
	apiKey, err := apikey.Decrypt(encryptedKey)
	if err != nil {
		log.Printf("HandleProcessJob: ERROR - Decryption failed: %v", err)
		http.Error(w, "decryption failed", http.StatusInternalServerError)
		return
	}

	log.Printf("HandleProcessJob: Decrypted API key retrieved successfully")

	// Check for duplicate using source-specific IDs if provided
	if req.LinkedinJobID != "" || req.NaukriJobID != "" {
		var exists bool
		var checkErr error
		if req.LinkedinJobID != "" {
			log.Printf("HandleProcessJob: Checking for duplicate LinkedIn job ID: %s", req.LinkedinJobID)
			exists, checkErr = checkLinkedinJobExists(r.Context(), dbInst, req.LinkedinJobID)
		} else {
			log.Printf("HandleProcessJob: Checking for duplicate Naukri job ID: %s", req.NaukriJobID)
			exists, checkErr = checkNaukriJobExists(r.Context(), dbInst, req.NaukriJobID)
		}
		if checkErr != nil {
			log.Printf("HandleProcessJob: ERROR - Failed to check for duplicate job: %v", checkErr)
			http.Error(w, fmt.Sprintf("failed to check for duplicate job: %v", checkErr), http.StatusInternalServerError)
			return
		}
		if exists {
			log.Printf("HandleProcessJob: Duplicate job found - %s", func() string {
				if req.LinkedinJobID != "" {
					return "LinkedIn ID: " + req.LinkedinJobID
				} else {
					return "Naukri ID: " + req.NaukriJobID
				}
			}())
			// Return duplicate response
			response := ProcessJobResponse{
				Success: false,
				Message: "Job already exists (duplicate)",
				JobID:   "", // No new job created
				Data:    map[string]string{"duplicate": "true"},
			}
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Printf("HandleProcessJob: ERROR - Failed to encode duplicate response: %v", err)
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(responseBytes)
			return
		}
		log.Printf("HandleProcessJob: No duplicate found for job")
	}

	// Create AI provider for LLM extraction (only for skills/responsibilities from description)
	gen, err := provider.Factory(r.Context(), req.Provider, apiKey, req.Model)
	if err != nil {
		log.Printf("HandleProcessJob: ERROR - Failed to create AI provider: %v", err)
		http.Error(w, fmt.Sprintf("unsupported provider %s: %v", req.Provider, err), http.StatusBadGateway)
		return
	}

	// Build prompt for skills extraction using the new template (only processes description)
	promptData := map[string]string{
		"Description": req.JobDescription,
	}
	prompt, err := prompts.RenderTemplate("job_skills_extraction", promptData)
	if err != nil {
		log.Printf("HandleProcessJob: ERROR - Failed to render job skills extraction template: %v", err)
		http.Error(w, fmt.Sprintf("failed to render job skills extraction template: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate skills/responsibilities/etc. from description ONLY
	log.Printf("HandleProcessJob: Calling LLM for skills extraction")
	generatedText, err := gen.GenerateText(r.Context(), prompt)
	if err != nil {
		log.Printf("HandleProcessJob: ERROR - Failed to generate text: %v", err)
		http.Error(w, fmt.Sprintf("failed to generate text: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("HandleProcessJob: LLM generation successful, response length: %d", len(generatedText))

	// Extract JSON object from the LLM output
	jsonStr := strings.TrimSpace(generatedText)
	startIdx := strings.Index(jsonStr, "{")
	endIdx := strings.LastIndex(jsonStr, "}")
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		log.Printf("HandleProcessJob: ERROR - No JSON object found in provider response")
		http.Error(w, "no JSON object found in provider response", http.StatusBadGateway)
		return
	}
	jsonStr = jsonStr[startIdx : endIdx+1]

	// Parse the LLM response to validate it's valid JSON
	var llmResponse map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &llmResponse); err != nil {
		log.Printf("HandleProcessJob: ERROR - Failed to parse provider response as JSON: %v", err)
		http.Error(w, fmt.Sprintf("failed to parse provider response as JSON: %v", err), http.StatusBadGateway)
		return
	}
	log.Printf("HandleProcessJob: LLM response parsed successfully")

	// Create a stable job ID for tracking
	jobID := uuid.New().String()

	// Build the final combined object with both pre-provided fields and LLM-extracted fields
	// Format:
	// {
	//   "title": {"value": string|null, "confidence": number, "evidence": string|null},
	//   "company": {"value": string|null, "confidence": number, "evidence": string|null},
	//   "location": {"city": string|null, "state": string|null, "country": string|null, "confidence": number, "evidence": string|null},
	//   "employment_type": {"value": "...", "confidence": number, "evidence": string|null},
	//   "work_mode": {"value": "...", "confidence": number, "evidence": string|null},
	//   "seniority": {"value": "...", "confidence": number, "evidence": string|null},
	//   "experience": {"minimum_years": number|null, "maximum_years": number|null, "confidence": number, "evidence": string|null},
	//   "skills": [...],  // from LLM
	//   "responsibilities": [...],  // from LLM
	//   "soft_skills": [...],  // from LLM
	//   "domain_knowledge": [...]  // from LLM
	// }

	// Helper function to create a field object with value, confidence, and evidence
	makeField := func(value string, confidence float64, evidence string) map[string]interface{} {
		if value == "" {
			return map[string]interface{}{
				"value":      nil,
				"confidence": 0.0,
				"evidence":   nil,
			}
		}
		return map[string]interface{}{
			"value":      value,
			"confidence": confidence,
			"evidence":   evidence,
		}
	}

	// Helper function to safely convert *string to string

	// Helper function to safely convert *int to *int (for JSON marshaling)

	// Helper function to extract city from location string (simple implementation)
	extractCity := func(location string) string {
		if location == "" {
			return ""
		}
		// Simple split by comma and take first part as city
		parts := strings.Split(location, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
		return ""
	}

	// Helper function to extract state from location string (simple implementation)
	extractState := func(location string) string {
		if location == "" {
			return ""
		}
		// Simple split by comma and take second part as state if exists
		parts := strings.Split(location, ",")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
		return ""
	}

	// Helper function to extract country from location string (simple implementation)
	extractCountry := func(location string) string {
		if location == "" {
			return ""
		}
		// Simple split by comma and take last part as country if exists
		parts := strings.Split(location, ",")
		if len(parts) > 2 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
		return ""
	}

	// Build the final object
	combinedObject := map[string]interface{}{
		"title": makeField(
			req.Title,
			0.95, // High confidence for pre-provided data
			fmt.Sprintf("Provided in API request: %s", req.Title),
		),
		"company": makeField(
			req.Company,
			0.95, // High confidence for pre-provided data
			fmt.Sprintf("Provided in API request: %s", req.Company),
		),
		"location": map[string]interface{}{
			"city":       extractCity(req.Location),
			"state":      extractState(req.Location),
			"country":    extractCountry(req.Location),
			"confidence": 0.95, // High confidence for pre-provided data
			"evidence":   fmt.Sprintf("Provided in API request: %s", req.Location),
		},
		"employment_type": makeField(
			req.EmploymentType,
			0.95, // High confidence for pre-provided data
			fmt.Sprintf("Provided in API request: %s", req.EmploymentType),
		),
		"work_mode": makeField(
			req.WorkMode,
			0.95, // High confidence for pre-provided data
			fmt.Sprintf("Provided in API request: %s", req.WorkMode),
		),
		// "seniority": makeField(
		// 	req.Seniority,
		// 	0.95, // High confidence for pre-provided data
		// 	fmt.Sprintf("Provided in API request: %s", req.Seniority),
		// ),
		// "experience": map[string]interface{}{
		// 	"minimum_years": nullableIntToPtr(req.ExperienceMinYears),
		// 	"maximum_years": nullableIntToPtr(req.ExperienceMaxYears),
		// 	"confidence":    0.95, // High confidence for pre-provided data
		// 	"evidence": fmt.Sprintf("Provided in API request: min=%v, max=%v",
		// 		nullableIntToPtr(req.ExperienceMinYears), nullableIntToPtr(req.ExperienceMaxYears)),
		// },
		"skills":           llmResponse["skills"],
		"responsibilities": llmResponse["responsibilities"],
		"soft_skills":      llmResponse["soft_skills"],
		"domain_knowledge": llmResponse["domain_knowledge"],
		"experience":       llmResponse["experience"],
		"seniority":        llmResponse["seniority"],
	}

	// Ensure LLM-extracted fields are present (default to empty arrays if missing)
	if combinedObject["skills"] == nil {
		combinedObject["skills"] = []interface{}{}
	}
	if combinedObject["responsibilities"] == nil {
		combinedObject["responsibilities"] = []interface{}{}
	}
	if combinedObject["soft_skills"] == nil {
		combinedObject["soft_skills"] = []interface{}{}
	}
	if combinedObject["domain_knowledge"] == nil {
		combinedObject["domain_knowledge"] = []interface{}{}
	}

	// Save the combined object to job_extractions table
	if dbInst == nil {
		log.Printf("HandleProcessJob: ERROR - Database not initialized")
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	job_extract_id := uuid.New().String()
	combinedObjectJSON, err := json.Marshal(combinedObject)
	if err != nil {
		log.Printf("HandleProcessJob: ERROR - Failed to marshal combined object: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	_, err = dbInst.ExecContext(r.Context(), `
		INSERT INTO job_extractions
			(id, job_id, provider, model, prompt_version, raw_response, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		job_extract_id,             // id
		jobID,                      // job_id
		req.Provider,               // provider
		req.Model,                  // model
		"v1",                       // prompt_version
		string(combinedObjectJSON), // raw_response (the combined object as JSON)
		time.Now(),                 // created_at
	)
	if err != nil {
		log.Printf("HandleProcessJob: ERROR - Failed to save job extraction: %v", err)
		http.Error(w, "failed to save job extraction", http.StatusInternalServerError)
		return
	}
	log.Printf("HandleProcessJob: Successfully saved combined object to job_extractions table with ID: %s", job_extract_id)

	// Prepare and return the response with the combined object
	response := ProcessJobResponse{
		Success: true,
		Message: "Job processed successfully",
		JobID:   jobID,
		Data:    combinedObject,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("HandleProcessJob: ERROR - Failed to encode response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	// Return response
	log.Printf("HandleProcessJob: END - Job processed successfully. Returning combined object")
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}

// checkLinkedinJobExists checks if a job with the given LinkedIn job ID already exists
func checkLinkedinJobExists(ctx context.Context, dbInst *db.DB, linkedinJobID string) (bool, error) {
	log.Printf("checkLinkedinJobExists: Checking for LinkedIn job ID: %s", linkedinJobID)
	var exists bool
	err := dbInst.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM job_staging WHERE linkedin_job_id = $1
		)
	`, linkedinJobID).Scan(&exists)
	if err != nil {
		log.Printf("checkLinkedinJobExists: ERROR - Failed to check for LinkedIn job ID %s: %v", linkedinJobID, err)
		return exists, err
	}
	log.Printf("checkLinkedinJobExists: LinkedIn job ID %s exists: %v", linkedinJobID, exists)
	return exists, err
}

// checkNaukriJobExists checks if a job with the given Naukri job ID already exists
func checkNaukriJobExists(ctx context.Context, dbInst *db.DB, naukriJobID string) (bool, error) {
	var exists bool
	err := dbInst.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM job_staging WHERE naukri_job_id = $1
		)
	`, naukriJobID).Scan(&exists)
	return exists, err
}

// BatchProcessJobRequest represents the payload for starting a batch job.
type BatchProcessJobRequest struct {
	Jobs []ProcessJobRequest `json:"jobs"`
}

// batchJob is the internal unit of work sent to workers.
type batchJob struct {
	JobDescription string
	// Pre-populated basic job facts
	Title              string
	Company            string
	Location           string
	EmploymentType     string
	WorkMode           string
	Seniority          string
	ExperienceMinYears *int
	ExperienceMaxYears *int
	// Source-specific IDs for duplicate detection
	LinkedinJobID string
	NaukriJobID   string
	Source        string
	Provider      string
	Model         string
	UserID        int64
	BatchID       string // UUID of the batch
	JobID         string // UUID for this specific job
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
	promptData := map[string]string{"Description": job.JobDescription}
	prompt, err := prompts.RenderTemplate("job_skills_extraction", promptData)
	if err != nil {
		return "", err
	}

	fmt.Printf("processSingleJob: Processing job %s with provider %s and model %s\n", job.JobID, job.Provider, job.Model, "Prompt: ", prompt)
	// Call the LLM
	generatedText, err := gen.GenerateText(ctx, prompt)

	fmt.Printf("processSingleJob: Generated text for job %s: %s\n", job.JobID, generatedText)
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
			JobDescription     string
			Provider           string
			Model              string
			Title              string
			Company            string
			Location           string
			EmploymentType     string
			WorkMode           string
			Seniority          string
			ExperienceMinYears *int
			ExperienceMaxYears *int
			LinkedinJobID      string
			NaukriJobID        string
			Source             string
		}
		err := dbInst.QueryRowContext(context.Background(), `
			SELECT job_description, provider, model,
				title, company, location, employment_type, work_mode, seniority,
				min_years, max_years,
				linkedin_job_id, naukri_job_id, source
			FROM job_staging
			WHERE id = $1 AND batch_id = $2
		`, job.JobID, batchID).Scan(&jobData.JobDescription, &jobData.Provider, &jobData.Model,
			&jobData.Title, &jobData.Company, &jobData.Location, &jobData.EmploymentType, &jobData.WorkMode, &jobData.Seniority,
			&jobData.ExperienceMinYears, &jobData.ExperienceMaxYears,
			&jobData.LinkedinJobID, &jobData.NaukriJobID, &jobData.Source)
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
			JobDescription:     jobData.JobDescription,
			Provider:           jobData.Provider,
			Model:              jobData.Model,
			Title:              jobData.Title,
			Company:            jobData.Company,
			Location:           jobData.Location,
			EmploymentType:     jobData.EmploymentType,
			WorkMode:           jobData.WorkMode,
			Seniority:          jobData.Seniority,
			ExperienceMinYears: jobData.ExperienceMinYears,
			ExperienceMaxYears: jobData.ExperienceMaxYears,
			LinkedinJobID:      jobData.LinkedinJobID,
			NaukriJobID:        jobData.NaukriJobID,
			Source:             jobData.Source,
			UserID:             userID,
			BatchID:            batchID,
			JobID:              job.JobID,
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
	log.Printf("HandleProcessJobBatch: START - Processing batch job request")

	var req BatchProcessJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("HandleProcessJobBatch: ERROR - Invalid JSON: %v", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Jobs) == 0 {
		log.Printf("HandleProcessJobBatch: ERROR - Jobs array is empty")
		http.Error(w, "jobs array must not be empty", http.StatusBadRequest)
		return
	}
	for _, j := range req.Jobs {
		if strings.TrimSpace(j.JobDescription) == "" {
			log.Printf("HandleProcessJobBatch: ERROR - Job description is required for each job")
			http.Error(w, "job_description is required for each job", http.StatusBadRequest)
			return
		}
	}

	// Auth – userID from context (set by auth middleware)
	userIDVal := r.Context().Value("userID")
	if userIDVal == nil {
		log.Printf("HandleProcessJobBatch: ERROR - Unauthorized request")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		log.Printf("HandleProcessJobBatch: ERROR - Invalid user ID in context")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("HandleProcessJobBatch: Batch request received - userID: %d, total jobs: %d", userID, len(req.Jobs))

	if dbInst == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	// Check for duplicates and separate jobs into new vs duplicate
	log.Printf("HandleProcessJobBatch: Starting duplicate check for %d jobs", len(req.Jobs))
	var newJobs []ProcessJobRequest
	var duplicateCount int
	var savedCount int

	for _, j := range req.Jobs {
		// Check for duplicate using source-specific IDs if provided
		var isDuplicate bool
		var checkErr error

		if j.LinkedinJobID != "" {
			log.Printf("HandleProcessJobBatch: Checking duplicate for LinkedIn ID: %s", j.LinkedinJobID)
			isDuplicate, checkErr = checkLinkedinJobExists(r.Context(), dbInst, j.LinkedinJobID)
		} else if j.NaukriJobID != "" {
			log.Printf("HandleProcessJobBatch: Checking duplicate for Naukri ID: %s", j.NaukriJobID)
			isDuplicate, checkErr = checkNaukriJobExists(r.Context(), dbInst, j.NaukriJobID)
		}

		if checkErr != nil {
			log.Printf("HandleProcessJobBatch: ERROR - Failed to check for duplicate job: %v", checkErr)
			// If we can't check, treat as new to be safe
			isDuplicate = false
		}

		if isDuplicate {
			log.Printf("HandleProcessJobBatch: Duplicate job found - %s", func() string {
				if j.LinkedinJobID != "" {
					return "LinkedIn ID: " + j.LinkedinJobID
				} else {
					return "Naukri ID: " + j.NaukriJobID
				}
			}())
			duplicateCount++
			continue
		}

		log.Printf("HandleProcessJobBatch: No duplicate found for job - %s", func() string {
			if j.LinkedinJobID != "" {
				return "LinkedIn ID: " + j.LinkedinJobID
			} else if j.NaukriJobID != "" {
				return "Naukri ID: " + j.NaukriJobID
			} else {
				return "No source ID"
			}
		}())
		newJobs = append(newJobs, j)
	}

	savedCount = len(newJobs)
	log.Printf("HandleProcessJobBatch: Duplicate check complete - Total: %d, New: %d, Duplicates: %d", len(req.Jobs), savedCount, duplicateCount)

	// If all jobs are duplicates, return early
	if savedCount == 0 {
		log.Printf("HandleProcessJobBatch: All jobs are duplicates, returning early")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"batch_id":        "",
			"total_jobs":      len(req.Jobs),
			"saved_count":     0,
			"duplicate_count": duplicateCount,
			"message":         "All jobs were duplicates",
		})
		return
	}

	// Create batch record with actual number of new jobs to process
	batchID := uuid.New().String()
	_, err := dbInst.ExecContext(r.Context(), `
		INSERT INTO job_batches
			(batch_id, user_id, total_jobs, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, batchID, userID, savedCount, "pending", time.Now(), time.Now())
	if err != nil {
		log.Printf("failed to create batch record: %v", err)
		http.Error(w, "failed to create batch record", http.StatusInternalServerError)
		return
	}

	// Create staging entries for new jobs only and keep the database IDs for the sender loop.
	stagedJobIDs := make([]string, 0, savedCount)
	for _, j := range newJobs {
		jobID := uuid.New().String()
		// Set default provider and model if not specified
		var provider = j.Provider
		if provider == "" {
			provider = "gemini"
		}
		var model = j.Model
		if model == "" {
			model = "gemini-2.0-flash"
		}
		_, err := dbInst.ExecContext(r.Context(), `
			INSERT INTO job_staging
				(id, batch_id, user_id, provider, model, job_description,
				 linkedin_job_id, naukri_job_id, source, title, company, location,
				 employment_type, work_mode, seniority, min_years, max_years, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		`, jobID, batchID, userID, provider, model, j.JobDescription,
			func() interface{} {
				if j.LinkedinJobID != "" {
					return j.LinkedinJobID
				}
				return nil
			}(),
			func() interface{} {
				if j.NaukriJobID != "" {
					return j.NaukriJobID
				}
				return nil
			}(),
			j.Source,
			j.Title,
			j.Company,
			j.Location,
			j.EmploymentType,
			j.WorkMode,
			j.Seniority,
			j.ExperienceMinYears,
			j.ExperienceMaxYears,
			"pending", // status
			time.Now(),
			time.Now(),
		)
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

	// Respond with batch ID and counts immediately (202 Accepted)
	log.Printf("HandleProcessJobBatch: END - Batch processed. BatchID: %s, TotalJobs: %d, SavedCount: %d, DuplicateCount: %d", batchID, len(req.Jobs), savedCount, duplicateCount)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"batch_id":        batchID,
		"total_jobs":      len(req.Jobs),
		"saved_count":     savedCount,
		"duplicate_count": duplicateCount,
	})
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
