package executions_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"gdf/internal/aws/executions"
	"gdf/internal/drive"
	"gdf/internal/execution"
)

const beerExecution = "arn:aws:lambda:eu-west-1:123456789012:function:workflow:1/durable-execution/beer/1"

// The first page of a history: the execution starts and its first step runs.
const firstPage = `{
	"Events": [
		{"EventId": 1, "EventType": "ExecutionStarted", "EventTimestamp": 1794477600, "Id": "exec", "Name": "beer"},
		{"EventId": 2, "EventType": "StepStarted", "EventTimestamp": 1794477601, "Id": "s1", "Name": "name-run", "SubType": "Step"}
	],
	"NextMarker": "page-2"
}`

// The second page: the step fails for good, and the execution with it.
const secondPage = `{
	"Events": [
		{"EventId": 3, "EventType": "StepFailed", "EventTimestamp": 1794477602, "Id": "s1", "Name": "name-run", "SubType": "Step",
		 "StepFailedDetails": {
			"Error": {"Payload": {"ErrorType": "Error", "ErrorMessage": "no entropy"}},
			"RetryDetails": {"CurrentAttempt": 1}
		 }},
		{"EventId": 4, "EventType": "ExecutionFailed", "EventTimestamp": 1794477603, "Id": "exec", "Name": "beer",
		 "ExecutionFailedDetails": {"Error": {"Payload": {"ErrorType": "StepError", "ErrorMessage": "name-run failed"}}}}
	]
}`

const noSuchExecution = `{"Type": "User", "Message": "Durable Execution does not exist"}`

// executionsAnswering returns a history reader whose Lambda answers with
// answer, and the request URLs it received.
func executionsAnswering(t *testing.T, answer http.HandlerFunc) (*executions.Executions, *[]string) {
	t.Helper()

	var received []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = append(received, r.URL.RequestURI())

		w.Header().Set("Content-Type", "application/json")
		answer(w, r)
	}))
	t.Cleanup(server.Close)

	client := lambda.New(lambda.Options{
		Region:           "local",
		BaseEndpoint:     aws.String(server.URL),
		Credentials:      credentials.NewStaticCredentialsProvider("local", "local", ""),
		RetryMaxAttempts: 1,
	})

	return executions.New(client), &received
}

// inTwoPages answers with the first page, and with the second one to the
// request that asks for it by its marker.
func inTwoPages(w http.ResponseWriter, r *http.Request) {
	page := firstPage
	if r.URL.Query().Get("Marker") == "page-2" {
		page = secondPage
	}

	_, _ = io.WriteString(w, page)
}

func TestHistoryIsReadPageByPage(t *testing.T) {
	t.Parallel()

	executions, received := executionsAnswering(t, inTwoPages)

	history, err := executions.History(t.Context(), drive.File{Execution: beerExecution})
	if err != nil {
		t.Fatal(err)
	}

	if len(*received) != 2 {
		t.Fatalf("requests = %v, want one per page", *received)
	}

	// The step starts on the first page and fails on the second.
	if len(history.Operations) != 1 || history.Operations[0].Status != execution.StatusFailed {
		t.Errorf("operations = %+v, want the one step, failed", history.Operations)
	}
}

func TestHistoryComesWithTheErrorsOfWhatFailed(t *testing.T) {
	t.Parallel()

	executions, received := executionsAnswering(t, inTwoPages)

	history, err := executions.History(t.Context(), drive.File{Execution: beerExecution})
	if err != nil {
		t.Fatal(err)
	}

	// Without the execution data Lambda leaves error messages out.
	if first := (*received)[0]; !strings.Contains(first, "IncludeExecutionData=true") {
		t.Errorf("request = %s, want it to ask for the execution data", first)
	}

	if history.Execution.Error != "StepError: name-run failed" || history.Operations[0].Error != "Error: no entropy" {
		t.Errorf("errors = %q and %q, want those of the execution and of its step",
			history.Execution.Error, history.Operations[0].Error)
	}
}

func TestHistoryThatLambdaNoLongerKeepsIsNotFound(t *testing.T) {
	t.Parallel()

	executions, _ := executionsAnswering(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Amzn-Errortype", "ResourceNotFoundException")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, noSuchExecution)
	})

	_, err := executions.History(t.Context(), drive.File{Execution: beerExecution})
	if !errors.Is(err, execution.ErrNotFound) {
		t.Errorf("err = %v, want execution.ErrNotFound", err)
	}
}

func TestHistoryPassesOtherFailuresOn(t *testing.T) {
	t.Parallel()

	executions, _ := executionsAnswering(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Amzn-Errortype", "TooManyRequestsException")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"message": "Rate exceeded"}`)
	})

	_, err := executions.History(t.Context(), drive.File{Execution: beerExecution})
	if err == nil || errors.Is(err, execution.ErrNotFound) {
		t.Errorf("err = %v, want the throttling error, not a missing history", err)
	}
}
