package pipeline_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable"
	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"
	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
	"gdf/internal/drive/drivetest"
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
	vision   *drivetest.Vision
	catalog  *drivetest.Catalog
	objects  *drivetest.Objects
	workflow *pipeline.Pipeline
	runner   *durabletest.LocalRunner[events.S3Event, drive.File]
}

func newHarness(t *testing.T, vision *drivetest.Vision) *harness {
	t.Helper()

	catalog := drivetest.NewCatalog()
	objects := drivetest.NewObjects()
	objects.Add(drive.Object{Bucket: uploadBucket, Key: catPhoto}, drivetest.PNG(t, 640, 480))

	workflow, err := pipeline.New(vision, catalog, objects, pipeline.Config{
		ThumbnailerFunction: thumbnailerName,
		ReviewTimeout:       24 * time.Hour,
		ReviewPolicy:        drive.ReviewPolicy{MinConfidence: 80},
	})
	if err != nil {
		t.Fatalf("configure the pipeline: %v", err)
	}

	h := &harness{vision: vision, catalog: catalog, objects: objects, workflow: workflow}
	h.runner = h.newRunner(t)

	return h
}

// newRunner returns a runner for one execution of the workflow.
func (h *harness) newRunner(t *testing.T) *durabletest.LocalRunner[events.S3Event, drive.File] {
	t.Helper()

	// The runner takes the same options as durable.Start. Here the workflow's
	// log lines go to the test log instead of stderr.
	logs := slog.NewTextHandler(t.Output(), nil)
	runner := durabletest.NewLocalRunner(h.workflow.Handle, durable.WithLogHandler(logs))

	// In the cloud the thumbnailer is a second Lambda function. Here the same
	// code is registered under the same name, so durable.InvokeAsync really runs it.
	generator := thumbnail.NewGenerator(h.objects, thumbnailBucket)
	runner.RegisterFunction(thumbnailerName, durabletest.PlainFunction(generator.Generate))

	return runner
}

// uploadedAgain returns a second execution of the same workflow on the same
// services: what S3 starts when the same key is uploaded once more.
func (h *harness) uploadedAgain(t *testing.T) *harness {
	t.Helper()

	again := *h
	again.runner = h.newRunner(t)

	return &again
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

// endsFailed runs an upload that the pipeline cannot process, and checks
// that the owner gets to see that.
func (h *harness) endsFailed(t *testing.T, key string) {
	t.Helper()

	// The execution still fails: a failure stays visible as one.
	result := h.run(t, key)
	if result.Status != durabletest.Failed {
		t.Fatalf("status = %s, want FAILED", result.Status)
	}

	// But the owner is not left looking at PROCESSING forever.
	file, _ := h.catalog.File(key)
	if file.Status != drive.StatusFailed || file.Note == "" {
		t.Errorf("catalog record = %+v, want FAILED with a note", file)
	}

	if result.Operation("mark-failed") == nil {
		t.Errorf("the mark-failed operation never ran:\n%s", result.FormatTree())
	}
}

// leftAlone checks that a superseded run kept its hands off the upload: the
// record still has the status the newer run gave it, and nothing was deleted.
func (h *harness) leftAlone(t *testing.T, result *durabletest.TestResult, status drive.Status) {
	t.Helper()

	if file := h.file(t); file.Status != status {
		t.Errorf("catalog record = %s, want it left at %s", file.Status, status)
	}

	for _, bucket := range []string{uploadBucket, thumbnailBucket} {
		if !h.objects.Has(drive.Object{Bucket: bucket, Key: catPhoto}) {
			t.Errorf("the superseded run deleted the object in %s", bucket)
		}
	}

	if result.Operation("discard-upload") != nil {
		t.Errorf("the superseded run started deleting:\n%s", result.FormatTree())
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
