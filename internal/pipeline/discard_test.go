package pipeline_test

import (
	"testing"

	"gdf/internal/drive"
)

func TestOtherFileTypesAreRejectedAndRemovedWithoutAnalysis(t *testing.T) {
	t.Parallel()

	notes := drive.Object{Bucket: uploadBucket, Key: "users/alice/notes.pdf"}

	vision := aCat()
	h := newHarness(t, vision)
	h.objects.Add(notes, []byte("%PDF-1.7"))

	file := succeededWith(t, h.run(t, notes.Key))

	if file.Status != drive.StatusRejected || vision.Calls() != 0 {
		t.Errorf("status = %s after %d vision calls, want REJECTED after none", file.Status, vision.Calls())
	}

	// REJECTED means gone, whoever or whatever rejected the upload.
	if h.objects.Has(notes) {
		t.Error("the rejected upload is still in the bucket")
	}
}
