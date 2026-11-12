package drive

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
