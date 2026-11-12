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

// Label is something recognised in an image, with a confidence from 0 to 100.
type Label struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	// Categories group related labels, such as "Animals and Pets".
	Categories []string `json:"categories,omitempty"`
	// Aliases are other names for the same thing, such as "Kitty" for "Cat".
	Aliases []string `json:"aliases,omitempty"`
}

// SearchTerms returns every word an upload with this label can be found by:
// the label's name and its aliases.
func (l Label) SearchTerms() []string {
	return append([]string{l.Name}, l.Aliases...)
}

// File is the catalog record of an upload.
type File struct {
	Upload

	// Run names the pipeline run that owns the record. Uploading the same key
	// again starts a new run, which takes the record over; the older run can
	// then no longer change it.
	Run string `json:"run,omitempty"`

	Status    Status  `json:"status"`
	Labels    []Label `json:"labels,omitempty"`
	Thumbnail string  `json:"thumbnail,omitempty"`
	Note      string  `json:"note,omitempty"`
	// Review is set only while the upload waits for a reviewer.
	Review *Review `json:"review,omitempty"`
}

// Review is an open request for a person to decide about an upload.
type Review struct {
	// CallbackID resumes the suspended pipeline execution once the decision is in.
	CallbackID string   `json:"callbackId"`
	Reasons    []string `json:"reasons"`
}

// Decision is a reviewer's verdict on an upload.
type Decision struct {
	Approved bool   `json:"approved"`
	Reviewer string `json:"reviewer,omitempty"`
	Note     string `json:"note,omitempty"`
}

// AwaitingReview returns the record as it looks while a reviewer decides.
func (f File) AwaitingReview(callbackID string, reasons []string) File {
	f.Status = StatusPendingReview
	f.Review = &Review{CallbackID: callbackID, Reasons: reasons}

	return f
}

// Settled returns the record in its final status, with the review closed.
func (f File) Settled(status Status, note string) File {
	f.Status = status
	f.Note = note
	f.Review = nil

	return f
}
