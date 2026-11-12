package main

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"

	"gdf/internal/drive"
	history "gdf/internal/execution"
)

// nameRunStep is the step of the workflow that names its run.
const nameRunStep = "name-run"

// History returns what the execution that owns the file has done. It is the
// local stand-in for Lambda's GetDurableExecutionHistory.
//
// Under the local runner every execution has the same ARN, so the ARN in the
// file record does not tell them apart. An execution is found by the upload
// it is for and by the name of its run, which the record carries too.
//
// The runner reports when an execution suspends or ends, not while a step is
// running. Until the first report all that is known is that the execution
// has started.
func (e *executions) History(_ context.Context, file drive.File) (history.History, error) {
	e.mu.Lock()
	candidates := slices.Clone(e.recent)
	e.mu.Unlock()

	// The newest execution first: an upload can have been replaced.
	slices.Reverse(candidates)

	var starting *execution

	for _, run := range candidates {
		if run.key != file.Key {
			continue
		}

		result := run.seen.Load()

		switch {
		case result != nil && runName(result) == file.Run:
			return history.FromEvents(file.Execution, result.Events), nil
		case result == nil && starting == nil:
			starting = run
		}
	}

	if starting == nil {
		return history.History{}, fmt.Errorf("%w: %s (was the local API restarted?)", history.ErrNotFound, file.Key)
	}

	return justStarted(file, starting), nil
}

// runName is the name the execution gave its run, read from the result of
// the step that made it up.
func runName(result *durabletest.TestResult) string {
	step := result.Operation(nameRunStep)
	if step == nil || step.StepDetails == nil {
		return ""
	}

	var name string
	if err := json.Unmarshal([]byte(step.StepDetails.Result), &name); err != nil {
		return ""
	}

	return name
}

// justStarted is the history of an execution the runner has not reported on yet.
func justStarted(file drive.File, run *execution) history.History {
	return history.History{
		Execution:   history.Execution{ARN: file.Execution, Status: history.StatusRunning, StartedAt: run.started},
		Invocations: []history.Invocation{},
		Operations:  []history.Operation{},
	}
}
