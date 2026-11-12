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

	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"

	"gdf/internal/drive"
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
