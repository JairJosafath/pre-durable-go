package pipeline

import (
	"errors"
	"fmt"

	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
)

var errUnexpectedEvent = errors.New("expected an S3 event with exactly one record")

// uploadFrom reads the uploaded object out of the S3 event. It is plain,
// deterministic code, so it needs no step: replaying it gives the same answer.
func uploadFrom(event events.S3Event) (drive.Upload, error) {
	if len(event.Records) != 1 {
		return drive.Upload{}, fmt.Errorf("%w: got %d", errUnexpectedEvent, len(event.Records))
	}

	entity := event.Records[0].S3

	return drive.ParseUpload(entity.Bucket.Name, entity.Object.URLDecodedKey)
}
