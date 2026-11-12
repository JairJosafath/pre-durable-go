package pipeline

import (
	"errors"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
	"gdf/internal/thumbnail"
)

const (
	retryAttempts     = 4
	retryInitialDelay = 2 * time.Second
	retryBackoffRate  = 2
)

// analyzeAndThumbnail looks at the image and makes its thumbnail at the same
// time. It was the Parallel state of the state machine.
func (p *Pipeline) analyzeAndThumbnail(ctx durable.Context, upload drive.Upload) (drive.Analysis, drive.Object, error) {
	// durable.Go is the replay-safe `go` statement. The body runs on its own
	// goroutine and gets its own child context; a context belongs to the
	// goroutine it was created for, so the body must use child, never ctx.
	analysis := durable.Go(ctx, "analyze-image", func(child durable.Context) (drive.Analysis, error) {
		return p.analyze(child, upload)
	})

	// The thumbnailer is a separate Lambda function. Invoking it durably
	// checkpoints the call: the workflow can be suspended while it runs and
	// never invokes it a second time on replay.
	thumb := durable.InvokeAsync[thumbnail.Result](ctx, "generate-thumbnail",
		p.cfg.ThumbnailerFunction, thumbnail.Request{Source: upload.Object})

	// Join awaits futures of different result types. Awaiting them together,
	// rather than one Result call after the other, lets every branch reach a
	// checkpoint before the invocation is suspended.
	if err := durable.Join(ctx, "await-analysis-and-thumbnail", []durable.Awaitable{analysis, thumb}); err != nil {
		return drive.Analysis{}, drive.Object{}, err
	}

	found, err := analysis.Result(ctx)
	if err != nil {
		return drive.Analysis{}, drive.Object{}, err
	}

	made, err := thumb.Result(ctx)

	return found, made.Thumbnail, err
}

// analyze asks the vision service what is in the image and whether anything
// about it is a concern. Each call is its own step, so a failure in the
// second one never repeats the first.
func (p *Pipeline) analyze(ctx durable.Context, upload drive.Upload) (drive.Analysis, error) {
	retry := durable.WithRetry(retryTransient())

	labels, err := durable.Step(ctx, "detect-labels", func(sc durable.StepContext) ([]drive.Label, error) {
		// A step context is a context.Context: pass it to SDK and HTTP calls.
		sc.Logger().Info("detecting labels")

		return p.vision.DetectLabels(sc, upload.Object)
	}, retry)
	if err != nil {
		return drive.Analysis{}, err
	}

	flags, err := durable.Step(ctx, "detect-moderation", func(sc durable.StepContext) ([]drive.Label, error) {
		return p.vision.DetectModerationLabels(sc, upload.Object)
	}, retry)

	return drive.Analysis{Labels: labels, Flags: flags}, err
}

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
