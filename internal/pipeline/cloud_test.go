//go:build cloud

package pipeline_test

// The cloud runner invokes the deployed function and returns the same
// TestResult as the local runner, so the same assertions apply. It is the
// only test here that needs AWS, which is why it sits behind a build tag:
//
//	PIPELINE_FUNCTION=<WorkflowFunction output> \
//	UPLOAD_BUCKET=<UploadBucket output> \
//	UPLOAD_KEY=users/<your sub>/cat.jpg \
//	go test -tags cloud -run TestDeployed ./internal/pipeline/

import (
	"os"
	"testing"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"gdf/internal/drive"
)

const cloudTimeout = 2 * time.Minute

func TestDeployedPipelinePublishesAnUpload(t *testing.T) {
	t.Parallel()

	runner, event := deployedPipeline(t)

	result, err := runner.Run(t.Context(), event)
	if err != nil {
		t.Fatalf("the runner itself failed: %v", err)
	}

	t.Logf("execution %s\n%s", result.Status, result.FormatTree())

	if file := succeededWith(t, result); file.Status != drive.StatusReady {
		t.Errorf("status = %s, want READY", file.Status)
	}

	for _, operation := range []string{"register-upload", "detect-labels", "generate-thumbnail", "mark-ready"} {
		if result.Operation(operation) == nil {
			t.Errorf("operation %q is missing from the execution", operation)
		}
	}
}

// deployedPipeline returns a runner for the deployed function and the event
// for an object that is already in its upload bucket.
func deployedPipeline(t *testing.T) (*durabletest.CloudRunner, events.S3Event) {
	t.Helper()

	function, bucket, key := os.Getenv("PIPELINE_FUNCTION"), os.Getenv("UPLOAD_BUCKET"), os.Getenv("UPLOAD_KEY")
	if function == "" || bucket == "" || key == "" {
		t.Skip("set PIPELINE_FUNCTION, UPLOAD_BUCKET and UPLOAD_KEY to test the deployed function")
	}

	aws, err := config.LoadDefaultConfig(t.Context())
	if err != nil {
		t.Fatalf("load AWS configuration: %v", err)
	}

	runner := durabletest.NewCloudRunner(lambda.NewFromConfig(aws), function, durabletest.WithTimeout(cloudTimeout))

	return runner, events.S3Event{Records: []events.S3EventRecord{{
		S3: events.S3Entity{Bucket: events.S3Bucket{Name: bucket}, Object: events.S3Object{Key: key}},
	}}}
}
