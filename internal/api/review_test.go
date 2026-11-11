package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/api"
	"gdf/internal/drive"
)

// files is a catalog with one upload, owned by alice, that waits for review.
type files struct{}

func (files) File(_ context.Context, owner, key string) (drive.File, error) {
	if owner != "alice" || key != "users/alice/beer.jpg" {
		return drive.File{}, drive.ErrFileNotFound
	}

	return drive.File{Review: &drive.Review{CallbackID: "callback-1"}}, nil
}

// decisions records what was sent to which callback.
type decisions map[string]drive.Decision

func (d decisions) Decide(_ context.Context, callbackID string, decision drive.Decision) error {
	d[callbackID] = decision

	return nil
}

func reviewBy(subject string) events.APIGatewayProxyRequest {
	return events.APIGatewayProxyRequest{
		Body: `{"key": "users/alice/beer.jpg", "approved": true}`,
		RequestContext: events.APIGatewayProxyRequestContext{
			Authorizer: map[string]any{"claims": map[string]any{"sub": subject}},
		},
	}
}

func TestReviewerCompletesTheCallbackOfTheCallersOwnFile(t *testing.T) {
	t.Parallel()

	sent := decisions{}
	reviewer := api.NewReviewer(files{}, sent)

	response, err := reviewer.Handle(t.Context(), reviewBy("alice"))
	if err != nil || response.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, err = %v, want 202", response.StatusCode, err)
	}

	want := drive.Decision{Approved: true, Reviewer: "alice"}
	if sent["callback-1"] != want {
		t.Errorf("sent = %+v, want %+v to callback-1", sent, want)
	}
}

func TestReviewerCannotReachAnotherUsersFile(t *testing.T) {
	t.Parallel()

	sent := decisions{}
	reviewer := api.NewReviewer(files{}, sent)

	response, err := reviewer.Handle(t.Context(), reviewBy("mallory"))
	if err != nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, err = %v, want 404", response.StatusCode, err)
	}

	if len(sent) != 0 {
		t.Errorf("a decision was sent for a file the caller does not own: %+v", sent)
	}
}
