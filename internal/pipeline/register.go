package pipeline

import (
	"crypto/rand"

	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
)

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
