package thumbnail_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"gdf/internal/drive"
	"gdf/internal/drive/drivetest"
	"gdf/internal/thumbnail"
)

func TestShrinkKeepsAspectRatio(t *testing.T) {
	t.Parallel()

	wide := image.NewRGBA(image.Rect(0, 0, 1024, 512))

	if got := thumbnail.Shrink(wide, 256).Bounds().Size(); got != image.Pt(256, 128) {
		t.Errorf("size = %v, want 256x128", got)
	}

	small := image.NewRGBA(image.Rect(0, 0, 100, 50))
	if got := thumbnail.Shrink(small, 256); got != image.Image(small) {
		t.Error("an image that already fits should be returned unchanged")
	}
}

func TestGenerateStoresThumbnailUnderSameKey(t *testing.T) {
	t.Parallel()

	original := drive.Object{Bucket: "uploads", Key: "users/alice/cat.png"}
	objects := drivetest.NewObjects()
	objects.Add(original, drivetest.PNG(t, 800, 600))

	result, err := thumbnail.NewGenerator(objects, "thumbnails").Generate(t.Context(), thumbnail.Request{Source: original})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	want := drive.Object{Bucket: "thumbnails", Key: original.Key}
	if result.Thumbnail != want {
		t.Fatalf("thumbnail = %+v, want %+v", result.Thumbnail, want)
	}

	data, err := objects.Get(t.Context(), want)
	if err != nil {
		t.Fatalf("thumbnail was not stored: %v", err)
	}

	stored, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("stored thumbnail is not a PNG: %v", err)
	}

	if got := stored.Bounds().Size(); got != image.Pt(256, 192) {
		t.Errorf("stored size = %v, want 256x192", got)
	}
}
