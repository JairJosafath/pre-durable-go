// Package pipeline is the PrimeDrive upload workflow, written as a durable
// function. It replaces the Step Functions state machine of the original
// project: the same stages, expressed as ordinary Go.
//
// The workflow only talks to the outside world through the small interfaces
// declared here. Production wires in AWS services, the tests wire in
// in-memory fakes, and the local runner wires in open-source stand-ins.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable"
	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
)

// Vision recognises what is in an image.
type Vision interface {
	DetectLabels(ctx context.Context, image drive.Object) ([]drive.Label, error)
	DetectModerationLabels(ctx context.Context, image drive.Object) ([]drive.Label, error)
}

// Catalog records uploads and makes them searchable by label. Both methods
// overwrite what is there, so calling one twice has the same effect as
// calling it once.
type Catalog interface {
	SaveFile(ctx context.Context, file drive.File) error
	IndexLabel(ctx context.Context, upload drive.Upload, label drive.Label) error
}

// ObjectStore removes objects that must not be kept.
type ObjectStore interface {
	Delete(ctx context.Context, object drive.Object) error
}

// Config holds the settings that differ between environments.
type Config struct {
	// ThumbnailerFunction is the name or ARN of the function that makes thumbnails.
	ThumbnailerFunction string
	// ReviewTimeout is how long an upload may wait for a reviewer.
	ReviewTimeout time.Duration
	// ReviewPolicy decides which uploads need a reviewer at all.
	ReviewPolicy drive.ReviewPolicy
}

// Pipeline processes uploads. Create one with New.
type Pipeline struct {
	vision  Vision
	catalog Catalog
	objects ObjectStore
	cfg     Config
}

// New returns a pipeline that uses the given services.
func New(vision Vision, catalog Catalog, objects ObjectStore, cfg Config) *Pipeline {
	return &Pipeline{vision: vision, catalog: catalog, objects: objects, cfg: cfg}
}

var errUnexpectedEvent = errors.New("expected an S3 event with exactly one record")

// Handle is the durable function. S3 invokes it once per uploaded object.
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

	file := drive.File{Upload: upload, Status: drive.StatusProcessing}
	if err := p.save(ctx, "register-upload", file); err != nil {
		return drive.File{}, err
	}

	if !upload.IsImage() {
		return p.settle(ctx, "reject-unsupported", file.Settled(drive.StatusRejected, "not a supported image type"))
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

// uploadFrom reads the uploaded object out of the S3 event. It is plain,
// deterministic code, so it needs no step: replaying it gives the same answer.
func uploadFrom(event events.S3Event) (drive.Upload, error) {
	if len(event.Records) != 1 {
		return drive.Upload{}, fmt.Errorf("%w: got %d", errUnexpectedEvent, len(event.Records))
	}

	entity := event.Records[0].S3

	return drive.ParseUpload(entity.Bucket.Name, entity.Object.URLDecodedKey)
}

// save writes the catalog record in a step. A step runs at least once, and
// its outcome is checkpointed: on replay the write is skipped.
func (p *Pipeline) save(ctx durable.Context, step string, file drive.File) error {
	_, err := durable.Step(ctx, step, func(sc durable.StepContext) (durable.Void, error) {
		return durable.Void{}, p.catalog.SaveFile(sc, file)
	})

	return err
}

// settle writes the final record and returns it as the execution's result.
func (p *Pipeline) settle(ctx durable.Context, step string, file drive.File) (drive.File, error) {
	if err := p.save(ctx, step, file); err != nil {
		return drive.File{}, err
	}

	// The context logger is replay-aware: this line is written once, however
	// often the function is replayed.
	ctx.Logger().Info("upload settled", "key", file.Key, "status", file.Status)

	return file, nil
}
