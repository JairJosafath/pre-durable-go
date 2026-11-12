package execution_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"gdf/internal/execution"
)

const arn = "arn:aws:lambda:eu-west-1:123456789012:function:workflow:1/durable-execution/name/id"

// at is a moment a number of seconds into the execution.
func at(second int) *time.Time {
	return aws.Time(time.Date(2026, time.November, 12, 10, 0, second, 0, time.UTC))
}

// event is a history event of the operation with the given ID.
func event(kind types.EventType, second int, id, name, parent, subType string) types.Event {
	e := types.Event{EventType: kind, EventTimestamp: at(second), Id: aws.String(id), SubType: aws.String(subType)}

	if name != "" {
		e.Name = aws.String(name)
	}

	if parent != "" {
		e.ParentId = aws.String(parent)
	}

	return e
}

func failure(kind, text string) *types.EventError {
	return &types.EventError{Payload: &types.ErrorObject{ErrorType: aws.String(kind), ErrorMessage: aws.String(text)}}
}

// failedDetection is a failed attempt of the detect-labels step. With a
// delay, another attempt follows.
func failedDetection(second int, id, parent string, attempt int32, delay *int32) types.Event {
	e := event(types.EventTypeStepFailed, second, id, "detect-labels", parent, "Step")
	e.StepFailedDetails = &types.StepFailedDetails{
		Error:        failure("Error", "vision service unavailable"),
		RetryDetails: &types.RetryDetails{CurrentAttempt: attempt, NextAttemptDelaySeconds: delay},
	}

	return e
}

func succeededStep(second int, id, name, parent string, attempt int32) types.Event {
	e := event(types.EventTypeStepSucceeded, second, id, name, parent, "Step")
	e.StepSucceededDetails = &types.StepSucceededDetails{RetryDetails: &types.RetryDetails{CurrentAttempt: attempt}}

	return e
}

func invocation(start, end int, problem *types.EventError) types.Event {
	return types.Event{
		EventType:      types.EventTypeInvocationCompleted,
		EventTimestamp: at(end),
		InvocationCompletedDetails: &types.InvocationCompletedDetails{
			StartTimestamp: at(start), EndTimestamp: at(end), Error: problem,
		},
	}
}

// waitingForReview is the history of an upload of a beer photo, as Lambda
// records it: the label detection fails twice before it works, the
// thumbnailer is invoked next to it, and the execution then suspends on a
// callback until a reviewer answers.
func waitingForReview() []types.Event {
	return []types.Event{
		event(types.EventTypeExecutionStarted, 0, "exec", "name", "", ""),
		event(types.EventTypeStepStarted, 0, "s1", "name-run", "", "Step"),
		succeededStep(0, "s1", "name-run", "", 1),
		event(types.EventTypeContextStarted, 1, "c1", "analyze-image", "", "RunInChildContext"),
		event(types.EventTypeStepStarted, 1, "s2", "detect-labels", "c1", "Step"),
		event(types.EventTypeChainedInvokeStarted, 1, "i1", "generate-thumbnail", "", "ChainedInvoke"),
		failedDetection(2, "s2", "c1", 1, aws.Int32(2)),
		event(types.EventTypeChainedInvokeSucceeded, 2, "i1", "generate-thumbnail", "", "ChainedInvoke"),
		invocation(0, 2, nil),
		event(types.EventTypeStepStarted, 4, "s2", "detect-labels", "c1", "Step"),
		failedDetection(4, "s2", "c1", 2, aws.Int32(4)),
		invocation(4, 4, nil),
		event(types.EventTypeStepStarted, 8, "s2", "detect-labels", "c1", "Step"),
		succeededStep(9, "s2", "detect-labels", "c1", 3),
		event(types.EventTypeContextSucceeded, 9, "c1", "analyze-image", "", "RunInChildContext"),
		event(types.EventTypeContextStarted, 10, "c2", "human-review", "", "WaitForCallback"),
		event(types.EventTypeCallbackStarted, 10, "cb", "", "c2", "Callback"),
		event(types.EventTypeStepStarted, 10, "s3", "", "c2", "Step"),
		succeededStep(10, "s3", "", "c2", 1),
		invocation(8, 11, nil),
	}
}

// find returns the operation with the given ID.
func find(t *testing.T, history execution.History, id string) execution.Operation {
	t.Helper()

	for _, operation := range history.Operations {
		if operation.ID == id {
			return operation
		}
	}

	t.Fatalf("no operation %s in %+v", id, history.Operations)

	return execution.Operation{}
}

func TestExecutionWaitingForAReviewIsStillRunning(t *testing.T) {
	t.Parallel()

	history := execution.FromEvents(arn, waitingForReview())

	want := execution.Execution{ARN: arn, Status: execution.StatusRunning, StartedAt: *at(0)}
	if history.Execution != want {
		t.Errorf("execution = %+v, want %+v", history.Execution, want)
	}

	// Three invocations: the first run and one replay per retry.
	if len(history.Invocations) != 3 || !history.Invocations[2].EndedAt.Equal(*at(11)) {
		t.Errorf("invocations = %+v, want three, the last one ending at second 11", history.Invocations)
	}
}

func TestOpenCallbackIsWhatTheExecutionIsSuspendedOn(t *testing.T) {
	t.Parallel()

	history := execution.FromEvents(arn, waitingForReview())

	want := execution.Operation{
		ID: "cb", ParentID: "c2", Type: execution.KindCallback, SubType: "Callback",
		Status: execution.StatusRunning, StartedAt: *at(10), Attempts: 1,
	}
	if callback := find(t, history, "cb"); callback != want {
		t.Errorf("callback = %+v, want %+v", callback, want)
	}

	if review := find(t, history, "c2"); review.Type != execution.KindContext || review.SubType != "WaitForCallback" {
		t.Errorf("human-review = %+v, want a WaitForCallback context", review)
	}
}

func TestOperationsAreListedInTheOrderTheyStarted(t *testing.T) {
	t.Parallel()

	history := execution.FromEvents(arn, waitingForReview())

	var ids []string
	for _, operation := range history.Operations {
		ids = append(ids, operation.ID)
	}

	want := []string{"s1", "c1", "s2", "i1", "c2", "cb", "s3"}
	if len(ids) != len(want) {
		t.Fatalf("operations = %v, want %v", ids, want)
	}

	for index := range want {
		if ids[index] != want[index] {
			t.Fatalf("operations = %v, want %v", ids, want)
		}
	}
}

func TestRetriedStepCountsItsAttempts(t *testing.T) {
	t.Parallel()

	history := execution.FromEvents(arn, waitingForReview())

	// Success on the third attempt, from its first start to its success,
	// and without the error of the attempts before.
	want := execution.Operation{
		ID: "s2", ParentID: "c1", Name: "detect-labels", Type: execution.KindStep, SubType: "Step",
		Status: execution.StatusSucceeded, StartedAt: *at(1), EndedAt: *at(9), Attempts: 3,
	}
	if labels := find(t, history, "s2"); labels != want {
		t.Errorf("detect-labels = %+v, want %+v", labels, want)
	}
}

func TestInvokeOfAnotherFunctionIsAnOperationOfItsOwn(t *testing.T) {
	t.Parallel()

	history := execution.FromEvents(arn, waitingForReview())

	want := execution.Operation{
		ID: "i1", Name: "generate-thumbnail", Type: execution.KindInvoke, SubType: "ChainedInvoke",
		Status: execution.StatusSucceeded, StartedAt: *at(1), EndedAt: *at(2), Attempts: 1,
	}
	if invoke := find(t, history, "i1"); invoke != want {
		t.Errorf("generate-thumbnail = %+v, want %+v", invoke, want)
	}
}

func TestStepWaitingForItsNextAttemptIsRetrying(t *testing.T) {
	t.Parallel()

	// The history up to the first failure of detect-labels.
	history := execution.FromEvents(arn, waitingForReview()[:7])

	labels := find(t, history, "s2")
	if labels.Status != execution.StatusRetrying || labels.Attempts != 1 || !labels.EndedAt.IsZero() {
		t.Errorf("detect-labels = %+v, want it retrying after one attempt, not ended", labels)
	}

	if labels.Error != "Error: vision service unavailable" {
		t.Errorf("error = %q, want the failure of the last attempt", labels.Error)
	}
}

func TestStepThatGetsNoOtherAttemptHasFailed(t *testing.T) {
	t.Parallel()

	events := []types.Event{
		event(types.EventTypeStepStarted, 0, "s1", "detect-labels", "", "Step"),
		failedDetection(1, "s1", "", 1, nil),
	}

	step := execution.FromEvents(arn, events).Operations[0]
	if step.Status != execution.StatusFailed || !step.EndedAt.Equal(*at(1)) {
		t.Errorf("step = %+v, want it failed at second 1", step)
	}
}

// The local runner counts the attempts in the retry details from zero.
func TestAttemptsCountFromOneWhateverTheRecorderCountsFrom(t *testing.T) {
	t.Parallel()

	events := []types.Event{
		event(types.EventTypeStepStarted, 0, "s1", "name-run", "", "Step"),
		succeededStep(0, "s1", "name-run", "", 0),
	}

	if step := execution.FromEvents(arn, events).Operations[0]; step.Attempts != 1 {
		t.Errorf("attempts = %d, want 1 for a step that worked at once", step.Attempts)
	}
}

func TestMapIterationsHangUnderTheirMap(t *testing.T) {
	t.Parallel()

	events := []types.Event{
		event(types.EventTypeContextStarted, 0, "m", "index-labels", "", "Map"),
		event(types.EventTypeContextStarted, 0, "m1", "Cat", "m", "MapIteration"),
		event(types.EventTypeStepStarted, 0, "s1", "save-index-entry", "m1", "Step"),
		succeededStep(1, "s1", "save-index-entry", "m1", 1),
		event(types.EventTypeContextSucceeded, 1, "m1", "Cat", "m", "MapIteration"),
		event(types.EventTypeContextSucceeded, 1, "m", "index-labels", "", "Map"),
	}

	history := execution.FromEvents(arn, events)

	iteration := find(t, history, "m1")
	if iteration.ParentID != "m" || iteration.Name != "Cat" || iteration.Status != execution.StatusSucceeded {
		t.Errorf("iteration = %+v, want Cat, succeeded, inside the map", iteration)
	}

	if step := find(t, history, "s1"); step.ParentID != "m1" {
		t.Errorf("step = %+v, want it inside its iteration", step)
	}
}

func TestFailedInvokeAndItsContextCarryTheirErrors(t *testing.T) {
	t.Parallel()

	refused := event(types.EventTypeChainedInvokeFailed, 1, "i1", "generate-thumbnail", "", "ChainedInvoke")
	refused.ChainedInvokeFailedDetails = &types.ChainedInvokeFailedDetails{
		Error: failure("TooManyRequestsException", "Rate Exceeded."),
	}

	timedOut := event(types.EventTypeCallbackTimedOut, 2, "cb", "", "c1", "Callback")
	timedOut.CallbackTimedOutDetails = &types.CallbackTimedOutDetails{Error: failure("CallbackTimeoutError", "")}

	history := execution.FromEvents(arn, []types.Event{
		event(types.EventTypeChainedInvokeStarted, 0, "i1", "generate-thumbnail", "", "ChainedInvoke"),
		refused,
		event(types.EventTypeCallbackStarted, 1, "cb", "", "c1", "Callback"),
		timedOut,
	})

	invoke := find(t, history, "i1")
	if invoke.Status != execution.StatusFailed || invoke.Error != "TooManyRequestsException: Rate Exceeded." {
		t.Errorf("invoke = %+v, want it failed with the throttling error", invoke)
	}

	callback := find(t, history, "cb")
	if callback.Status != execution.StatusTimedOut || callback.Error != "CallbackTimeoutError" {
		t.Errorf("callback = %+v, want it timed out", callback)
	}
}

func TestInvocationThatFailedKeepsItsError(t *testing.T) {
	t.Parallel()

	history := execution.FromEvents(arn, []types.Event{
		event(types.EventTypeExecutionStarted, 0, "exec", "name", "", ""),
		invocation(0, 1, failure("TooManyRequestsException", "Rate Exceeded.")),
	})

	if got := history.Invocations[0].Error; got != "TooManyRequestsException: Rate Exceeded." {
		t.Errorf("invocation error = %q, want the throttling error", got)
	}
}

func TestExecutionEndsTheWayItsLastEventSays(t *testing.T) {
	t.Parallel()

	failed := event(types.EventTypeExecutionFailed, 5, "exec", "name", "", "")
	failed.ExecutionFailedDetails = &types.ExecutionFailedDetails{Error: failure("StepError", "detect-labels failed")}

	timedOut := event(types.EventTypeExecutionTimedOut, 5, "exec", "name", "", "")
	timedOut.ExecutionTimedOutDetails = &types.ExecutionTimedOutDetails{Error: failure("", "took longer than 25h")}

	stopped := event(types.EventTypeExecutionStopped, 5, "exec", "name", "", "")
	stopped.ExecutionStoppedDetails = &types.ExecutionStoppedDetails{Error: failure("Stopped", "by an operator")}

	endings := []struct {
		last   types.Event
		status execution.Status
		reason string
	}{
		{event(types.EventTypeExecutionSucceeded, 5, "exec", "name", "", ""), execution.StatusSucceeded, ""},
		{failed, execution.StatusFailed, "StepError: detect-labels failed"},
		{timedOut, execution.StatusTimedOut, "took longer than 25h"},
		{stopped, execution.StatusStopped, "Stopped: by an operator"},
	}

	for _, ending := range endings {
		started := event(types.EventTypeExecutionStarted, 0, "exec", "name", "", "")
		got := execution.FromEvents(arn, []types.Event{started, ending.last}).Execution

		want := execution.Execution{ARN: arn, Status: ending.status, StartedAt: *at(0), EndedAt: *at(5), Error: ending.reason}
		if got != want {
			t.Errorf("after %s: execution = %+v, want %+v", ending.last.EventType, got, want)
		}

		if len(execution.FromEvents(arn, []types.Event{started, ending.last}).Operations) != 0 {
			t.Errorf("after %s: the execution itself was listed as an operation", ending.last.EventType)
		}
	}
}

// The web page reads this JSON, so its shape is pinned here.
func TestHistoryAsTheFrontendReadsIt(t *testing.T) {
	t.Parallel()

	events := []types.Event{
		event(types.EventTypeExecutionStarted, 0, "exec", "name", "", ""),
		event(types.EventTypeContextStarted, 1, "c1", "analyze-image", "", "RunInChildContext"),
		event(types.EventTypeStepStarted, 1, "s2", "detect-labels", "c1", "Step"),
		failedDetection(2, "s2", "c1", 1, aws.Int32(2)),
		invocation(0, 2, nil),
	}

	got, err := json.MarshalIndent(execution.FromEvents(arn, events), "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	const want = `{
  "execution": {
    "arn": "arn:aws:lambda:eu-west-1:123456789012:function:workflow:1/durable-execution/name/id",
    "status": "RUNNING",
    "startedAt": "2026-11-12T10:00:00Z"
  },
  "invocations": [
    {
      "startedAt": "2026-11-12T10:00:00Z",
      "endedAt": "2026-11-12T10:00:02Z"
    }
  ],
  "operations": [
    {
      "id": "c1",
      "name": "analyze-image",
      "type": "CONTEXT",
      "subType": "RunInChildContext",
      "status": "RUNNING",
      "startedAt": "2026-11-12T10:00:01Z",
      "attempts": 1
    },
    {
      "id": "s2",
      "parentId": "c1",
      "name": "detect-labels",
      "type": "STEP",
      "subType": "Step",
      "status": "RETRYING",
      "startedAt": "2026-11-12T10:00:01Z",
      "attempts": 1,
      "error": "Error: vision service unavailable"
    }
  ]
}`

	if string(got) != want {
		t.Errorf("history as JSON:\n%s\nwant:\n%s", got, want)
	}
}

func TestHistoryWithoutEventsHasEmptyLists(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(execution.FromEvents(arn, nil))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"execution":{"arn":"` + arn + `","status":"RUNNING"},"invocations":[],"operations":[]}`
	if string(got) != want {
		t.Errorf("empty history = %s, want %s", got, want)
	}
}
