package execution

import (
	"errors"
	"time"
)

// ErrNotFound reports an execution whose history is not kept, or no longer:
// Lambda deletes it when the execution's retention period is over.
var ErrNotFound = errors.New("execution history not found")

// Status is how far an execution or one of its operations got.
type Status string

// The statuses of an execution and of an operation. Retrying is a step that
// failed and waits for its next attempt; only steps are retried.
const (
	StatusRunning   Status = "RUNNING"
	StatusRetrying  Status = "RETRYING"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
	StatusTimedOut  Status = "TIMED_OUT"
	StatusStopped   Status = "STOPPED"
	StatusCancelled Status = "CANCELLED"
)

// Kind is the kind of durable operation.
type Kind string

// The kinds of operation. A context groups the operations of durable.Go,
// Map, Parallel and WaitForCallback; an invoke calls another function.
const (
	KindStep     Kind = "STEP"
	KindContext  Kind = "CONTEXT"
	KindWait     Kind = "WAIT"
	KindCallback Kind = "CALLBACK"
	KindInvoke   Kind = "INVOKE"
)

// History is everything recorded about one durable execution.
type History struct {
	Execution Execution `json:"execution"`
	// Invocations are the times Lambda ran the handler for this execution,
	// oldest first. Every one after the first is a replay.
	Invocations []Invocation `json:"invocations"`
	// Operations are in the order in which they first started.
	Operations []Operation `json:"operations"`
}

// Execution is the execution as a whole.
type Execution struct {
	ARN       string    `json:"arn"`
	Status    Status    `json:"status"`
	StartedAt time.Time `json:"startedAt,omitzero"`
	EndedAt   time.Time `json:"endedAt,omitzero"`
	Error     string    `json:"error,omitempty"`
}

// Invocation is one run of the handler, from the top of the function to its
// end or to the point where the execution suspended.
type Invocation struct {
	StartedAt time.Time `json:"startedAt,omitzero"`
	EndedAt   time.Time `json:"endedAt,omitzero"`
	Error     string    `json:"error,omitempty"`
}

// Operation is one durable operation of the workflow.
type Operation struct {
	ID string `json:"id"`
	// ParentID is the context the operation runs in. Operations of the
	// handler itself have none.
	ParentID string `json:"parentId,omitempty"`
	// Name is the name the workflow gave the operation. Operations the SDK
	// starts on its own, such as the callback inside WaitForCallback, have none.
	Name string `json:"name"`
	Type Kind   `json:"type"`
	// SubType says what a context is for: "Map", "MapIteration",
	// "WaitForCallback" and so on.
	SubType   string    `json:"subType,omitempty"`
	Status    Status    `json:"status"`
	StartedAt time.Time `json:"startedAt,omitzero"`
	EndedAt   time.Time `json:"endedAt,omitzero"`
	// Attempts is how often the operation was tried, counting the attempt
	// that is running.
	Attempts int    `json:"attempts"`
	Error    string `json:"error,omitempty"`
}
