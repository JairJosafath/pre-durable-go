// Command presigner is the Lambda function behind GET /presigned.
package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gdf/internal/api"
	"gdf/internal/awsdrive"
	"gdf/internal/env"
)

func main() {
	bucket, err := env.Require("UPLOAD_BUCKET")
	if err != nil {
		log.Fatalf("start presigner: %v", err)
	}

	aws, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("start presigner: %v", err)
	}

	presigner := api.NewPresigner(awsdrive.NewObjects(s3.NewFromConfig(aws)), bucket)

	lambda.Start(presigner.Handle)
}
