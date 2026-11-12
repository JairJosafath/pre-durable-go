package execution

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// kinds maps the start of an event type's name, such as the "Step" of
// "StepSucceeded", to the kind of operation the event is about.
func kinds() map[string]Kind {
	return map[string]Kind{
		"Step":          KindStep,
		"Context":       KindContext,
		"Wait":          KindWait,
		"Callback":      KindCallback,
		"ChainedInvoke": KindInvoke,
	}
}

// split reads an event type such as "ChainedInvokeTimedOut" as the kind of
// operation it is about and the status it leaves that operation in.
func split(eventType types.EventType) (Kind, Status, bool) {
	for prefix, kind := range kinds() {
		phase, found := strings.CutPrefix(string(eventType), prefix)
		if !found {
			continue
		}

		status, known := statuses()[phase]

		return kind, status, known
	}

	return "", "", false
}

// operated applies an event of an operation: its start, or one of its ends.
func (f *fold) operated(event types.Event) {
	kind, status, known := split(event.EventType)
	if !known || event.Id == nil {
		return
	}

	at := aws.ToTime(event.EventTimestamp)
	operation := f.operation(event, kind, at)
	operation.Status = status
	operation.Error = message(operationFailure(event))

	// A step that starts again after a failed attempt is running once more.
	operation.EndedAt = time.Time{}
	if status != StatusRunning {
		operation.EndedAt = at
	}

	if kind == KindStep {
		f.attempted(operation, event)
	}
}

// operation returns the operation an event is about. The first event of an
// operation adds it to the history; that is when the operation started.
func (f *fold) operation(event types.Event, kind Kind, at time.Time) *Operation {
	id := aws.ToString(event.Id)

	index, seen := f.position[id]
	if !seen {
		index = len(f.history.Operations)
		f.position[id] = index
		f.history.Operations = append(f.history.Operations, Operation{
			ID:        id,
			ParentID:  aws.ToString(event.ParentId),
			Name:      aws.ToString(event.Name),
			Type:      kind,
			SubType:   aws.ToString(event.SubType),
			StartedAt: at,
			Attempts:  1,
		})
	}

	return &f.history.Operations[index]
}

// attempted counts the attempts of a step, and tells a failure that ends the
// step from one that is followed by another attempt.
//
// Every attempt records a StepStarted event, so those are counted. The
// attempt number in the retry details is used as well, where it is higher:
// Lambda counts it from one and the local runner from zero.
func (f *fold) attempted(step *Operation, event types.Event) {
	if event.EventType == types.EventTypeStepStarted {
		f.started[step.ID]++
	}

	retry := retryDetails(event)
	if retry == nil {
		step.Attempts = max(step.Attempts, f.started[step.ID])

		return
	}

	step.Attempts = max(step.Attempts, f.started[step.ID], int(retry.CurrentAttempt))

	if event.EventType == types.EventTypeStepFailed && retry.NextAttemptDelaySeconds != nil {
		step.Status = StatusRetrying
		step.EndedAt = time.Time{}
	}
}

func retryDetails(event types.Event) *types.RetryDetails {
	if details := event.StepFailedDetails; details != nil {
		return details.RetryDetails
	}

	if details := event.StepSucceededDetails; details != nil {
		return details.RetryDetails
	}

	return nil
}
