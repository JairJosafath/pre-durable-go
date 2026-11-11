// Command thumbnailer is the Lambda function that makes thumbnails. It is an
// ordinary function; the pipeline calls it with durable.Invoke.
package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gdf/internal/awsdrive"
	"gdf/internal/env"
	"gdf/internal/thumbnail"
)

func main() {
	bucket, err := env.Require("THUMBNAIL_BUCKET")
	if err != nil {
		log.Fatalf("start thumbnailer: %v", err)
	}

	aws, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("start thumbnailer: %v", err)
	}

	generator := thumbnail.NewGenerator(awsdrive.NewObjects(s3.NewFromConfig(aws)), bucket)

	lambda.Start(generator.Generate)
}
