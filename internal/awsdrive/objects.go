// Package awsdrive implements PrimeDrive's services on AWS: S3 for objects,
// DynamoDB for the catalog, Rekognition for vision and the Lambda API for
// review callbacks.
//
// The S3 and DynamoDB adapters only need an endpoint that speaks the same
// API, so the local stack reuses them against SeaweedFS and DynamoDB Local.
package awsdrive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gdf/internal/drive"
)

// Objects stores and removes objects in S3.
type Objects struct {
	client  *s3.Client
	presign *s3.PresignClient
}

// NewObjects returns an object store that uses the given S3 client.
func NewObjects(client *s3.Client) *Objects {
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
