package pipeline

import (
	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
)

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
