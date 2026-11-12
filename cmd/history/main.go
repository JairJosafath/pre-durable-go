// Command history is the Lambda function behind GET /executions. It shows
// what the durable execution of an upload has done so far.
package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	lambdaapi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"gdf/internal/api"
	"gdf/internal/awsdrive"
	"gdf/internal/env"
)

func main() {
	table, err := env.Require("TABLE_NAME")
	if err != nil {
		log.Fatalf("start history: %v", err)
	}

	aws, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("start history: %v", err)
	}

	history := api.NewHistory(
		awsdrive.NewCatalog(dynamodb.NewFromConfig(aws), table),
		awsdrive.NewExecutions(lambdaapi.NewFromConfig(aws)),
	)

	lambda.Start(history.Handle)
}
