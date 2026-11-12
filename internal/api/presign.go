package api

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
)

const urlLifetime = time.Hour

// Signer creates short-lived URLs for direct access to one object.
type Signer interface {
	UploadURL(ctx context.Context, object drive.Object, ttl time.Duration) (string, error)
	DownloadURL(ctx context.Context, object drive.Object, ttl time.Duration) (string, error)
}

// Presigner answers GET /presigned?action=put|get&key=<file name>.
type Presigner struct {
	signer Signer
	bucket string
}

// NewPresigner returns a handler that signs URLs for objects in bucket.
func NewPresigner(signer Signer, bucket string) *Presigner {
	return &Presigner{signer: signer, bucket: bucket}
}

// Handle returns a URL for the caller's own file. The object key is always
// built from the caller's identity, so nobody can reach another user's files.
func (p *Presigner) Handle(
	ctx context.Context, request events.APIGatewayProxyRequest,
) (events.APIGatewayProxyResponse, error) {
	owner, ok := caller(request)
	if !ok {
		return unauthorized()
	}

	name := request.QueryStringParameters["key"]
	if name == "" {
		return refuse(http.StatusBadRequest, "missing query parameter: key")
	}

	object := drive.Object{Bucket: p.bucket, Key: drive.UserKey(owner, name)}

	sign, ok := p.action(request.QueryStringParameters["action"])
	if !ok {
		return refuse(http.StatusBadRequest, "action must be put or get")
	}

	url, err := sign(ctx, object, urlLifetime)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return respond(http.StatusOK, map[string]string{"url": url})
}

type signFunc func(ctx context.Context, object drive.Object, ttl time.Duration) (string, error)

func (p *Presigner) action(name string) (signFunc, bool) {
	switch name {
	case "put":
		return p.signer.UploadURL, true
	case "get":
		return p.signer.DownloadURL, true
	default:
		return nil, false
	}
}
