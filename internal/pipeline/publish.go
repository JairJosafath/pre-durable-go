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
