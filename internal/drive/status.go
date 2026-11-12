package drive

// Status is where an upload is on its way through the pipeline.
type Status string

// The statuses an upload moves through. PendingReview is skipped when the
// analysis raises no concerns.
//
// Rejected means the upload is gone. A reviewer decided against it, nobody
// reviewed it in time, or it is not an image at all; in each case the
// original is removed, with its thumbnail.
//
// Failed means the pipeline gave up: nobody decided against the upload, it
// just could not be processed. The original is kept.
const (
	StatusProcessing    Status = "PROCESSING"
	StatusPendingReview Status = "PENDING_REVIEW"
	StatusReady         Status = "READY"
	StatusRejected      Status = "REJECTED"
	StatusFailed        Status = "FAILED"
)

// StatusSuperseded is how a pipeline run ends when a newer upload of the same
// key took the file over. It is only ever the result of a run, never stored:
// the catalog record belongs to the newer run.
const StatusSuperseded Status = "SUPERSEDED"
