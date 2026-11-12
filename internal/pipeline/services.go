package pipeline

import (
	"context"

	"gdf/internal/drive"
)

// Vision recognises what is in an image.
type Vision interface {
	DetectLabels(ctx context.Context, image drive.Object) ([]drive.Label, error)
	DetectModerationLabels(ctx context.Context, image drive.Object) ([]drive.Label, error)
}

// Catalog records uploads and makes them searchable by label. Every method
// overwrites what is there, so calling one twice has the same effect as
// calling it once.
type Catalog interface {
	// RegisterFile writes the record of a new upload. The record then belongs
	// to file.Run, whichever run owned it before.
	RegisterFile(ctx context.Context, file drive.File) error
	// SaveFile overwrites the record while file.Run still owns it. Once a
	// newer upload has taken the record over, it writes nothing and returns
	// drive.ErrSuperseded.
	SaveFile(ctx context.Context, file drive.File) error
	IndexLabel(ctx context.Context, upload drive.Upload, label drive.Label) error
}

// ObjectStore removes objects that must not be kept.
type ObjectStore interface {
	Delete(ctx context.Context, object drive.Object) error
}
