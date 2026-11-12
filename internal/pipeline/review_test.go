package pipeline_test

import (
	"slices"
	"testing"

	"gdf/internal/drive"
)

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
