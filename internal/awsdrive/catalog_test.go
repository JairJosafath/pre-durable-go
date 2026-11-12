package awsdrive_test

// These tests point the real DynamoDB client at a local HTTP server. They
// check what the adapter asks DynamoDB for and what it makes of the answer,
// not DynamoDB itself.

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"gdf/internal/awsdrive"
	"gdf/internal/drive"
)

const conditionFailed = `{
	"__type": "com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException",
	"message": "The conditional request failed"
}`

// putItem is the part of a PutItem request these tests look at.
type putItem struct {
	ConditionExpression       string
	ExpressionAttributeNames  map[string]string
	ExpressionAttributeValues map[string]map[string]string
	Item                      map[string]json.RawMessage
}

// catalogAnswering returns a catalog whose DynamoDB answers every request
// with the given status and body, and the requests it received.
func catalogAnswering(t *testing.T, status int, body string) (*awsdrive.Catalog, *[]putItem) {
	t.Helper()

	var received []putItem

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request putItem

		payload, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(payload, &request); err != nil {
			t.Errorf("request is not JSON: %v", err)
		}

		received = append(received, request)

		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	client := dynamodb.New(dynamodb.Options{
		Region:           "local",
		BaseEndpoint:     aws.String(server.URL),
		Credentials:      credentials.NewStaticCredentialsProvider("local", "local", ""),
		RetryMaxAttempts: 1,
	})

	return awsdrive.NewCatalog(client, "primedrive"), &received
}

func beerPhoto(run string) drive.File {
	return drive.File{
		Upload: drive.Upload{Object: drive.Object{Bucket: "uploads", Key: "users/alice/beer.jpg"}, Owner: "alice"},
		Run:    run,
		Status: drive.StatusReady,
	}
}

func TestRegisterFileTakesTheRecordOverUnconditionally(t *testing.T) {
	t.Parallel()

	catalog, requests := catalogAnswering(t, http.StatusOK, `{}`)

	if err := catalog.RegisterFile(t.Context(), beerPhoto("run-2")); err != nil {
		t.Fatalf("register: %v", err)
	}

	if len(*requests) != 1 || (*requests)[0].ConditionExpression != "" {
		t.Errorf("requests = %+v, want one put without a condition", *requests)
	}
}

func TestSaveFileWritesOnlyWhileTheRunOwnsTheRecord(t *testing.T) {
	t.Parallel()

	catalog, requests := catalogAnswering(t, http.StatusOK, `{}`)

	if err := catalog.SaveFile(t.Context(), beerPhoto("run-1")); err != nil {
		t.Fatalf("save: %v", err)
	}

	if len(*requests) != 1 {
		t.Fatalf("got %d requests, want 1", len(*requests))
	}

	request := (*requests)[0]

	// The condition compares the stored run with the run that is writing.
	if request.ConditionExpression != "#run = :run" ||
		request.ExpressionAttributeNames["#run"] != "run" ||
		request.ExpressionAttributeValues[":run"]["S"] != "run-1" {
		t.Errorf("condition = %+v, want the record's run to equal run-1", request)
	}

	// And the item carries that run, under the name the condition reads.
	if string(request.Item["run"]) != `{"S":"run-1"}` {
		t.Errorf("item.run = %s, want run-1", request.Item["run"])
	}
}

func TestSaveFileReportsARecordThatWasTakenOver(t *testing.T) {
	t.Parallel()

	catalog, _ := catalogAnswering(t, http.StatusBadRequest, conditionFailed)

	err := catalog.SaveFile(t.Context(), beerPhoto("run-1"))
	if !errors.Is(err, drive.ErrSuperseded) {
		t.Errorf("err = %v, want ErrSuperseded", err)
	}
}

func TestSaveFilePassesOtherFailuresOn(t *testing.T) {
	t.Parallel()

	catalog, _ := catalogAnswering(t, http.StatusBadRequest, `{
		"__type": "com.amazonaws.dynamodb.v20120810#ResourceNotFoundException",
		"message": "Requested resource not found"
	}`)

	err := catalog.SaveFile(t.Context(), beerPhoto("run-1"))
	if err == nil || errors.Is(err, drive.ErrSuperseded) {
		t.Errorf("err = %v, want a failure that is not ErrSuperseded", err)
	}
}
