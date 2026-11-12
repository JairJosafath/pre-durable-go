package pipeline

import (
	"errors"
	"fmt"

	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
)

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
