// Command pipeline is the durable Lambda function that processes uploads.
//
// This file is the composition root: it decides which implementations the
// workflow runs with. Here those are AWS services. cmd/local makes a
// different choice for the same workflow.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gdf/internal/awsdrive"
	"gdf/internal/drive"
	"gdf/internal/env"
	"gdf/internal/pipeline"
)

const (
	defaultReviewTimeout = 24 * time.Hour
	minLabelConfidence   = 80
)

func main() {
	workflow, err := newWorkflow(context.Background())
	if err != nil {
		log.Fatalf("start pipeline: %v", err)
	}

	// durable.Start is lambda.Start for durable functions: it registers the
	// handler and takes care of checkpointing and replay.
	durable.Start(workflow.Handle)
}

func newWorkflow(ctx context.Context) (*pipeline.Pipeline, error) {
	table, err := env.Require("TABLE_NAME")
	if err != nil {
		return nil, err
	}

	thumbnailer, err := env.Require("THUMBNAILER_FUNCTION")
	if err != nil {
		return nil, err
	}

	reviewTimeout, err := env.Duration("REVIEW_TIMEOUT", defaultReviewTimeout)
	if err != nil {
		return nil, err
	}

	aws, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}

	return pipeline.New(
		awsdrive.NewVision(rekognition.NewFromConfig(aws)),
		awsdrive.NewCatalog(dynamodb.NewFromConfig(aws), table),
		awsdrive.NewObjects(s3.NewFromConfig(aws)),
		pipeline.Config{
			ThumbnailerFunction: thumbnailer,
			ReviewTimeout:       reviewTimeout,
			ReviewPolicy:        drive.ReviewPolicy{MinConfidence: minLabelConfidence},
		},
	), nil
}
