# Real-time Batch Processing - FIXED

## Issue Resolved
Fixed the bug causing negative `jobs_waiting` values in the batch processing system.

## Root Cause
The sender goroutine was incorrectly decrementing the waiting count when sending jobs to the channel, causing each job to decrement the waiting count twice:
1. Once by the sender (incorrect)
2. Once by the worker when receiving the job (correct)

This led to impossible negative values like `Wait=-8` seen in the logs.

## Fix Applied
**File:** `passo/backend/internal/processor/handler.go`  
**Location:** Lines 577-580 (removed incorrect code)

**Removed:**
```go
// Decrement waiting count as we're about to send a job to the channel
if err := updateBatchJobCounts(context.Background(), batchID, 0, -1, dbInst); err != nil {
    log.Printf("failed to update batch job counts: %v", err)
}
```

**Corrected Logic:**
- Sender: Only puts jobs in channel (no count modification)
- Worker: Handles all counting when it receives a job from channel
  - `updateBatchJobCounts(1, -1)` when starting a job (waiting--, inProgress++)
  - `updateBatchJobCounts(-1, 0)` when finishing a job (inProgress--)

## Verification
The fix ensures mathematical consistency is maintained at all times:
```
total_jobs = processed_ok + processed_fail + jobs_in_progress + jobs_waiting
```

## Behavior
- **Normal processing**: Waiting count decreases properly from N to 0
- **Batch completion**: Automatically sets both counts to 0
- **Stopping**: Remaining unsent jobs marked as failures, in-flight jobs complete normally
- **No negative counts**: Waiting count never goes below 0

## Files Unchanged
- Migration scripts (008_add_batch_progress_columns.up/.down.sql) - still correct
- GetBatchStatus endpoint - still returns real-time metrics
- All other processing logic - remains correct

The system now provides accurate real-time visibility into batch processing states:
- `jobs_in_progress`: Jobs currently being processed by LLM
- `jobs_waiting`: Jobs waiting for their turn to call LLM  
- `processed_ok`/`processed_fail`: Successfully/failed job counts
- `status`: pending/processing/completed/stopped