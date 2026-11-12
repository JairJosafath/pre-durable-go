package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
)

// Files looks up the catalog record of an upload.
type Files interface {
	File(ctx context.Context, owner, key string) (drive.File, error)
}

// Decider delivers a decision to the execution waiting on a callback. It
// returns drive.ErrReviewClosed when that execution has stopped waiting.
type Decider interface {
	Decide(ctx context.Context, callbackID string, decision drive.Decision) error
}

// Reviewer answers POST /reviews with a body like
// {"key": "users/<sub>/beer.jpg", "approved": true, "note": "fine"}.
type Reviewer struct {
	files   Files
	decider Decider
}

// NewReviewer returns a handler that reads open reviews from files and
// delivers decisions through decider.
func NewReviewer(files Files, decider Decider) *Reviewer {
	return &Reviewer{files: files, decider: decider}
}

type reviewRequest struct {
	Key      string `json:"key"`
	Approved bool   `json:"approved"`
	Note     string `json:"note"`
}

// Handle passes the caller's decision to the durable execution that is
// suspended on this upload. The callback ID never leaves the backend: it is
// read from the caller's own catalog record.
func (r *Reviewer) Handle(
	ctx context.Context, request events.APIGatewayProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	reviewer, ok := caller(request)
	if !ok {
		return unauthorized()
	}

	var body reviewRequest
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil || body.Key == "" {
		return refuse(http.StatusBadRequest, `expected a JSON body with "key" and "approved"`)
	}

	file, err := r.files.File(ctx, reviewer, body.Key)
	if errors.Is(err, drive.ErrFileNotFound) {
		return refuse(http.StatusNotFound, "no such file")
	}

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if file.Review == nil {
		return refuse(http.StatusConflict, "this file is not waiting for review")
	}

	decision := drive.Decision{Approved: body.Approved, Reviewer: reviewer, Note: body.Note}

	return r.deliver(ctx, file.Review.CallbackID, decision)
}

// deliver sends the decision to the execution that waits on the callback.
func (r *Reviewer) deliver(
	ctx context.Context, callbackID string, decision drive.Decision,
) (events.APIGatewayProxyResponse, error) {
	// The record shows the review as open until the pipeline has acted on the
	// decision, so a second answer can arrive in between.
	err := r.decider.Decide(ctx, callbackID, decision)
	if errors.Is(err, drive.ErrReviewClosed) {
		return refuse(http.StatusConflict, "this review is already closed")
	}

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return respond(http.StatusAccepted, decision)
}
