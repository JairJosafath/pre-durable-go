package pipeline_test

import (
	"testing"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"

	"gdf/internal/drive"
	"gdf/internal/drive/drivetest"
)

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
