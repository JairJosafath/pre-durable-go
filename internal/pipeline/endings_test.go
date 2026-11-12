package pipeline_test

import (
	"testing"

	"gdf/internal/drive"
	"gdf/internal/drivetest"
)

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
