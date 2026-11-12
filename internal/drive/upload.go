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

// ErrFileNotFound reports that a user has no upload with the given key.
var ErrFileNotFound = errors.New("file not found")

// ErrSuperseded reports that a file record now belongs to a newer upload of
// the same key, so an older pipeline run may no longer write to it.
var ErrSuperseded = errors.New("superseded by a newer upload")

// ErrReviewClosed reports a decision that came too late: the review was
// already answered, or it timed out.
var ErrReviewClosed = errors.New("review is closed")

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

// UserKey returns the object key of a file in a user's drive.
func UserKey(owner, name string) string {
	return userPrefix + "/" + owner + "/" + name
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
