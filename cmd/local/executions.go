package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"
	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
)

// remembered is how many executions the local API keeps the history of.
// Lambda keeps them for days; here the oldest ones make room.
const remembered = 500

var errNoSuchCallback = errors.New("no execution is waiting on this callback (was the local API restarted?)")

// executions plays the part of the Lambda durable execution service for the
// local API: it starts one execution per upload, keeps the suspended ones in
// memory, resumes the right one when a review decision arrives, and
// remembers what each one did.
type executions struct {
	newRunner func() *runner

	mu   sync.Mutex
	open []*execution
	// recent holds the latest executions, finished ones too, oldest first.
	recent []*execution
}

// execution is one run of the workflow. A local runner must not be used from
// two goroutines at once, so every use holds the lock.
type execution struct {
	mu     sync.Mutex
	runner *runner
	event  events.S3Event

	// key is the object key of the upload the execution is for.
	key     string
	started time.Time
	// seen is what the runner reported when the execution last suspended or
	// ended. It is read while the runner is busy, so it is not under mu.
	seen atomic.Pointer[durabletest.TestResult]
}

func newExecutions(newRunner func() *runner) *executions {
	return &executions{newRunner: newRunner}
}

// Start runs the workflow for an upload in the background, as S3 does in AWS
// when it invokes the durable function with an ObjectCreated event.
func (e *executions) Start(upload drive.Upload) {
	run := &execution{runner: e.newRunner(), event: uploadedEvent(upload), key: upload.Key, started: time.Now()}

	e.mu.Lock()
	e.open = append(e.open, run)
	e.recent = append(e.recent, run)
	e.recent = e.recent[max(0, len(e.recent)-remembered):]
	e.mu.Unlock()

	go e.advance(run)
}

// Decide delivers a review decision to the execution that waits for it and
// lets that execution continue in the background. It is the local stand-in
// for Lambda's SendDurableExecutionCallbackSuccess.
func (e *executions) Decide(_ context.Context, callbackID string, decision drive.Decision) error {
	e.mu.Lock()
	candidates := slices.Clone(e.open)
	e.mu.Unlock()

	for _, run := range candidates {
		waiting, err := run.resolve(callbackID, decision)
		if err != nil {
			return err
		}

		if waiting {
			go e.advance(run)

			return nil
		}
	}

	return fmt.Errorf("%w: %s", errNoSuchCallback, callbackID)
}

// advance runs an execution until it ends or suspends to wait for a review.
func (e *executions) advance(run *execution) {
	run.mu.Lock()
	result, err := run.runner.RunUntilComplete(run.event)
	run.mu.Unlock()

	if err != nil {
		slog.Error("run workflow", "error", err)
		e.forget(run)

		return
	}

	run.seen.Store(result)

	fmt.Printf("\nExecution %s after %d invocation(s):\n\n%s\n", result.Status, len(result.Invocations), result.FormatTree())

	if result.Status != durabletest.Pending {
		e.forget(run)
	}
}

func (e *executions) forget(run *execution) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.open = slices.DeleteFunc(e.open, func(other *execution) bool { return other == run })
}

// resolve completes the callback if this execution is the one waiting on it.
func (x *execution) resolve(callbackID string, decision drive.Decision) (bool, error) {
	x.mu.Lock()
	defer x.mu.Unlock()

	for _, callback := range x.runner.OpenCallbacks() {
		if callback.CallbackID != callbackID {
			continue
		}

		if err := x.runner.SendCallbackSuccess(callbackID, decision); err != nil {
			return false, fmt.Errorf("send decision: %w", err)
		}

		return true, nil
	}

	return false, nil
}
