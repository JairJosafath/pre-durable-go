package pipeline_test

import (
	"testing"

	"gdf/internal/drive"
)

func TestRecordNamesTheExecutionThatOwnsIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t, aBeer())
	h.runUntilReview(t)

	// The record is the only place the frontend can learn which execution to
	// ask Lambda about, so it carries the ARN from the first write on.
	waiting := h.file(t)
	if waiting.Execution == "" {
		t.Fatalf("record while waiting for review = %+v, want the execution's ARN in it", waiting)
	}

	h.decide(t, waiting.Review, drive.Decision{Approved: true})

	settled := succeededWith(t, h.run(t, catPhoto))
	if settled.Execution != waiting.Execution || h.file(t).Execution != waiting.Execution {
		t.Errorf("execution = %q in the result and %q in the record, want %q in both",
			settled.Execution, h.file(t).Execution, waiting.Execution)
	}
}
