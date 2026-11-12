package pipeline

import (
	"errors"

	"github.com/aws/aws-durable-execution-sdk-go/durable"
	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
)

// Pipeline processes uploads. Create one with New.
type Pipeline struct {
	vision  Vision
	catalog Catalog
	objects ObjectStore
	cfg     Config
}

// New returns a pipeline that uses the given services. It refuses a
// configuration that would only fail later, in the middle of an execution.
func New(vision Vision, catalog Catalog, objects ObjectStore, cfg Config) (*Pipeline, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &Pipeline{vision: vision, catalog: catalog, objects: objects, cfg: cfg}, nil
}

// Handle is the durable function. S3 invokes it for every uploaded object,
// and now and then twice for the same one: events are delivered at least once.
//
// The function body is replayed from the top every time the execution
// resumes. Durable operations that already completed return their recorded
// result instead of running again, so the code reads top to bottom even
// though it may run across many invocations and wait a day for a reviewer.
func (p *Pipeline) Handle(ctx durable.Context, event events.S3Event) (drive.File, error) {
	upload, err := uploadFrom(event)
	if err != nil {
		return drive.File{}, err
	}

	file, err := p.register(ctx, upload)
	if err != nil {
		return drive.File{}, err
	}

	settled, err := p.process(ctx, file)

	// Not every error is a failure. When the execution suspends, for a retry
	// delay or a reviewer, the operation it is blocked on returns a signal
	// that must be passed on unchanged. A failure is an error of one of the
	// SDK's public types, and all of those match OperationError.
	if _, ok := errors.AsType[*durable.OperationError](err); ok {
		err = p.fail(ctx, file, err)
	}

	// A run that lost its upload to a newer one has nothing left to do, and
	// nothing to report as failed.
	if errors.Is(err, drive.ErrSuperseded) {
		return p.superseded(ctx, file), nil
	}

	return settled, err
}

// process takes a registered upload through analysis, review and publishing.
func (p *Pipeline) process(ctx durable.Context, file drive.File) (drive.File, error) {
	upload := file.Upload

	if !upload.IsImage() {
		// Nothing was derived from the upload yet: there is no thumbnail.
		return p.discard(ctx, file, drive.Object{}, "not a supported image type")
	}

	analysis, thumb, err := p.analyzeAndThumbnail(ctx, upload)
	if err != nil {
		return drive.File{}, err
	}

	file.Labels = analysis.Labels
	file.Thumbnail = thumb.Key

	decision, err := p.review(ctx, file, p.cfg.ReviewPolicy.Reasons(analysis))
	if err != nil {
		return drive.File{}, err
	}

	if !decision.Approved {
		return p.discard(ctx, file, thumb, decision.Note)
	}

	return p.publish(ctx, file)
}
