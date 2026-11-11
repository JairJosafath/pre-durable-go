package pipeline_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable"
	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"
	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
	"gdf/internal/drivetest"
	"gdf/internal/pipeline"
	"gdf/internal/thumbnail"
)

const (
	uploadBucket    = "uploads"
	thumbnailBucket = "thumbnails"
	thumbnailerName = "thumbnailer"
	catPhoto        = "users/alice/cat.png"
)

// harness is one pipeline wired to in-memory services, plus the local runner
// that executes it. This is dependency injection doing its job: the workflow
// under test is the production code, only its collaborators are swapped.
type harness struct {
	vision  *drivetest.Vision
	catalog *drivetest.Catalog
	objects *drivetest.Objects
	runner  *durabletest.LocalRunner[events.S3Event, drive.File]
}

func newHarness(t *testing.T, vision *drivetest.Vision) *harness {
	t.Helper()

	catalog := drivetest.NewCatalog()
	objects := drivetest.NewObjects()
	objects.Add(drive.Object{Bucket: uploadBucket, Key: catPhoto}, drivetest.PNG(t, 640, 480))

	workflow := pipeline.New(vision, catalog, objects, pipeline.Config{
		ThumbnailerFunction: thumbnailerName,
		ReviewTimeout:       24 * time.Hour,
		ReviewPolicy:        drive.ReviewPolicy{MinConfidence: 80},
	})

	// The runner takes the same options as durable.Start. Here the workflow's
	// log lines go to the test log instead of stderr.
	logs := slog.NewTextHandler(t.Output(), nil)
	runner := durabletest.NewLocalRunner(workflow.Handle, durable.WithLogHandler(logs))

	// In the cloud the thumbnailer is a second Lambda function. Here the same
	// code is registered under the same name, so durable.Invoke really runs it.
	generator := thumbnail.NewGenerator(objects, thumbnailBucket)
	runner.RegisterFunction(thumbnailerName, durabletest.PlainFunction(generator.Generate))

	return &harness{vision: vision, catalog: catalog, objects: objects, runner: runner}
}

// run drives the execution until it finishes or needs something from outside.
func (h *harness) run(t *testing.T, key string) *durabletest.TestResult {
	t.Helper()

	result, err := h.runner.RunUntilComplete(uploaded(key))
	if err != nil {
		t.Fatalf("the runner itself failed: %v", err)
	}

	t.Logf("execution %s after %d invocation(s)\n%s", result.Status, len(result.Invocations), result.FormatTree())

	return result
}

// runUntilReview runs the cat photo through the pipeline until the execution
// suspends, and returns the review it is waiting on.
func (h *harness) runUntilReview(t *testing.T) *drive.Review {
	t.Helper()

	// PENDING means the execution is waiting for the outside world.
	if result := h.run(t, catPhoto); result.Status != durabletest.Pending {
		t.Fatalf("status = %s, want PENDING while the review is open", result.Status)
	}

	file := h.file(t)
	if file.Status != drive.StatusPendingReview || file.Review == nil {
		t.Fatalf("catalog record = %+v, want PENDING_REVIEW with an open review", file)
	}

	return file.Review
}

// decide answers an open review, as the reviewer Lambda does in the cloud.
func (h *harness) decide(t *testing.T, review *drive.Review, decision drive.Decision) {
	t.Helper()

	if err := h.runner.SendCallbackSuccess(review.CallbackID, decision); err != nil {
		t.Fatalf("send decision: %v", err)
	}
}

// file returns the catalog record of the cat photo.
func (h *harness) file(t *testing.T) drive.File {
	t.Helper()

	file, ok := h.catalog.File(catPhoto)
	if !ok {
		t.Fatal("the upload was never registered in the catalog")
	}

	return file
}

// succeededWith returns the execution's result, and fails the test with the
// recorded error if the execution did not succeed.
func succeededWith(t *testing.T, result *durabletest.TestResult) drive.File {
	t.Helper()

	file, err := durabletest.ResultAs[drive.File](result)
	if err != nil {
		t.Fatalf("execution did not succeed: %v", err)
	}

	return file
}

// uploaded is the event S3 sends when an object is created.
func uploaded(key string) events.S3Event {
	return events.S3Event{Records: []events.S3EventRecord{{
		S3: events.S3Entity{
			Bucket: events.S3Bucket{Name: uploadBucket},
			Object: events.S3Object{Key: key},
		},
	}}}
}

func aCat() *drivetest.Vision {
	return &drivetest.Vision{Labels: []drive.Label{
		{Name: "Cat", Confidence: 98, Categories: []string{"Animals and Pets"}, Aliases: []string{"Kitty"}},
		{Name: "Pet", Confidence: 91, Categories: []string{"Animals and Pets"}},
	}}
}

func aBeer() *drivetest.Vision {
	return &drivetest.Vision{
		Labels: []drive.Label{{Name: "Beer", Confidence: 96}},
		Flags:  []drive.Label{{Name: "Alcohol", Confidence: 93}},
	}
}
