package drive

// Status is where an upload is on its way through the pipeline.
type Status string

// The statuses an upload moves through. PendingReview is skipped when the
// analysis raises no concerns.
const (
	StatusProcessing    Status = "PROCESSING"
	StatusPendingReview Status = "PENDING_REVIEW"
	StatusReady         Status = "READY"
	StatusRejected      Status = "REJECTED"
)

// Label is something recognised in an image, with a confidence from 0 to 100.
type Label struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	// Categories group related labels, such as "Animals and Pets".
	Categories []string `json:"categories,omitempty"`
	// Aliases are other names for the same thing, such as "Kitty" for "Cat".
	Aliases []string `json:"aliases,omitempty"`
}

// File is the catalog record of an upload.
type File struct {
	Upload

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
