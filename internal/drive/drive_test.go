package drive_test

import (
	"errors"
	"slices"
	"testing"

	"gdf/internal/drive"
)

func TestParseUpload(t *testing.T) {
	t.Parallel()

	upload, err := drive.ParseUpload("uploads", "users/alice/holiday/beach.jpg")
	if err != nil {
		t.Fatalf("ParseUpload: %v", err)
	}

	if upload.Owner != "alice" || upload.Key != "users/alice/holiday/beach.jpg" {
		t.Errorf("got owner %q key %q", upload.Owner, upload.Key)
	}

	for _, key := range []string{"beach.jpg", "users/alice", "users//beach.jpg", "tmp/alice/beach.jpg"} {
		if _, err := drive.ParseUpload("uploads", key); !errors.Is(err, drive.ErrNotAnUpload) {
			t.Errorf("ParseUpload(%q) error = %v, want ErrNotAnUpload", key, err)
		}
	}
}

func TestReviewPolicy(t *testing.T) {
	t.Parallel()

	policy := drive.ReviewPolicy{MinConfidence: 80}
	confident := []drive.Label{{Name: "Cat", Confidence: 97}}
	unsure := []drive.Label{{Name: "Cat", Confidence: 61}}
	alcohol := []drive.Label{{Name: "Alcohol", Confidence: 92}}

	tests := []struct {
		name     string
		analysis drive.Analysis
		want     []string
	}{
		{"confident and clean", drive.Analysis{Labels: confident}, []string{}},
		{"flagged", drive.Analysis{Labels: confident, Flags: alcohol}, []string{"flagged as Alcohol"}},
		{"unsure", drive.Analysis{Labels: unsure}, []string{"no label reaches 80% confidence"}},
		{"nothing recognised", drive.Analysis{}, []string{"no label reaches 80% confidence"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := policy.Reasons(test.analysis); !slices.Equal(got, test.want) {
				t.Errorf("Reasons() = %q, want %q", got, test.want)
			}
		})
	}
}
