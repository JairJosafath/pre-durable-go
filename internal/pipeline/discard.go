package pipeline

import (
	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
)

// discard rejects an upload and removes it, with the thumbnail if one was
// made. REJECTED always means that the upload is gone.
func (p *Pipeline) discard(ctx durable.Context, file drive.File, thumb drive.Object, note string) (drive.File, error) {
	file.Thumbnail = ""

	// The record goes first, because writing it is also the check that this
	// run still owns the upload. When a newer upload has taken the key over,
	// the write is refused and nothing is deleted: the objects are no longer
	// this run's to remove.
	rejected, err := p.settle(ctx, "mark-rejected", file.Settled(drive.StatusRejected, note))
	if err != nil {
		return drive.File{}, err
	}

	doomed := []durable.Branch[durable.Void]{{Name: "original", Func: p.remove(file.Object)}}
	if thumb != (drive.Object{}) {
		doomed = append(doomed, durable.Branch[durable.Void]{Name: "thumbnail", Func: p.remove(thumb)})
	}

	// Parallel runs named branches side by side and waits for all of them.
	_, err = durable.Parallel(ctx, "discard-upload", doomed)
	if err != nil {
		return drive.File{}, err
	}

	return rejected, nil
}

// remove returns a branch that deletes one object.
func (p *Pipeline) remove(object drive.Object) func(durable.Context) (durable.Void, error) {
	return func(branch durable.Context) (durable.Void, error) {
		return durable.Step(branch, "delete-object", func(sc durable.StepContext) (durable.Void, error) {
			return durable.Void{}, p.objects.Delete(sc, object)
		})
	}
}
