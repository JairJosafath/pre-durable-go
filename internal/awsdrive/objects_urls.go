package awsdrive

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gdf/internal/drive"
)

// UploadURL returns a URL a browser can PUT the object to until it expires.
func (o *Objects) UploadURL(ctx context.Context, object drive.Object, ttl time.Duration) (string, error) {
	request, err := o.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(object.Bucket),
		Key:    aws.String(object.Key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign upload: %w", err)
	}

	return request.URL, nil
}

// DownloadURL returns a URL a browser can GET the object from until it expires.
func (o *Objects) DownloadURL(ctx context.Context, object drive.Object, ttl time.Duration) (string, error) {
	request, err := o.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(object.Bucket),
		Key:    aws.String(object.Key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign download: %w", err)
	}

	return request.URL, nil
}
