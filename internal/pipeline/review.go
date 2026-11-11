package pipeline

import (
	"errors"

	"github.com/aws/aws-durable-execution-sdk-go/durable"

	"gdf/internal/drive"
)

// review asks a person to decide about the upload when the analysis gave a
// reason to. Uploads without concerns are approved right away.
func (p *Pipeline) review(ctx durable.Context, file drive.File, reasons []string) (drive.Decision, error) {
	if len(reasons) == 0 {
		return drive.Decision{Approved: true}, nil
	}

	ctx.Logger().Info("upload needs review", "key", file.Key, "reasons", reasons)

	// WaitForCallback hands us a callback ID, runs the submitter once to
	// publish it, and then suspends the execution. No Lambda is running while
	// the reviewer makes up their mind. When the reviewer's decision is sent
	// to that callback ID, the execution resumes right here with the decision.
	decision, err := durable.WaitForCallback[drive.Decision](ctx, "human-review",
		func(sc durable.StepContext, callbackID string) error {
			return p.catalog.SaveFile(sc, file.AwaitingReview(callbackID, reasons))
		},
		durable.WithCallbackTimeout(p.cfg.ReviewTimeout),
		durable.WithCallbackSerdes(durable.JSONSerdes),
	)

	// Durable operations fail with typed errors. A review nobody answered is
	// not a failure of the workflow; it is a decision.
	var timedOut *durable.CallbackTimeoutError
	if errors.As(err, &timedOut) {
		return drive.Decision{Approved: false, Note: "no reviewer responded in time"}, nil
	}

	return decision, err
}
