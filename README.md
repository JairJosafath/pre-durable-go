# PrimeDrive on Lambda durable functions

A port of [PrimeDrive](https://github.com/JairJosafath/PrimeDrive-CDK-Project) from
Step Functions to a single **durable Lambda function written in Go**.

A user uploads a photo. The pipeline labels it, checks it for moderation
concerns, makes a thumbnail, asks a person to review it when something looks
off, and indexes it so it can be found by what is in it.

```
browser ──presigned PUT──▶ S3 uploads ──ObjectCreated──▶ durable function (alias "live")
                                                           │
   register-upload            step                         │
   analyze-image              durable.Go ─┐                │
     detect-labels            step        ├─ durable.Join  │
     detect-moderation        step        │                │
   generate-thumbnail         InvokeAsync ┘  ──▶ thumbnailer function
   human-review               WaitForCallback  ◀── POST /reviews (reviewer function)
   index-labels               Map              ──▶ DynamoDB
   discard-upload             Parallel         ──▶ S3 deletes (rejected uploads)
   mark-ready / mark-rejected step
   mark-failed                step             (when any stage above gives up)
```

## Layout

| Path | What it is |
| --- | --- |
| `infra/` | CDK app: buckets, table, Cognito, REST API, the durable function |
| `internal/pipeline/` | The durable workflow and its tests |
| `internal/drive/` | Domain types and the review policy; no AWS, no durable SDK |
| `internal/awsdrive/` | S3, DynamoDB, Rekognition and callback adapters |
| `internal/imagenet/` | Open-source vision adapter (ResNet-50 on TensorFlow Serving) |
| `internal/drivetest/` | In-memory fakes for tests |
| `internal/thumbnail/`, `internal/api/` | The thumbnailer and the two API handlers |
| `cmd/pipeline/` | Composition root for Lambda: AWS implementations |
| `cmd/local/` | Composition root for your laptop: open-source implementations |
| `frontend/` | The Next.js frontend, with a review panel |
| `local/` | Docker Compose file for the offline stack |

## Where each SDK idea lives

From the [Go SDK guide](https://docs.aws.amazon.com/durable-execution/sdk-reference/languages/go/):

| Idea | Code |
| --- | --- |
| The handler and `durable.Start` | [cmd/pipeline/main.go](cmd/pipeline/main.go), [pipeline.go `Handle`](internal/pipeline/pipeline.go) |
| Steps and checkpointing | [pipeline.go `save`](internal/pipeline/pipeline.go) |
| Retry strategies | [analysis.go `retryTransient`](internal/pipeline/analysis.go) |
| Goroutines and `durable.Go` | [analysis.go `analyzeAndThumbnail`](internal/pipeline/analysis.go) |
| Futures and combinators (`InvokeAsync`, `Join`) | [analysis.go `analyzeAndThumbnail`](internal/pipeline/analysis.go) |
| Invoking another function | [analysis.go](internal/pipeline/analysis.go), [cmd/thumbnailer](cmd/thumbnailer/main.go) |
| Callbacks (`WaitForCallback`) | [review.go](internal/pipeline/review.go), [awsdrive/callbacks.go](internal/awsdrive/callbacks.go) |
| Typed errors | [review.go](internal/pipeline/review.go) (`CallbackTimeoutError`) |
| `Map` and `Parallel` | [outcome.go](internal/pipeline/outcome.go) |
| Replay-aware logging | `ctx.Logger()` in [pipeline.go](internal/pipeline/pipeline.go), `sc.Logger()` in [analysis.go](internal/pipeline/analysis.go) |
| Static analysis with durablelint | `make lint`, pinned in [tools/go.mod](tools/go.mod) |
| Testing with durabletest | [harness_test.go](internal/pipeline/harness_test.go), [pipeline_test.go](internal/pipeline/pipeline_test.go) |
| Cloud runner | [cloud_test.go](internal/pipeline/cloud_test.go) |

## Data model

One DynamoDB table, single-table design. Everything a user owns is in their
partition (`pk = user#<sub>`), and each question the app asks is one query on
a sort-key prefix:

| Question | `sk` begins with | Written by |
| --- | --- | --- |
| My files | `#file#` | every status change of the pipeline |
| Files found by "cat" | `#index#cat#` | `index-labels`, one item per file for the label's name and for each alias |
| Which labels do I have | `#label#` | `index-labels`, one item per label, with categories and aliases |

The table is shaped for reads. An upload is indexed once, in the background;
searches happen all the time. Queries are paged (24 items), so the cost of a
read stays flat as a drive grows. Every item is keyed by what it describes,
so the pipeline's writes are plain puts that are safe to retry and replay.

## Test it offline

No AWS account, credentials, network or containers:

```bash
make test
```

```bash
make test-workflow
```

`test-workflow` prints the operation tree of every execution. The tests run
the real handler on the SDK's local runner, with in-memory fakes behind the
workflow's interfaces. They cover publishing, retries on a virtual clock, a
non-retryable failure, an upload that ends as `FAILED`, and a review that is
approved, rejected, or never answered.

```bash
make lint
```

`lint` runs golangci-lint (cognitive and cyclomatic complexity, nesting,
function length and more, see [.golangci.yml](.golangci.yml)) and durablelint,
which finds code that would break on replay.

## Run it locally on open-source services

| AWS | Local stand-in |
| --- | --- |
| S3 | SeaweedFS (same adapter, other endpoint) |
| DynamoDB | DynamoDB Local (same adapter, other endpoint) |
| Rekognition | ResNet-50 ImageNet classifier on TensorFlow Serving |
| Durable execution service | The SDK's local runner |

```bash
make local-up
```

```bash
make local-run IMAGE=~/Pictures/cat.jpg
```

A photo of a beer glass or wine bottle is flagged as Alcohol and the
execution suspends until you answer on the terminal.

### With the frontend

The same stack can serve the frontend. Start the API in one terminal and the
frontend in another, then open http://localhost:3000:

```bash
make local-api
```

```bash
make local-web
```

| AWS | Local stand-in |
| --- | --- |
| API Gateway | `cmd/local -serve`, the same routes on `localhost:8080` |
| Presigner and reviewer functions | The same handlers from `internal/api`, called in-process |
| Cognito | Nothing: you are signed in as `alice` |
| S3 event that starts the workflow | The upload goes through the local API, which starts it |

Nothing in the code that runs in AWS knows about local mode. The frontend
reads two settings ([frontend/.env.example](frontend/.env.example)), which
`make local-web` sets:

| Setting | AWS | Local |
| --- | --- | --- |
| `NEXT_PUBLIC_ENDPOINT` | the `ApiUrl` stack output | `http://localhost:8080` |
| `NEXT_PUBLIC_LOCAL_USER` | unset: users sign in | `alice`: no sign-in |

With `NEXT_PUBLIC_LOCAL_USER` set, the frontend skips the sign-in form and
sends the user's name as its token. The local API takes that name where AWS
takes the `sub` claim of a verified Cognito token. A deployed API rejects
such a token, so the setting cannot open a real drive.

Suspended executions live in the memory of `make local-api`. Restart it and
an upload that was waiting for a review can no longer be answered; upload it
again.

```bash
make local-down
```

## Deploy

Requires Node.js, Go, and AWS credentials for a region that offers both
Lambda durable functions and Amazon Rekognition.

```bash
cd infra && npm install && npx cdk deploy
```

Then start the frontend with the `ApiUrl` stack output:

```bash
cd frontend && npm install && NEXT_PUBLIC_ENDPOINT=<ApiUrl> npm run dev
```

Users sign up through `POST /auth/signup` and `POST /auth/confirm`, as in the
original project.
