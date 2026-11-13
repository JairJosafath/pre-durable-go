// Command local runs the upload pipeline on your own machine, against
// open-source services instead of AWS:
//
//	Amazon S3            ->  SeaweedFS            (same adapter, other endpoint)
//	Amazon DynamoDB      ->  DynamoDB Local       (same adapter, other endpoint)
//	Amazon Rekognition   ->  ResNet-50 on TensorFlow Serving
//	Lambda durable execution service  ->  the SDK's local runner
//
// The workflow code is exactly what runs in Lambda. Only this package, the
// composition root, differs from cmd/pipeline.
//
// It runs in one of two ways. With -image it uploads one file, runs the
// workflow for it and asks for a review on the terminal:
//
//	make local-up
//	go run ./cmd/local -image ~/Pictures/cat.jpg
//
// With -serve it stays up as the REST API the frontend talks to, in place of
// API Gateway and Cognito (see server.go):
//
//	go run ./cmd/local -serve localhost:8080
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-durable-execution-sdk-go/durable"
	"github.com/aws/aws-durable-execution-sdk-go/durable/durabletest"
	"github.com/aws/aws-lambda-go/events"

	"gdf/internal/aws/catalog"
	"gdf/internal/drive"
	"gdf/internal/local/imagenet"
	"gdf/internal/pipeline"
	"gdf/internal/thumbnail"
)

const (
	uploadBucket    = "primedrive-uploads"
	thumbnailBucket = "primedrive-thumbnails"
	tableName       = "primedrive"
	thumbnailerName = "thumbnailer"
	visionModel     = "resnet"

	reviewTimeout = 24 * time.Hour
	visionTimeout = 30 * time.Second
	// A 1000-class classifier spreads its confidence more thinly than
	// Rekognition does, so the local review threshold is lower.
	defaultMinConfidence = 40
)

type options struct {
	image         string
	serve         string
	owner         string
	s3URL         string
	dynamoURL     string
	visionURL     string
	minConfidence float64
}

func main() {
	opts := options{}
	flag.StringVar(&opts.image, "image", "", "path of a JPEG or PNG to upload and process once")
	flag.StringVar(&opts.serve, "serve", "", "serve the REST API on this host:port instead, for example localhost:8080")
	flag.StringVar(&opts.owner, "owner", "alice", "user who uploads the image")
	flag.StringVar(&opts.s3URL, "s3", "http://localhost:8333", "SeaweedFS S3 endpoint")
	flag.StringVar(&opts.dynamoURL, "dynamodb", "http://localhost:8000", "DynamoDB Local endpoint")
	flag.StringVar(&opts.visionURL, "vision", "http://localhost:8501", "TensorFlow Serving endpoint")
	flag.Float64Var(&opts.minConfidence, "min-confidence", defaultMinConfidence, "review uploads without a label this confident")
	flag.Parse()

	mode := run
	if opts.serve != "" {
		mode = serve
	}

	if opts.image == "" && opts.serve == "" {
		flag.Usage()
		os.Exit(2) //nolint:mnd // conventional exit code for a usage error
	}

	if err := mode(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run uploads one image and runs the workflow for it.
func run(ctx context.Context, opts options) error {
	services := connect(opts)
	if err := services.prepare(ctx); err != nil {
		return err
	}

	upload, err := services.upload(ctx, opts.owner, opts.image)
	if err != nil {
		return err
	}

	workflow, err := newWorkflow(services, opts)
	if err != nil {
		return err
	}

	result, err := execute(ctx, newRunner(workflow, services), services.catalog, upload)
	if err != nil {
		return err
	}

	return report(result)
}

type runner = durabletest.LocalRunner[events.S3Event, drive.File]

// newWorkflow wires the workflow to the local services. It is the same
// constructor as in cmd/pipeline, with different implementations.
func newWorkflow(services *services, opts options) (*pipeline.Pipeline, error) {
	return pipeline.New(
		imagenet.NewVision(services.objects, &http.Client{Timeout: visionTimeout}, opts.visionURL, visionModel),
		services.catalog,
		services.objects,
		pipeline.Config{
			ThumbnailerFunction: thumbnailerName,
			ReviewTimeout:       reviewTimeout,
			ReviewPolicy:        drive.ReviewPolicy{MinConfidence: opts.minConfidence},
		},
	)
}

// newRunner returns a runner for one execution of the workflow.
func newRunner(workflow *pipeline.Pipeline, services *services) *runner {
	// The local runner plays the part of the Lambda durable execution
	// service: it keeps the checkpoints and re-invokes the handler.
	logs := slog.NewTextHandler(os.Stderr, nil)
	local := durabletest.NewLocalRunner(workflow.Handle, durable.WithLogHandler(logs))
	generator := thumbnail.NewGenerator(services.objects, thumbnailBucket)
	local.RegisterFunction(thumbnailerName, durabletest.PlainFunction(generator.Generate))

	return local
}

// execute runs the workflow to its end, asking for a review decision on the
// terminal when the execution suspends to wait for one.
func execute(
	ctx context.Context, local *runner, catalog *catalog.Catalog, upload drive.Upload,
) (*durabletest.TestResult, error) {
	event := uploadedEvent(upload)

	result, err := local.RunUntilComplete(event)
	if err != nil {
		return nil, fmt.Errorf("run workflow: %w", err)
	}

	if result.Status != durabletest.Pending {
		return result, nil
	}

	fmt.Printf("\nThe execution is suspended. Nothing is running while it waits:\n\n%s\n", result.FormatTree())

	file, err := catalog.File(ctx, upload.Owner, upload.Key)
	if err != nil {
		return nil, err
	}

	if file.Review == nil {
		return result, nil
	}

	decision := askReviewer(*file.Review)
	if err := local.SendCallbackSuccess(file.Review.CallbackID, decision); err != nil {
		return nil, fmt.Errorf("send decision: %w", err)
	}

	result, err = local.RunUntilComplete(event)
	if err != nil {
		return nil, fmt.Errorf("resume workflow: %w", err)
	}

	return result, nil
}

// askReviewer shows why the upload needs review and reads the decision.
func askReviewer(review drive.Review) drive.Decision {
	fmt.Println("This upload needs a reviewer:")

	for _, reason := range review.Reasons {
		fmt.Println("  -", reason)
	}

	fmt.Print("Approve it? [y/N] ")

	var answer string

	_, _ = fmt.Scanln(&answer) // no answer means no

	approved := answer == "y" || answer == "Y"

	return drive.Decision{Approved: approved, Reviewer: "you", Note: "reviewed on the command line"}
}

// report prints how the execution went and what it produced.
func report(result *durabletest.TestResult) error {
	fmt.Printf("\nExecution %s after %d invocation(s):\n\n%s\n", result.Status, len(result.Invocations), result.FormatTree())

	file, err := durabletest.ResultAs[drive.File](result)
	if err != nil {
		return fmt.Errorf("workflow did not succeed: %w", err)
	}

	pretty, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode result: %w", err)
	}

	fmt.Printf("Result:\n%s\n", pretty)

	return nil
}

// uploadedEvent is the event S3 would send for the upload.
func uploadedEvent(upload drive.Upload) events.S3Event {
	return events.S3Event{Records: []events.S3EventRecord{{
		S3: events.S3Entity{
			Bucket: events.S3Bucket{Name: upload.Bucket},
			Object: events.S3Object{Key: upload.Key},
		},
	}}}
}

// objectName is the file name an image gets in the drive.
func objectName(path string) string {
	return filepath.Base(path)
}
