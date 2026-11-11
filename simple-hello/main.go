package main

import (
	"github.com/aws/aws-durable-execution-sdk-go/durable"
)

func main() {
	durable.Start(handler)
}

func handler(ctx durable.Context, event any) (any, error) {
	return 0, nil
}
