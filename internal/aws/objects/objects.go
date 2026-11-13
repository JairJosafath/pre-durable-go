// Package objects keeps the uploads and their thumbnails in S3.
//
//	objects.go  read, write and delete objects
//	urls.go     presigned upload and download URLs
//
// The adapter only needs an endpoint that speaks the S3 API, so the local
// stack uses it against SeaweedFS.
package objects

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gdf/internal/drive"
)

// Objects stores and removes objects in S3.
type Objects struct {
	client  *s3.Client
	presign *s3.PresignClient
}

// New returns an object store that uses the given S3 client.
func New(client *s3.Client) *Objects {
	return &Objects{client: client, presign: s3.NewPresignClient(client)}
}

// Get downloads an object.
func (o *Objects) Get(ctx context.Context, object drive.Object) ([]byte, error) {
	output, err := o.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(object.Bucket),
		Key:    aws.String(object.Key),
	})
	if err != nil {
		return nil, fmt.Errorf("get s3://%s/%s: %w", object.Bucket, object.Key, err)
	}

	defer output.Body.Close() //nolint:errcheck // nothing to do about a failed close after a read

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("read s3://%s/%s: %w", object.Bucket, object.Key, err)
	}

	return data, nil
}

// Put uploads an object, replacing any existing one.
func (o *Objects) Put(ctx context.Context, object drive.Object, data []byte, contentType string) error {
	_, err := o.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(object.Bucket),
		Key:         aws.String(object.Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", object.Bucket, object.Key, err)
	}

	return nil
}

// Delete removes an object. Deleting an object that is already gone succeeds,
// which makes the call safe to retry.
func (o *Objects) Delete(ctx context.Context, object drive.Object) error {
	_, err := o.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(object.Bucket),
		Key:    aws.String(object.Key),
	})
	if err != nil {
		return fmt.Errorf("delete s3://%s/%s: %w", object.Bucket, object.Key, err)
	}

	return nil
}
