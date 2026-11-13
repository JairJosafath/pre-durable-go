// Package vision recognises what is in an image with Amazon Rekognition: the
// labels, and the moderation concerns.
package vision

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"

	"gdf/internal/drive"
)

const (
	maxLabels               = 10
	minLabelConfidence      = 55
	minModerationConfidence = 60
)

// Vision recognises image content with Amazon Rekognition. Rekognition reads
// the image straight from S3, so no image bytes pass through the function.
type Vision struct {
	client *rekognition.Client
}

// New returns a vision service that uses the given Rekognition client.
func New(client *rekognition.Client) *Vision {
	return &Vision{client: client}
}

// DetectLabels returns what Rekognition recognises in the image.
func (v *Vision) DetectLabels(ctx context.Context, image drive.Object) ([]drive.Label, error) {
	output, err := v.client.DetectLabels(ctx, &rekognition.DetectLabelsInput{
		Image:         s3Image(image),
		MaxLabels:     aws.Int32(maxLabels),
		MinConfidence: aws.Float32(minLabelConfidence),
	})
	if err != nil {
		return nil, fmt.Errorf("detect labels: %w", permanent(err))
	}

	labels := make([]drive.Label, 0, len(output.Labels))
	for _, label := range output.Labels {
		labels = append(labels, describe(label))
	}

	return labels, nil
}

// DetectModerationLabels returns the content concerns Rekognition raises
// about the image, such as alcohol or weapons.
func (v *Vision) DetectModerationLabels(ctx context.Context, image drive.Object) ([]drive.Label, error) {
	output, err := v.client.DetectModerationLabels(ctx, &rekognition.DetectModerationLabelsInput{
		Image:         s3Image(image),
		MinConfidence: aws.Float32(minModerationConfidence),
	})
	if err != nil {
		return nil, fmt.Errorf("detect moderation labels: %w", permanent(err))
	}

	flags := make([]drive.Label, 0, len(output.ModerationLabels))
	for _, label := range output.ModerationLabels {
		flags = append(flags, toLabel(label.Name, label.Confidence))
	}

	return flags, nil
}

func s3Image(image drive.Object) *types.Image {
	return &types.Image{S3Object: &types.S3Object{
		Bucket: aws.String(image.Bucket),
		Name:   aws.String(image.Key),
	}}
}

// describe keeps the categories and aliases Rekognition knows for a label;
// the catalog stores them once per label.
func describe(label types.Label) drive.Label {
	described := toLabel(label.Name, label.Confidence)

	for _, category := range label.Categories {
		described.Categories = append(described.Categories, aws.ToString(category.Name))
	}

	for _, alias := range label.Aliases {
		described.Aliases = append(described.Aliases, aws.ToString(alias.Name))
	}

	return described
}

func toLabel(name *string, confidence *float32) drive.Label {
	return drive.Label{Name: aws.ToString(name), Confidence: float64(aws.ToFloat32(confidence))}
}

// permanent marks the failures that retrying cannot fix, in terms the
// workflow understands, so it does not need to know Rekognition's error types.
func permanent(err error) error {
	var (
		badFormat *types.InvalidImageFormatException
		tooLarge  *types.ImageTooLargeException
	)

	if errors.As(err, &badFormat) || errors.As(err, &tooLarge) {
		return fmt.Errorf("%w: %w", drive.ErrUnsupportedImage, err)
	}

	return err
}
