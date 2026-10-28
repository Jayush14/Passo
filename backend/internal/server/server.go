package server

import (
	"net/http"
	"os"
	"strconv"
	"time"
	"runtime"

	"passo/backend/internal/config"
	"passo/backend/internal/db"
	"passo/backend/internal/plugins/market-data-engine/api/handlers"
	"passo/backend/internal/plugins/market-data-engine/processing"
)

// Server holds server configuration.
type Server struct {
	Cfg *config.Config
	DB  *db.DB
}

// NewServer creates a new Server instance.
func NewServer(cfg *config.Config, dbInst *db.DB) *Server {
	return &Server{
		Cfg: cfg,
		DB:  dbInst,
	}
}

// Router returns the HTTP request multiplexer with all routes registered.
func (s *Server) Router() *http.ServeMux {
	// Initialize MDE worker pool with configuration from environment
	workerConfig := processing.WorkerConfig{}

	// Job poll interval (how often workers check for new jobs)
	if pollInterval := os.Getenv("MDE_WORKER_POLL_INTERVAL"); pollInterval != "" {
		if secs, err := strconv.Atoi(pollInterval); err == nil && secs > 0 {
			workerConfig.JobPollInterval = time.Duration(secs) * time.Second
		}
	}
	if workerConfig.JobPollInterval == 0 {
		workerConfig.JobPollInterval = 5 * time.Second // default 5 seconds
	}

	// Max concurrent workers
	if maxJobs := os.Getenv("MDE_MAX_CONCURRENT_JOBS"); maxJobs != "" {
		if n, err := strconv.Atoi(maxJobs); err == nil && n > 0 {
			workerConfig.MaxConcurrentJobs = n
		}
	}
	if workerConfig.MaxConcurrentJobs == 0 {
		workerConfig.MaxConcurrentJobs = runtime.GOMAXPROCS(0) // default to number of CPUs
	}

	// Batch size (how many jobs to process per batch)
	if batchSize := os.Getenv("MDE_BATCH_SIZE"); batchSize != "" {
		if n, err := strconv.Atoi(batchSize); err == nil && n > 0 {
			workerConfig.BatchSize = n
		}
	}
	if workerConfig.BatchSize == 0 {
		workerConfig.BatchSize = 10 // default batch size
	}

	// Max retry attempts for failed jobs
	if maxRetry := os.Getenv("MDE_MAX_RETRY_ATTEMPTS"); maxRetry != "" {
		if n, err := strconv.Atoi(maxRetry); err == nil && n >= 0 {
			workerConfig.MaxRetryAttempts = n
		}
	}
	if workerConfig.MaxRetryAttempts == 0 {
		workerConfig.MaxRetryAttempts = 3 // default 3 retries
	}

	// Fuzzy distance limit for entity resolution
	if fuzzyLimit := os.Getenv("MDE_FUZZY_DISTANCE_LIMIT"); fuzzyLimit != "" {
		if n, err := strconv.Atoi(fuzzyLimit); err == nil && n >= 0 {
			workerConfig.FuzzyDistanceLimit = n
		}
	}
	if workerConfig.FuzzyDistanceLimit == 0 {
		workerConfig.FuzzyDistanceLimit = 3 // default fuzzy distance
	}

	// Normalization confidence threshold (0.0 to 1.0)
	if confThreshold := os.Getenv("MDE_NORMALIZATION_CONFIDENCE_THRESHOLD"); confThreshold != "" {
		if f, err := strconv.ParseFloat(confThreshold, 64); err == nil && f >= 0.0 && f <= 1.0 {
			workerConfig.NormalizationConfidenceThreshold = f
		}
	}
	if workerConfig.NormalizationConfidenceThreshold == 0.0 {
		workerConfig.NormalizationConfidenceThreshold = 0.8 // default 80% confidence
	}

	// Enable synchronous normalization (for testing)
	if syncNorm := os.Getenv("MDE_ENABLE_SYNC_NORMALIZATION"); syncNorm != "" {
		if b, err := strconv.ParseBool(syncNorm); err == nil {
			workerConfig.EnableSyncNormalization = b
		}
	}
	// default remains false

	workerPool := processing.NewWorkerPool(s.DB.Pool, workerConfig)

	// Initialize MDE processing handler with the worker pool
	mdeHandler := handlers.NewProcessingHandler(workerPool)

	// Update the router to use the properly initialized handler
	return RouterWithMDEHandler(mdeHandler)
}
