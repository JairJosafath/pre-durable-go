// Package drive holds the PrimeDrive domain: what a user uploaded, what was
// found in it, and when a person has to look at it. It knows nothing about
// AWS or about durable execution.
package drive

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
)

// ErrNotAnUpload reports an object key outside the users/<owner>/<name> layout.
var ErrNotAnUpload = errors.New("key is not of the form users/<owner>/<name>")

// ErrUnsupportedImage reports an image that can never be analysed, however
// often the analysis is retried.
var ErrUnsupportedImage = errors.New("unsupported image")

const (
	userPrefix   = "users"
	keySegments  = 3
	ownerSegment = 1
)

// Object addresses a blob in an object store.
type Object struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// Upload is an object a user put into their drive.
type Upload struct {
	Object

	Owner string `json:"owner"`
}

// ParseUpload derives the owner of an object from its key, which has the
// form users/<owner>/<name>.
func ParseUpload(bucket, key string) (Upload, error) {
	segments := strings.SplitN(key, "/", keySegments)
	if len(segments) != keySegments || segments[0] != userPrefix || slices.Contains(segments, "") {
		return Upload{}, fmt.Errorf("%w: %q", ErrNotAnUpload, key)
	}

	return Upload{
		Object: Object{Bucket: bucket, Key: key},
		Owner:  segments[ownerSegment],
	}, nil
}

// IsImage reports whether the upload has a file type the pipeline can analyse.
func (u Upload) IsImage() bool {
	switch strings.ToLower(path.Ext(u.Key)) {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
}
