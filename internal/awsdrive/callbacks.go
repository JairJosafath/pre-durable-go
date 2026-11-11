package awsdrive

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"gdf/internal/drive"
)

// Callbacks delivers review decisions to suspended durable executions.
type Callbacks struct {
	client *lambda.Client
}

// NewCallbacks returns a callback sender that uses the given Lambda client.
func NewCallbacks(client *lambda.Client) *Callbacks {
	return &Callbacks{client: client}
}

// Decide completes the callback an execution is waiting on. The execution
// resumes, and its WaitForCallback call returns the decision.
func (c *Callbacks) Decide(ctx context.Context, callbackID string, decision drive.Decision) error {
	payload, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("encode decision: %w", err)
	}

	_, err = c.client.SendDurableExecutionCallbackSuccess(ctx, &lambda.SendDurableExecutionCallbackSuccessInput{
		CallbackId: aws.String(callbackID),
		Result:     payload,
	})
	if err != nil {
		return fmt.Errorf("complete review callback: %w", err)
	}

	return nil
}
