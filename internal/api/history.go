package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/drive"
	"gdf/internal/execution"
)

// Executions knows what the durable execution that owns a file has done. It
// returns execution.ErrNotFound when the history is not kept, or no longer.
type Executions interface {
	History(ctx context.Context, file drive.File) (execution.History, error)
}

// History answers GET /executions?key=<file name> with what the pipeline has
// done for one of the caller's uploads: every operation, attempt and replay.
type History struct {
	files      Files
	executions Executions
}

// NewHistory returns a handler that finds the execution of an upload in
// files and reads its history from executions.
func NewHistory(files Files, executions Executions) *History {
	return &History{files: files, executions: executions}
}

// Handle returns the history of the execution that owns the caller's file.
// The request names a file, never an execution: which execution to read is
// taken from the caller's own catalog record, so nobody can look into the
// uploads of another user.
func (h *History) Handle(
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

	file, err := h.files.File(ctx, owner, ownKey(owner, name))
	if errors.Is(err, drive.ErrFileNotFound) {
		return refuse(http.StatusNotFound, "no such file")
	}

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if file.Execution == "" {
		return refuse(http.StatusNotFound, "this file was uploaded before executions were recorded")
	}

	history, err := h.executions.History(ctx, file)
	if errors.Is(err, execution.ErrNotFound) {
		return refuse(http.StatusNotFound, "the history of this execution is no longer kept")
	}

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return respond(http.StatusOK, history)
}

// ownKey is the object key of one of the owner's files. The file may be
// named the way GET /presigned names it, "cat.jpg", or by its whole key, as
// GET /files lists it. Either way the key is in the owner's own prefix.
func ownKey(owner, name string) string {
	if strings.HasPrefix(name, drive.UserKey(owner, "")) {
		return name
	}

	return drive.UserKey(owner, name)
}
