package execution

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// message is the text of the error an event carries: its type and what it
// says, such as "TooManyRequestsException: Rate Exceeded." It is empty for
// an event without an error. Lambda only includes the text when the history
// is read with its execution data.
func message(failure *types.EventError) string {
	if failure == nil || failure.Payload == nil {
		return ""
	}

	kind := aws.ToString(failure.Payload.ErrorType)
	text := aws.ToString(failure.Payload.ErrorMessage)

	if kind == "" || text == "" {
		return kind + text
	}

	return kind + ": " + text
}

// executionFailure is the error of an event that ends the execution.
func executionFailure(event types.Event) *types.EventError {
	switch {
	case event.ExecutionFailedDetails != nil:
		return event.ExecutionFailedDetails.Error
	case event.ExecutionTimedOutDetails != nil:
		return event.ExecutionTimedOutDetails.Error
	case event.ExecutionStoppedDetails != nil:
		return event.ExecutionStoppedDetails.Error
	default:
		return nil
	}
}

// operationFailure is the error of an event that ends an operation badly.
func operationFailure(event types.Event) *types.EventError {
	switch {
	case event.StepFailedDetails != nil:
		return event.StepFailedDetails.Error
	case event.ContextFailedDetails != nil:
		return event.ContextFailedDetails.Error
	case event.WaitCancelledDetails != nil:
		return event.WaitCancelledDetails.Error
	case event.CallbackFailedDetails != nil:
		return event.CallbackFailedDetails.Error
	case event.CallbackTimedOutDetails != nil:
		return event.CallbackTimedOutDetails.Error
	default:
		return invokeFailure(event)
	}
}

// invokeFailure is the error of a call to another function that went wrong.
func invokeFailure(event types.Event) *types.EventError {
	switch {
	case event.ChainedInvokeFailedDetails != nil:
		return event.ChainedInvokeFailedDetails.Error
	case event.ChainedInvokeTimedOutDetails != nil:
		return event.ChainedInvokeTimedOutDetails.Error
	case event.ChainedInvokeStoppedDetails != nil:
		return event.ChainedInvokeStoppedDetails.Error
	default:
		return nil
	}
}
