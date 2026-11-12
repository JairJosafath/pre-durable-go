package drive

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

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
