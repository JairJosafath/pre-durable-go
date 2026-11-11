package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"gdf/internal/awsdrive"
	"gdf/internal/drive"
)

// pageSize is PAGE_SIZE in infra/lib/constructs/rest-api.ts.
const pageSize = 24

// gateway does locally what API Gateway does without any Lambda function in
// AWS: it calls DynamoDB and S3 directly, on behalf of the signed-in user.
// Each method mirrors one integration in infra/lib/constructs/rest-api.ts.
type gateway struct {
	services   *services
	executions *executions
	dynamoURL  string
	http       *http.Client
}

// query answers a catalog route with one page of the user's partition. The
// request is the one the gateway's mapping template builds, and DynamoDB's
// answer goes back to the browser untouched, so the frontend sees the same
// JSON as in AWS.
func (g *gateway) query(sortKeyPrefix func(*http.Request) string) userHandler {
	return func(w http.ResponseWriter, r *http.Request, subject string) {
		partition := attribute("user#" + subject)

		input := map[string]any{
			"TableName":                tableName,
			"Limit":                    pageSize,
			"KeyConditionExpression":   "#pk = :pk AND begins_with(#sk, :sk)",
			"ExpressionAttributeNames": map[string]string{"#pk": "pk", "#sk": "sk"},
			"ExpressionAttributeValues": map[string]any{
				":pk": partition,
				":sk": attribute(sortKeyPrefix(r)),
			},
		}

		if next := r.URL.Query().Get("next"); next != "" {
			input["ExclusiveStartKey"] = map[string]any{"pk": partition, "sk": attribute(next)}
		}

		if err := g.forward(w, r, "Query", input); err != nil {
			fail(w, err)
		}
	}
}

// attribute is a string in DynamoDB's JSON format.
func attribute(value string) map[string]string {
	return map[string]string{"S": value}
}

func allFiles(*http.Request) string { return "#file#" }

func filesByLabel(r *http.Request) string {
	return "#index#" + strings.ToLower(r.URL.Query().Get("index"))
}

func labelsByPrefix(r *http.Request) string {
	return "#label#" + strings.ToLower(r.URL.Query().Get("prefix"))
}

// forward sends one action to DynamoDB Local over its HTTP API and copies
// the response to the browser.
func (g *gateway) forward(w http.ResponseWriter, r *http.Request, action string, input any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode %s: %w", action, err)
	}

	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, g.dynamoURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build %s: %w", action, err)
	}

	request.Header.Set("Content-Type", "application/x-amz-json-1.0")
	request.Header.Set("X-Amz-Target", "DynamoDB_20120810."+action)

	digest := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(
		r.Context(), localCredentials(), request, hex.EncodeToString(digest[:]), "dynamodb", localRegion, time.Now(),
	); err != nil {
		return fmt.Errorf("sign %s: %w", action, err)
	}

	response, err := g.http.Do(request)
	if err != nil {
		return fmt.Errorf("call DynamoDB Local (is the local stack up? run `make local-up`): %w", err)
	}

	defer response.Body.Close() //nolint:errcheck // nothing to do about a failed close after a read

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body) // the browser went away; nobody is left to tell

	return nil
}

// thumbnail answers GET /object/{key+} from the thumbnail bucket. The key is
// always under the caller's own users/<sub>/ prefix.
func (g *gateway) thumbnail(w http.ResponseWriter, r *http.Request, subject string) {
	output, err := g.services.s3.GetObject(r.Context(), &s3.GetObjectInput{
		Bucket: aws.String(thumbnailBucket),
		Key:    aws.String(drive.UserKey(subject, r.PathValue("key"))),
	})

	var missing *s3types.NoSuchKey
	if errors.As(err, &missing) {
		writeJSON(w, http.StatusNotFound, message("no such object"))

		return
	}

	if err != nil {
		fail(w, err)

		return
	}

	defer output.Body.Close() //nolint:errcheck // nothing to do about a failed close after a read

	w.Header().Set("Content-Type", aws.ToString(output.ContentType))
	_, _ = io.Copy(w, output.Body) // the browser went away; nobody is left to tell
}

// remove answers DELETE /object/{key+} by deleting the original.
func (g *gateway) remove(w http.ResponseWriter, r *http.Request, subject string) {
	original := drive.Object{Bucket: uploadBucket, Key: drive.UserKey(subject, r.PathValue("key"))}
	if err := g.services.objects.Delete(r.Context(), original); err != nil {
		fail(w, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// upload receives a file the browser PUTs to the URL from GET /presigned,
// stores it, and starts the workflow.
//
// In AWS the browser uploads straight to S3, and S3 invokes the durable
// function. SeaweedFS does not tell us when an object lands, so this route
// stands in for both: the upload bucket and its ObjectCreated event.
func (g *gateway) upload(w http.ResponseWriter, r *http.Request) {
	upload, err := drive.ParseUpload(uploadBucket, r.PathValue("key"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, message(err.Error()))

		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, fmt.Errorf("read upload: %w", err))

		return
	}

	if err := g.services.objects.Put(r.Context(), upload.Object, data, r.Header.Get("Content-Type")); err != nil {
		fail(w, err)

		return
	}

	g.executions.Start(upload)
	w.WriteHeader(http.StatusOK)
}

// localSigner is the api.Signer the Presigner handler gets on this machine.
// The handler asks it for two kinds of URL:
//
//   - DownloadURL is not written here. Embedding *awsdrive.Objects gives
//     localSigner every method of the real object store, so a download is a
//     presigned URL of the bucket, exactly as in AWS.
//   - UploadURL is replaced below. It returns the address of this server's
//     PUT /uploads route instead of a presigned URL of the bucket.
//
// The reason is the trigger. In AWS the browser PUTs to S3 and S3 then
// invokes the durable function. SeaweedFS stores the file but invokes
// nothing, so an upload sent straight to it would never be processed.
// Sending the upload through this server lets it store the file and start
// the workflow itself (see upload).
type localSigner struct {
	*awsdrive.Objects

	// host is where this server listens, such as localhost:8080.
	host string
}

// UploadURL returns the address of the local upload route for the object.
// Unlike a presigned URL it does not expire, so the lifetime is ignored.
func (s localSigner) UploadURL(_ context.Context, object drive.Object, _ time.Duration) (string, error) {
	address := url.URL{Scheme: "http", Host: s.host, Path: "/uploads/" + object.Key}

	return address.String(), nil
}

// fail reports an error of the local stack, the way the gateway reports a
// failed integration.
func fail(w http.ResponseWriter, err error) {
	slog.Error("request failed", "error", err)
	writeJSON(w, http.StatusInternalServerError, message("Internal server error"))
}
