package execution

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

const executionEvents = "Execution"

// FromEvents folds the history events of the execution with the given ARN,
// oldest first, into what the execution has done so far.
func FromEvents(arn string, events []types.Event) History {
	fold := newFold(arn)
	for _, event := range events {
		fold.add(event)
	}

	return fold.history
}

// fold is a History under construction.
type fold struct {
	history History
	// position is where each operation is in history.Operations, by its ID.
	position map[string]int
	// started counts the StepStarted events of each step.
	started map[string]int
}

func newFold(arn string) *fold {
	return &fold{
		history: History{
			Execution:   Execution{ARN: arn, Status: StatusRunning},
			Invocations: []Invocation{},
			Operations:  []Operation{},
		},
		position: map[string]int{},
		started:  map[string]int{},
	}
}

func (f *fold) add(event types.Event) {
	name := string(event.EventType)

	switch {
	case event.EventType == types.EventTypeInvocationCompleted:
		f.invoked(event)
	case strings.HasPrefix(name, executionEvents):
		f.executed(event, strings.TrimPrefix(name, executionEvents))
	default:
		f.operated(event)
	}
}

// invoked records an invocation of the handler that has ended.
func (f *fold) invoked(event types.Event) {
	details := event.InvocationCompletedDetails
	if details == nil {
		return
	}

	f.history.Invocations = append(f.history.Invocations, Invocation{
		StartedAt: aws.ToTime(details.StartTimestamp),
		EndedAt:   aws.ToTime(details.EndTimestamp),
		Error:     message(details.Error),
	})
}

// executed records the start or the end of the execution itself.
func (f *fold) executed(event types.Event, phase string) {
	execution := &f.history.Execution
	at := aws.ToTime(event.EventTimestamp)

	status, known := statuses()[phase]
	if !known {
		return
	}

	if status == StatusRunning {
		execution.StartedAt = at

		return
	}

	execution.Status = status
	execution.EndedAt = at
	execution.Error = message(executionFailure(event))
}

// statuses maps the end of an event type's name, such as the "Succeeded" of
// "StepSucceeded", to the status the event leaves behind.
func statuses() map[string]Status {
	return map[string]Status{
		"Started":   StatusRunning,
		"Succeeded": StatusSucceeded,
		"Failed":    StatusFailed,
		"TimedOut":  StatusTimedOut,
		"Stopped":   StatusStopped,
		"Cancelled": StatusCancelled,
	}
}
