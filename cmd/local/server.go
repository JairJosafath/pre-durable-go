package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/api"
)

const (
	gatewayTimeout    = 30 * time.Second
	readHeaderTimeout = 10 * time.Second
)

// serve runs the REST API of infra/lib/constructs/rest-api.ts on this
// machine, so the frontend works without AWS:
//
//	GET    /files, /query, /labels   DynamoDB Local, as the gateway queries DynamoDB
//	GET    /object/{key+}            thumbnail from SeaweedFS
//	DELETE /object/{key+}            delete an original from SeaweedFS
//	GET    /presigned                the Presigner handler that runs in Lambda
//	POST   /reviews                  the Reviewer handler that runs in Lambda
//	PUT    /uploads/{key+}           local only: the upload bucket and its S3 event
//
// There is no /auth route and no Cognito. A request is signed in as the user
// named in its bearer token; see signedIn.
func serve(ctx context.Context, opts options) error {
	services := connect(opts)
	if err := services.prepare(ctx); err != nil {
		return err
	}

	workflow, err := newWorkflow(services, opts)
	if err != nil {
		return err
	}

	running := newExecutions(func() *runner { return newRunner(workflow, services) })
	local := &gateway{
		services:   services,
		executions: running,
		dynamoURL:  opts.dynamoURL,
		http:       &http.Client{Timeout: gatewayTimeout},
	}

	// The two Lambda handlers are the ones cmd/presigner and cmd/reviewer
	// start, with local implementations behind their interfaces.
	presigner := api.NewPresigner(localSigner{Objects: services.objects, host: opts.serve}, uploadBucket)
	reviewer := api.NewReviewer(services.catalog, running)

	mux := http.NewServeMux()
	mux.Handle("GET /files", signedIn(local.query(allFiles)))
	mux.Handle("GET /query", signedIn(local.query(filesByLabel)))
	mux.Handle("GET /labels", signedIn(local.query(labelsByPrefix)))
	mux.Handle("GET /object/{key...}", signedIn(local.thumbnail))
	mux.Handle("DELETE /object/{key...}", signedIn(local.remove))
	mux.Handle("GET /presigned", signedIn(lambdaProxy(presigner.Handle)))
	mux.Handle("POST /reviews", signedIn(lambdaProxy(reviewer.Handle)))
	mux.HandleFunc("PUT /uploads/{key...}", local.upload)

	server := &http.Server{Addr: opts.serve, Handler: cors(mux), ReadHeaderTimeout: readHeaderTimeout}

	fmt.Printf("PrimeDrive API on http://%s (Ctrl+C to stop)\n", opts.serve)

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve API: %w", err)
	}

	return nil
}

// userHandler answers a request of a signed-in user.
type userHandler func(w http.ResponseWriter, r *http.Request, subject string)

// signedIn stands in for the Cognito authorizer. Locally nobody signs in:
// the bearer token is taken to be the user's name, and that name is used
// wherever AWS uses the token's "sub" claim. A request without a token is
// turned away, as the authorizer does.
func signedIn(next userHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !found || subject == "" || strings.Contains(subject, "/") {
			writeJSON(w, http.StatusUnauthorized, message("Unauthorized"))

			return
		}

		next(w, r, subject)
	})
}

// lambdaHandler is the signature of a Lambda function behind API Gateway.
type lambdaHandler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// lambdaProxy calls a Lambda handler the way the gateway's proxy integration
// does: the HTTP request becomes a proxy event that carries the caller's
// claims, and the handler's response becomes the HTTP response.
func lambdaProxy(handle lambdaHandler) userHandler {
	return func(w http.ResponseWriter, r *http.Request, subject string) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fail(w, fmt.Errorf("read request: %w", err))

			return
		}

		response, err := handle(r.Context(), events.APIGatewayProxyRequest{
			HTTPMethod:            r.Method,
			Path:                  r.URL.Path,
			QueryStringParameters: firstValues(r),
			Body:                  string(body),
			RequestContext: events.APIGatewayProxyRequestContext{
				Authorizer: map[string]any{"claims": map[string]any{"sub": subject}},
			},
		})
		if err != nil {
			fail(w, err)

			return
		}

		for name, value := range response.Headers {
			w.Header().Set(name, value)
		}

		w.WriteHeader(response.StatusCode)
		_, _ = io.WriteString(w, response.Body) // the browser went away; nobody is left to tell
	}
}

// firstValues are the query parameters as a proxy event carries them.
func firstValues(r *http.Request) map[string]string {
	parameters := map[string]string{}
	for name, values := range r.URL.Query() {
		parameters[name] = values[0]
	}

	return parameters
}

// cors lets the frontend, which runs on another port, call this API. In AWS
// the gateway answers the preflight requests.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// message is the body of an error response.
func message(text string) map[string]string {
	return map[string]string{"message": text}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write response", "error", err)
	}
}
