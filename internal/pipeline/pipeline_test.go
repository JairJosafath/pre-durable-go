package pipeline_test

// These tests run the real durable function, with real checkpointing, replay
// and suspension, entirely in this process. There is no AWS account, no
// credentials, no network and no container involved: durabletest provides an
// in-memory stand-in for the durable execution service, and drivetest
// provides in-memory stand-ins for S3, DynamoDB and Rekognition.

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"

	"gdf/internal/drive"
	"gdf/internal/drivetest"
)

func TestConfidentUploadIsPublished(t *testing.T) {
	t.Parallel()

	h := newHarness(t, aCat())
	result := h.run(t, catPhoto)
	file := succeededWith(t, result)

	if file.Status != drive.StatusReady || len(file.Labels) != 2 {
		t.Errorf("result = %+v, want READY with 2 labels", file)
	}

	for _, label := range []string{"Cat", "Pet"} {
		if keys := h.catalog.Keys(label); !slices.Equal(keys, []string{catPhoto}) {
			t.Errorf("index[%s] = %v, want the cat photo", label, keys)
		}
	}

	if !h.objects.Has(drive.Object{Bucket: thumbnailBucket, Key: catPhoto}) {
		t.Error("no thumbnail was stored")
	}

	// Nothing about the cat is questionable, so nobody was asked.
	if result.Operation("human-review") != nil {
		t.Error("a confident, unflagged upload should not go to review")
	}

	// The shape of the execution is pinned in a golden file. Parallel
	// branches may checkpoint in any order, so the comparison ignores order.
	// Regenerate with: UPDATE_GOLDEN=1 go test ./internal/pipeline/
	durabletest.AssertGoldenSignatureUnordered(t, result, filepath.Join("testdata", "published.golden.json"))
}

func TestUploadsAreFoundByLabelAliases(t *testing.T) {
	t.Parallel()

	h := newHarness(t, aCat())
	succeededWith(t, h.run(t, catPhoto))

	// The label itself is catalogued once, with its aliases, so the labels a
	// user has can be listed without reading every indexed file.
	if cat, ok := h.catalog.Label("Cat"); !ok || !slices.Equal(cat.Aliases, []string{"Kitty"}) {
		t.Errorf("label Cat = %+v, want it catalogued with its alias", cat)
	}

	// The photo is indexed under the alias too: searching "Kitty" is the same
	// single lookup as searching "Cat".
	if keys := h.catalog.Keys("Kitty"); !slices.Equal(keys, []string{catPhoto}) {
		t.Errorf("index[Kitty] = %v, want the cat photo", keys)
	}
}

func TestFlakyVisionIsRetriedWithoutRealWaiting(t *testing.T) {
	t.Parallel()

	vision := aCat()
	vision.FailFirst(2)

	h := newHarness(t, vision)

	started := time.Now()
	result := h.run(t, catPhoto)
	succeededWith(t, result)

	// Two attempts failed, the third one succeeded.
	if step := result.Operation("detect-labels"); step == nil || vision.Calls() != 3 {
		t.Errorf("detect-labels = %+v after %d calls, want success on the third call", step, vision.Calls())
	}

	// The retry strategy backs off for seconds between attempts. The runner
	// keeps a virtual clock, so the test does not sit through the backoff.
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Errorf("test took %s; the backoff should be virtual", elapsed)
	}
}

func TestBrokenImageIsNotRetried(t *testing.T) {
	t.Parallel()

	vision := &drivetest.Vision{Err: drive.ErrUnsupportedImage}
	h := newHarness(t, vision)

	if result := h.run(t, catPhoto); result.Status != durabletest.Failed {
		t.Fatalf("status = %s, want FAILED", result.Status)
	}

	if vision.Calls() != 1 {
		t.Errorf("vision was called %d times; an unsupported image must not be retried", vision.Calls())
	}
}

func TestFailedAnalysisMarksTheUploadFailed(t *testing.T) {
	t.Parallel()

	h := newHarness(t, &drivetest.Vision{Err: drive.ErrUnsupportedImage})

	h.endsFailed(t, catPhoto)
}

func TestFailedThumbnailMarksTheUploadFailed(t *testing.T) {
	t.Parallel()

	const brokenPhoto = "users/alice/broken.png"

	h := newHarness(t, aCat())
	h.objects.Add(drive.Object{Bucket: uploadBucket, Key: brokenPhoto}, []byte("not an image"))

	h.endsFailed(t, brokenPhoto)
}

func TestOtherFileTypesAreRejectedWithoutAnalysis(t *testing.T) {
	t.Parallel()

	vision := aCat()
	h := newHarness(t, vision)
	file := succeededWith(t, h.run(t, "users/alice/notes.pdf"))

	if file.Status != drive.StatusRejected || vision.Calls() != 0 {
		t.Errorf("status = %s after %d vision calls, want REJECTED after none", file.Status, vision.Calls())
	}
}

func TestFlaggedUploadWaitsForReviewerAndIsApproved(t *testing.T) {
	t.Parallel()

	vision := aBeer()
	h := newHarness(t, vision)

	// First run: everything up to the review happens, then the execution
	// suspends. The workflow has published its callback ID in the catalog,
	// which is where the review API finds it.
	review := h.runUntilReview(t)
	if want := []string{"flagged as Alcohol"}; !slices.Equal(review.Reasons, want) {
		t.Errorf("review reasons = %q, want %q", review.Reasons, want)
	}

	// The reviewer approves. In the cloud the reviewer Lambda does this with
	// SendDurableExecutionCallbackSuccess.
	h.decide(t, review, drive.Decision{Approved: true, Reviewer: "bob"})

	// Second run: the function replays from the top and continues after the review.
	result := h.run(t, catPhoto)
	file := succeededWith(t, result)

	if file.Status != drive.StatusReady || h.file(t).Review != nil {
		t.Errorf("result = %+v, want READY with the review closed", file)
	}

	// Replay does not repeat work: the function body ran more than once, but
	// the vision service was asked exactly once.
	if len(result.Invocations) < 2 || vision.Calls() != 1 {
		t.Errorf("%d invocations, %d vision calls; want several invocations and one call",
			len(result.Invocations), vision.Calls())
	}
}

func TestRejectedUploadIsRemoved(t *testing.T) {
	t.Parallel()

	h := newHarness(t, aBeer())
	decision := drive.Decision{Approved: false, Reviewer: "bob", Note: "not for this drive"}

	h.decide(t, h.runUntilReview(t), decision)

	result := h.run(t, catPhoto)
	file := succeededWith(t, result)

	if file.Status != drive.StatusRejected || file.Note != decision.Note {
		t.Errorf("result = %+v, want REJECTED with the reviewer's note", file)
	}

	for _, bucket := range []string{uploadBucket, thumbnailBucket} {
		if h.objects.Has(drive.Object{Bucket: bucket, Key: catPhoto}) {
			t.Errorf("%s still holds the rejected upload", bucket)
		}
	}

	if keys := h.catalog.Keys("Beer"); len(keys) != 0 {
		t.Errorf("a rejected upload was indexed: %v", keys)
	}

	if result.Operation("discard-upload") == nil {
		t.Errorf("the discard-upload operation never ran:\n%s", result.FormatTree())
	}
}

func TestUnansweredReviewRejectsTheUpload(t *testing.T) {
	t.Parallel()

	h := newHarness(t, aBeer())
	review := h.runUntilReview(t)

	// Nobody reacts for 24 hours. The test does not wait: it tells the runner
	// that the callback's timeout has passed.
	if err := h.runner.TimeoutCallback(review.CallbackID); err != nil {
		t.Fatalf("time out the review: %v", err)
	}

	// A review that times out is a decision, not a failure of the workflow.
	if file := succeededWith(t, h.run(t, catPhoto)); file.Status != drive.StatusRejected {
		t.Errorf("status = %s, want REJECTED", file.Status)
	}
}

func TestAbandonedReviewDoesNotRemoveANewerUpload(t *testing.T) {
	t.Parallel()

	first := newHarness(t, aBeer())
	abandoned := first.runUntilReview(t)

	// The same key is uploaded again while the first run waits. The second
	// run takes the file over, and its review is the one that gets answered.
	second := first.uploadedAgain(t)
	second.decide(t, second.runUntilReview(t), drive.Decision{Approved: true, Reviewer: "bob"})

	if file := succeededWith(t, second.run(t, catPhoto)); file.Status != drive.StatusReady {
		t.Fatalf("second run = %s, want READY", file.Status)
	}

	// A day later the first run's review times out. Before, that run rejected
	// the upload and deleted the object the second run had just published.
	if err := first.runner.TimeoutCallback(abandoned.CallbackID); err != nil {
		t.Fatalf("time out the abandoned review: %v", err)
	}

	result := first.run(t, catPhoto)
	if file := succeededWith(t, result); file.Status != drive.StatusSuperseded {
		t.Errorf("first run = %s, want SUPERSEDED", file.Status)
	}

	first.leftAlone(t, result, drive.StatusReady)
}

func TestRunOvertakenDuringAnalysisLeavesTheUploadAlone(t *testing.T) {
	t.Parallel()

	vision := aBeer()
	h := newHarness(t, vision)

	// While this run analyses the image, the same key is uploaded again and
	// the newer run registers itself.
	newer := drive.File{
		Upload: drive.Upload{Object: drive.Object{Bucket: uploadBucket, Key: catPhoto}, Owner: "alice"},
		Run:    "a newer run",
		Status: drive.StatusProcessing,
	}
	vision.OnDetect = func() { _ = h.catalog.RegisterFile(t.Context(), newer) }

	// The run does not wait for a review nobody can answer: its callback ID
	// never reached the catalog. It ends at once, and not as a failure.
	result := h.run(t, catPhoto)
	if file := succeededWith(t, result); file.Status != drive.StatusSuperseded {
		t.Errorf("status = %s, want SUPERSEDED", file.Status)
	}

	if vision.Calls() != 1 {
		t.Errorf("vision was called %d times, want 1", vision.Calls())
	}

	h.leftAlone(t, result, drive.StatusProcessing)

	if file := h.file(t); file.Run != newer.Run || file.Review != nil {
		t.Errorf("catalog record = %+v, want the newer run's record untouched", file)
	}
}
