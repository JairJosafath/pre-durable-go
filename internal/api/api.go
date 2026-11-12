// Package api holds the Lambda handlers behind the REST API: one hands out
// presigned URLs, one passes a reviewer's decision to the durable execution
// that is waiting for it, and one shows what such an execution has done.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// caller returns the Cognito subject of the signed-in user. API Gateway has
// already verified the token; the claims arrive in the request context.
func caller(request events.APIGatewayProxyRequest) (string, bool) {
	claims, _ := request.RequestContext.Authorizer["claims"].(map[string]any)
	subject, _ := claims["sub"].(string)

	return subject, subject != ""
}

// respond builds a JSON response the browser accepts from any origin.
func respond(status int, body any) (events.APIGatewayProxyResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("encode response: %w", err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(payload),
	}, nil
}

// refuse builds an error response with a message for the client.
func refuse(status int, message string) (events.APIGatewayProxyResponse, error) {
	return respond(status, map[string]string{"message": message})
}

func unauthorized() (events.APIGatewayProxyResponse, error) {
	return refuse(http.StatusUnauthorized, "not signed in")
}
