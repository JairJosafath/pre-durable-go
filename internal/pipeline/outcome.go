package pipeline

import (
	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
)

// indexConcurrency bounds how many labels are written to the catalog at once.
const indexConcurrency = 5

// publish makes an approved upload searchable by each of its labels. It was
// the Map state of the state machine.
func (p *Pipeline) publish(ctx durable.Context, file drive.File) (drive.File, error) {
	labels := file.Labels

	// Map runs the function once per item, each in its own child context with
	// its own checkpoints. If the function is interrupted halfway through,
	// the labels already indexed are not written again.
	_, err := durable.Map(ctx, "index-labels", labels,
		func(item durable.Context, label drive.Label, _ int) (durable.Void, error) {
			return durable.Step(item, "save-index-entry", func(sc durable.StepContext) (durable.Void, error) {
				return durable.Void{}, p.catalog.IndexLabel(sc, file.Upload, label)
			})
		},
		durable.WithItemNamer(func(index int) string { return labels[index].Name }),
		durable.WithMaxConcurrency(indexConcurrency),
	)
	if err != nil {
		return drive.File{}, err
	}

	return p.settle(ctx, "mark-ready", file.Settled(drive.StatusReady, ""))
}

// discard removes a rejected upload and everything derived from it.
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

	// Parallel runs named branches side by side and waits for all of them.
	_, err = durable.Parallel(ctx, "discard-upload", []durable.Branch[durable.Void]{
		{Name: "original", Func: p.remove(file.Object)},
		{Name: "thumbnail", Func: p.remove(thumb)},
	})
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
