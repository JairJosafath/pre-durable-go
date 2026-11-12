// Package thumbnail makes small previews of uploaded images. It runs as its
// own Lambda function, which the upload pipeline invokes durably.
package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"image"

	"gdf/internal/drive"
)

// MaxSide is the length in pixels of a thumbnail's longest side.
const MaxSide = 256

// Request asks for a thumbnail of one image.
type Request struct {
	Source drive.Object `json:"source"`
}

// Result says where the thumbnail was stored.
type Result struct {
	Thumbnail drive.Object `json:"thumbnail"`
}

// Store reads originals and writes thumbnails.
type Store interface {
	Get(ctx context.Context, object drive.Object) ([]byte, error)
	Put(ctx context.Context, object drive.Object, data []byte, contentType string) error
}

// Generator makes thumbnails and stores them in one bucket, under the same
// key as the original.
type Generator struct {
	store  Store
	bucket string
}

// NewGenerator returns a generator that stores thumbnails in bucket.
func NewGenerator(store Store, bucket string) *Generator {
	return &Generator{store: store, bucket: bucket}
}

// Generate makes the thumbnail for one image. Running it again overwrites the
// thumbnail with an identical one, so it is safe to retry.
func (g *Generator) Generate(ctx context.Context, request Request) (Result, error) {
	original, err := g.store.Get(ctx, request.Source)
	if err != nil {
		return Result{}, fmt.Errorf("read original: %w", err)
	}

	source, format, err := image.Decode(bytes.NewReader(original))
	if err != nil {
		return Result{}, fmt.Errorf("%w: %w", drive.ErrUnsupportedImage, err)
	}

	data, contentType, err := encode(Shrink(source, MaxSide), format)
	if err != nil {
		return Result{}, err
	}

	target := drive.Object{Bucket: g.bucket, Key: request.Source.Key}
	if err := g.store.Put(ctx, target, data, contentType); err != nil {
		return Result{}, fmt.Errorf("store thumbnail: %w", err)
	}

	return Result{Thumbnail: target}, nil
}
