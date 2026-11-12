package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/api"
	"gdf/internal/drive"
	"gdf/internal/execution"
)

const catExecution = "arn:aws:lambda:eu-west-1:123456789012:function:workflow:1/durable-execution/cat/1"

// uploads is a catalog with two uploads of alice: one made by the current
// pipeline, and one from before executions were recorded.
type uploads struct{}

func (uploads) File(_ context.Context, owner, key string) (drive.File, error) {
	if owner != "alice" {
		return drive.File{}, drive.ErrFileNotFound
	}

	switch key {
	case "users/alice/cat.jpg":
		return drive.File{Run: "run-1", Execution: catExecution}, nil
	case "users/alice/old.jpg":
		return drive.File{Run: "run-0"}, nil
	default:
		return drive.File{}, drive.ErrFileNotFound
	}
}

// recorded answers with a history of one step, and notes which files it was
// asked about.
type recorded struct {
	asked []drive.File
}

func (r *recorded) History(_ context.Context, file drive.File) (execution.History, error) {
	r.asked = append(r.asked, file)

	return execution.History{
		Execution:   execution.Execution{ARN: file.Execution, Status: execution.StatusSucceeded},
		Invocations: []execution.Invocation{},
		Operations:  []execution.Operation{{ID: "1", Name: "name-run", Type: execution.KindStep, Attempts: 1}},
	}, nil
}

// expired is Lambda after the retention period of an execution.
type expired struct{}

func (expired) History(context.Context, drive.File) (execution.History, error) {
	return execution.History{}, execution.ErrNotFound
}

func historyOf(name, subject string) events.APIGatewayProxyRequest {
	request := events.APIGatewayProxyRequest{QueryStringParameters: map[string]string{}}

	if name != "" {
		request.QueryStringParameters["key"] = name
	}

	if subject != "" {
		request.RequestContext.Authorizer = map[string]any{"claims": map[string]any{"sub": subject}}
	}

	return request
}

// historyByName asks alice's history of the cat photo, naming the file one
// way or the other, and returns what the source was asked and the answer.
func historyByName(t *testing.T, name string) ([]drive.File, execution.History) {
	t.Helper()

	source := &recorded{}

	response, err := api.NewHistory(uploads{}, source).Handle(t.Context(), historyOf(name, "alice"))
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("%s: status = %d, err = %v, want 200", name, response.StatusCode, err)
	}

	var history execution.History
	if err := json.Unmarshal([]byte(response.Body), &history); err != nil {
		t.Fatalf("%s: body is not a history: %v", name, err)
	}

	return source.asked, history
}

func TestHistoryIsReadForTheExecutionInTheCallersOwnRecord(t *testing.T) {
	t.Parallel()

	// The file can be named as GET /presigned does, or by its whole key.
	for _, name := range []string{"cat.jpg", "users/alice/cat.jpg"} {
		asked, history := historyByName(t, name)

		if len(asked) != 1 || asked[0].Execution != catExecution {
			t.Errorf("%s: asked about %+v, want the execution in alice's record", name, asked)
		}

		if history.Execution.ARN != catExecution || len(history.Operations) != 1 {
			t.Errorf("%s: history = %+v, want the one recorded for the cat photo", name, history)
		}
	}
}

func TestHistoryCannotReachAnotherUsersExecution(t *testing.T) {
	t.Parallel()

	source := &recorded{}

	// Mallory names alice's file by its whole key. It is still looked up in
	// mallory's own prefix, where there is no such file.
	response, err := api.NewHistory(uploads{}, source).Handle(t.Context(), historyOf("users/alice/cat.jpg", "mallory"))
	if err != nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, err = %v, want 404", response.StatusCode, err)
	}

	if len(source.asked) != 0 {
		t.Errorf("a history was read for a file the caller does not own: %+v", source.asked)
	}
}

func TestHistoryThatDoesNotExistIsNotFound(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		name   string
		source api.Executions
	}{
		"a file that was never uploaded":             {"dog.jpg", &recorded{}},
		"an upload from before executions were kept": {"old.jpg", &recorded{}},
		"an execution past its retention period":     {"cat.jpg", expired{}},
	}

	for what, c := range cases {
		response, err := api.NewHistory(uploads{}, c.source).Handle(t.Context(), historyOf(c.name, "alice"))
		if err != nil || response.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d, err = %v, want 404", what, response.StatusCode, err)
		}

		var body map[string]string
		if err := json.Unmarshal([]byte(response.Body), &body); err != nil || body["message"] == "" {
			t.Errorf("%s: body = %s, want a message", what, response.Body)
		}
	}
}

func TestHistoryNeedsACallerAndAFile(t *testing.T) {
	t.Parallel()

	handler := api.NewHistory(uploads{}, &recorded{})

	response, err := handler.Handle(t.Context(), historyOf("cat.jpg", ""))
	if err != nil || response.StatusCode != http.StatusUnauthorized {
		t.Errorf("without a caller: status = %d, err = %v, want 401", response.StatusCode, err)
	}

	response, err = handler.Handle(t.Context(), historyOf("", "alice"))
	if err != nil || response.StatusCode != http.StatusBadRequest {
		t.Errorf("without a key: status = %d, err = %v, want 400", response.StatusCode, err)
	}
}
