// Command reviewer is the Lambda function behind POST /reviews. It resumes
// the durable execution that is waiting for a review decision.
package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	lambdaapi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"gdf/internal/api"
	"gdf/internal/aws/callbacks"
	"gdf/internal/aws/catalog"
	"gdf/internal/env"
)

func main() {
	table, err := env.Require("TABLE_NAME")
	if err != nil {
		log.Fatalf("start reviewer: %v", err)
	}

	aws, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("start reviewer: %v", err)
	}

	reviewer := api.NewReviewer(
		catalog.New(dynamodb.NewFromConfig(aws), table),
		callbacks.New(lambdaapi.NewFromConfig(aws)),
	)

	lambda.Start(reviewer.Handle)
}
