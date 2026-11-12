package drive

import "errors"

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
