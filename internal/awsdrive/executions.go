package awsdrive

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"gdf/internal/drive"
	"gdf/internal/execution"
)

// Executions reads what durable executions have done from Lambda, which
// records every operation of an execution as it happens.
type Executions struct {
	client *lambda.Client
}

// NewExecutions returns a reader of execution histories that uses the given
// Lambda client.
func NewExecutions(client *lambda.Client) *Executions {
	return &Executions{client: client}
}

// History returns what the execution that owns the file has done so far. It
// can be asked while the execution runs: the history then ends where the
// execution is.
//
// Lambda keeps a history for the retention period of the execution. After
// that, History returns execution.ErrNotFound.
func (e *Executions) History(ctx context.Context, file drive.File) (execution.History, error) {
	pages := lambda.NewGetDurableExecutionHistoryPaginator(e.client, &lambda.GetDurableExecutionHistoryInput{
		DurableExecutionArn: aws.String(file.Execution),
		// Without the execution data, Lambda leaves the errors of failed
		// operations empty.
		IncludeExecutionData: aws.Bool(true),
	})

	var events []types.Event

	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)

		var gone *types.ResourceNotFoundException
		if errors.As(err, &gone) {
			return execution.History{}, fmt.Errorf("%w: %w", execution.ErrNotFound, err)
		}

		if err != nil {
			return execution.History{}, fmt.Errorf("get execution history: %w", err)
		}

		events = append(events, page.Events...)
	}

	return execution.FromEvents(file.Execution, events), nil
}
