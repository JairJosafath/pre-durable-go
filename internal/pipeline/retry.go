package pipeline

import (
	"errors"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
)

const (
	retryAttempts     = 4
	retryInitialDelay = 2 * time.Second
	retryBackoffRate  = 2
)

// retryTransient retries a failed step with exponential backoff, unless the
// failure is a permanent one. Between attempts the execution is suspended:
// waiting costs no compute.
func retryTransient() durable.RetryStrategy {
	backoff := durable.MustNewRetryStrategy(durable.RetryConfig{
		MaxAttempts:  retryAttempts,
		InitialDelay: retryInitialDelay,
		BackoffRate:  retryBackoffRate,
	})

	return func(attempt durable.RetryAttempt) durable.RetryDecision {
		if permanent(attempt.Err) {
			return durable.RetryDecision{Retry: false}
		}

		return backoff(attempt)
	}
}

// permanent reports whether a failure stays a failure however often the step
// is tried: an image that cannot be analysed, or a record that a newer upload
// has taken over.
func permanent(err error) bool {
	return errors.Is(err, drive.ErrUnsupportedImage) || errors.Is(err, drive.ErrSuperseded)
}
