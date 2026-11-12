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
	"crypto/rand"
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

// Catalog records uploads and makes them searchable by label. Every method
// overwrites what is there, so calling one twice has the same effect as
// calling it once.
type Catalog interface {
	// RegisterFile writes the record of a new upload. The record then belongs
	// to file.Run, whichever run owned it before.
	RegisterFile(ctx context.Context, file drive.File) error
	// SaveFile overwrites the record while file.Run still owns it. Once a
	// newer upload has taken the record over, it writes nothing and returns
	// drive.ErrSuperseded.
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
	// ReviewTimeout is how long an upload may wait for a reviewer. It must be
	// at least a second: the SDK reads zero as "wait without a timeout".
	ReviewTimeout time.Duration
	// ReviewPolicy decides which uploads need a reviewer at all.
	ReviewPolicy drive.ReviewPolicy
}

// ErrInvalidConfig reports settings the pipeline cannot run with.
var ErrInvalidConfig = errors.New("invalid pipeline configuration")

func (c Config) validate() error {
	if c.ThumbnailerFunction == "" {
		return fmt.Errorf("%w: no thumbnailer function", ErrInvalidConfig)
	}

	if c.ReviewTimeout < time.Second {
		return fmt.Errorf("%w: review timeout is %s, want at least 1s", ErrInvalidConfig, c.ReviewTimeout)
	}

	return nil
}

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

var errUnexpectedEvent = errors.New("expected an S3 event with exactly one record")

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
	var failure *durable.OperationError
	if errors.As(err, &failure) {
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

// uploadFrom reads the uploaded object out of the S3 event. It is plain,
// deterministic code, so it needs no step: replaying it gives the same answer.
func uploadFrom(event events.S3Event) (drive.Upload, error) {
	if len(event.Records) != 1 {
		return drive.Upload{}, fmt.Errorf("%w: got %d", errUnexpectedEvent, len(event.Records))
	}

	entity := event.Records[0].S3

	return drive.ParseUpload(entity.Bucket.Name, entity.Object.URLDecodedKey)
}

// register claims the upload for this run and writes its first record.
func (p *Pipeline) register(ctx durable.Context, upload drive.Upload) (drive.File, error) {
	// The run's name is random, and every replay must see the same one. So it
	// is made in a step of its own: the step has no side effect to repeat,
	// and its result is checkpointed.
	run, err := durable.Step(ctx, "name-run", func(durable.StepContext) (string, error) {
		return rand.Text(), nil
	})
	if err != nil {
		return drive.File{}, err
	}

	file := drive.File{Upload: upload, Run: run, Status: drive.StatusProcessing}

	_, err = durable.Step(ctx, "register-upload", func(sc durable.StepContext) (durable.Void, error) {
		return durable.Void{}, p.catalog.RegisterFile(sc, file)
	})

	return file, err
}

// save writes the catalog record in a step. A step runs at least once, and
// its outcome is checkpointed: on replay the write is skipped.
//
// It returns drive.ErrSuperseded when a newer upload owns the record.
func (p *Pipeline) save(ctx durable.Context, step string, file drive.File) error {
	owned, err := durable.Step(ctx, step, func(sc durable.StepContext) (bool, error) {
		err := p.catalog.SaveFile(sc, file)
		// Losing the record to a newer upload is not a failure to retry. It
		// is an answer, so it is recorded as the step's result. An error
		// would come back from the step as a StepError, without its identity.
		if errors.Is(err, drive.ErrSuperseded) {
			return false, nil
		}

		return err == nil, err
	})
	if err != nil {
		return err
	}

	if !owned {
		return fmt.Errorf("%w: %s", drive.ErrSuperseded, file.Key)
	}

	return nil
}

// failedNote is what the owner reads about an upload the pipeline gave up on.
// The cause goes to the log: it can name buckets and functions.
const failedNote = "could not be processed"

// fail records that the pipeline gave up on an upload, so the catalog does
// not keep saying PROCESSING for an execution that has ended. The execution
// itself still fails, with the error that stopped it.
func (p *Pipeline) fail(ctx durable.Context, file drive.File, cause error) error {
	ctx.Logger().Error("upload failed", "key", file.Key, "error", cause)

	// If this step fails or suspends in turn, or finds the record taken over,
	// its own error is the one to return: the cause is already in the log.
	if err := p.save(ctx, "mark-failed", file.Settled(drive.StatusFailed, failedNote)); err != nil {
		return err
	}

	return cause
}

// supersededNote is the note in the result of a run that was overtaken.
const supersededNote = "replaced by a newer upload"

// superseded is the result of a run whose upload was taken over by a newer
// one. The run leaves everything as it is: the record and the objects under
// this key belong to the newer run now.
func (p *Pipeline) superseded(ctx durable.Context, file drive.File) drive.File {
	ctx.Logger().Info("upload superseded", "key", file.Key)

	return file.Settled(drive.StatusSuperseded, supersededNote)
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
