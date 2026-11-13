package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"

	"gdf/internal/api"
	"gdf/internal/drive"
	"gdf/internal/drive/drivetest"
	history "gdf/internal/execution"
	"gdf/internal/pipeline"
	"gdf/internal/thumbnail"
)

const beerPhoto = "users/alice/beer.png"

// catalogFiles lets the handlers read the in-memory catalog the workflow
// writes to, as they read DynamoDB in AWS.
type catalogFiles struct {
	catalog *drivetest.Catalog
}

func (c catalogFiles) File(_ context.Context, _, key string) (drive.File, error) {
	file, found := c.catalog.File(key)
	if !found {
		return drive.File{}, drive.ErrFileNotFound
	}

	return file, nil
}

// memoryExecutions returns the local execution service with a workflow that
// runs on in-memory services, and the catalog that workflow writes to. Every
// upload is a photo of a beer: it is flagged, so its execution suspends
// until a reviewer answers.
func memoryExecutions(t *testing.T) (*executions, *drivetest.Catalog) {
	t.Helper()

	catalog := drivetest.NewCatalog()
	objects := drivetest.NewObjects()
	objects.Add(drive.Object{Bucket: uploadBucket, Key: beerPhoto}, drivetest.PNG(t, 64, 48))

	vision := &drivetest.Vision{
		Labels: []drive.Label{{Name: "Beer", Confidence: 96}},
		Flags:  []drive.Label{{Name: "Alcohol", Confidence: 93}},
	}

	workflow, err := pipeline.New(vision, catalog, objects, pipeline.Config{
		ThumbnailerFunction: thumbnailerName,
		ReviewTimeout:       reviewTimeout,
		ReviewPolicy:        drive.ReviewPolicy{MinConfidence: defaultMinConfidence},
	})
	if err != nil {
		t.Fatalf("configure the pipeline: %v", err)
	}

	return newExecutions(func() *runner {
		local := durabletest.NewLocalRunner(workflow.Handle)
		generator := thumbnail.NewGenerator(objects, thumbnailBucket)
		local.RegisterFunction(thumbnailerName, durabletest.PlainFunction(generator.Generate))

		return local
	}), catalog
}

// historyUntil asks GET /executions for the beer photo, as alice, until the
// answer satisfies done.
func historyUntil(t *testing.T, route http.Handler, done func(history.History) bool) history.History {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		response := httptest.NewRecorder()
		route.ServeHTTP(response, request(t, http.MethodGet, "/executions?key=beer.png", "alice"))

		var seen history.History
		if response.Code == http.StatusOK {
			if err := json.Unmarshal(response.Body.Bytes(), &seen); err != nil {
				t.Fatalf("body is not a history: %v\n%s", err, response.Body)
			}

			if done(seen) {
				return seen
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("the history never got to the expected point")

	return history.History{}
}

// operation returns the operation with the given name and its presence.
func operation(seen history.History, name string) (history.Operation, bool) {
	for _, candidate := range seen.Operations {
		if candidate.Name == name {
			return candidate, true
		}
	}

	return history.Operation{}, false
}

func TestHistoryOfALocalExecutionFollowsItToItsEnd(t *testing.T) {
	t.Parallel()

	running, catalog := memoryExecutions(t)
	route := signedIn(lambdaProxy(api.NewHistory(catalogFiles{catalog}, running).Handle))

	upload, err := drive.ParseUpload(uploadBucket, beerPhoto)
	if err != nil {
		t.Fatal(err)
	}

	running.Start(upload)

	// The execution suspends on the review: still running, with an open callback.
	waiting := historyUntil(t, route, func(seen history.History) bool {
		_, reviewing := operation(seen, "human-review")

		return reviewing
	})

	if waiting.Execution.Status != history.StatusRunning {
		t.Errorf("while waiting: execution = %+v, want it running", waiting.Execution)
	}

	if named, _ := operation(waiting, nameRunStep); named.Status != history.StatusSucceeded {
		t.Errorf("while waiting: name-run = %+v, want it succeeded", named)
	}

	file, _ := catalog.File(beerPhoto)
	if err := running.Decide(t.Context(), file.Review.CallbackID, drive.Decision{Approved: true}); err != nil {
		t.Fatal(err)
	}

	// The decision resumes the execution in the background.
	ended := historyUntil(t, route, func(seen history.History) bool {
		return seen.Execution.Status == history.StatusSucceeded
	})

	if _, published := operation(ended, "mark-ready"); !published {
		t.Errorf("after approval: operations = %+v, want them to end with mark-ready", ended.Operations)
	}

	// Resuming is a replay: the handler was invoked once more.
	if len(ended.Invocations) <= len(waiting.Invocations) {
		t.Errorf("invocations: %d while waiting, %d at the end, want more at the end",
			len(waiting.Invocations), len(ended.Invocations))
	}
}

func TestHistoryOfAnExecutionThatHasNotReportedYetSaysItStarted(t *testing.T) {
	t.Parallel()

	started := time.Now()
	running := newExecutions(nil)
	running.recent = []*execution{{key: beerPhoto, started: started}}

	file := drive.File{Upload: drive.Upload{Object: drive.Object{Key: beerPhoto}}, Run: "run-1", Execution: "arn"}

	seen, err := running.History(t.Context(), file)
	if err != nil {
		t.Fatal(err)
	}

	if seen.Execution.Status != history.StatusRunning || !seen.Execution.StartedAt.Equal(started) || len(seen.Operations) != 0 {
		t.Errorf("history = %+v, want a running execution without operations", seen)
	}
}

func TestHistoryIsGoneAfterARestartOfTheLocalAPI(t *testing.T) {
	t.Parallel()

	// The record survived in the database; the execution was in memory.
	file := drive.File{Upload: drive.Upload{Object: drive.Object{Key: beerPhoto}}, Run: "run-1", Execution: "arn"}

	_, err := newExecutions(nil).History(t.Context(), file)
	if !errors.Is(err, history.ErrNotFound) {
		t.Errorf("err = %v, want the history to be not found", err)
	}
}
