package pipeline

import (
	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
	"gdf/internal/thumbnail"
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
