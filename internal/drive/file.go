package drive

// File is the catalog record of an upload.
type File struct {
	Upload

	// Run names the pipeline run that owns the record. Uploading the same key
	// again starts a new run, which takes the record over; the older run can
	// then no longer change it.
	Run string `json:"run,omitempty"`

	// Execution is the ARN of the durable execution that owns the record. It
	// is how the history of that execution is found: which steps ran, which
	// were retried, and what it is waiting for.
	Execution string `json:"execution,omitempty"`

	Status    Status  `json:"status"`
	Labels    []Label `json:"labels,omitempty"`
	Thumbnail string  `json:"thumbnail,omitempty"`
	Note      string  `json:"note,omitempty"`
	// Review is set only while the upload waits for a reviewer.
	Review *Review `json:"review,omitempty"`
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
