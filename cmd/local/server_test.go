package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
)

// whoAmI answers with the name of the signed-in user.
func whoAmI(w http.ResponseWriter, _ *http.Request, subject string) {
	writeJSON(w, http.StatusOK, message(subject))
}

func request(t *testing.T, method, target, token string) *http.Request {
	t.Helper()

	r := httptest.NewRequestWithContext(t.Context(), method, target, http.NoBody)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}

	return r
}

func TestSignedInTakesTheBearerTokenAsTheUser(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	signedIn(whoAmI).ServeHTTP(response, request(t, http.MethodGet, "/files", "alice"))

	if response.Code != http.StatusOK || response.Body.String() != `{"message":"alice"}`+"\n" {
		t.Fatalf("got %d %s, want 200 with alice", response.Code, response.Body)
	}
}

func TestSignedInTurnsAwayRequestsWithoutAUser(t *testing.T) {
	t.Parallel()

	// A name with a slash could reach into another user's key prefix.
	for _, token := range []string{"", "alice/../bob"} {
		response := httptest.NewRecorder()
		signedIn(whoAmI).ServeHTTP(response, request(t, http.MethodGet, "/files", token))

		if response.Code != http.StatusUnauthorized {
			t.Errorf("token %q: got %d, want 401", token, response.Code)
		}
	}
}

func TestLambdaProxyPassesClaimsAndQueryToTheHandler(t *testing.T) {
	t.Parallel()

	var seen events.APIGatewayProxyRequest

	handler := func(_ context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		seen = event

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusAccepted,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"ok":true}`,
		}, nil
	}

	response := httptest.NewRecorder()
	signedIn(lambdaProxy(handler)).ServeHTTP(response, request(t, http.MethodGet, "/presigned?action=put&key=cat.jpg", "alice"))

	claims, _ := seen.RequestContext.Authorizer["claims"].(map[string]any)
	if claims["sub"] != "alice" {
		t.Errorf("sub claim = %v, want alice", claims["sub"])
	}

	if seen.QueryStringParameters["key"] != "cat.jpg" || seen.QueryStringParameters["action"] != "put" {
		t.Errorf("query = %v, want action=put and key=cat.jpg", seen.QueryStringParameters)
	}

	if response.Code != http.StatusAccepted || response.Body.String() != `{"ok":true}` {
		t.Errorf("got %d %s, want the handler's 202 response", response.Code, response.Body)
	}
}

func TestCORSAnswersThePreflight(t *testing.T) {
	t.Parallel()

	reached := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })

	response := httptest.NewRecorder()
	cors(next).ServeHTTP(response, request(t, http.MethodOptions, "/uploads/users/alice/cat.jpg", ""))

	if reached {
		t.Error("the preflight reached the route")
	}

	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("got %d with headers %v, want 204 that allows any origin", response.Code, response.Header())
	}
}

func TestLocalSignerSendsUploadsToThisServer(t *testing.T) {
	t.Parallel()

	signer := localSigner{host: "localhost:8080"}
	object := drive.Object{Bucket: uploadBucket, Key: drive.UserKey("alice", "my cat.jpg")}

	got, err := signer.UploadURL(t.Context(), object, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if want := "http://localhost:8080/uploads/users/alice/my%20cat.jpg"; got != want {
		t.Errorf("upload URL = %s, want %s", got, want)
	}
}
