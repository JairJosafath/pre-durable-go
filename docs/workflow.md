# The workflow

The upload pipeline is one durable function, [internal/pipeline](../internal/pipeline/doc.go).
Start reading in [pipeline.go](../internal/pipeline/pipeline.go): `Handle` and
`process` are the whole story, and every stage they call is in the file named
after it.

```
browser ──presigned PUT──▶ S3 uploads ──ObjectCreated──▶ durable function (alias "live")
                                                           │
   name-run                   step                         │
   register-upload            step                         │
   analyze-image              durable.Go ─┐                │
     detect-labels            step        ├─ durable.Join  │
     detect-moderation        step        │                │
   generate-thumbnail         InvokeAsync ┘  ──▶ thumbnailer function
   human-review               WaitForCallback  ◀── POST /reviews (reviewer function)
   index-labels               Map              ──▶ DynamoDB
   mark-ready / mark-rejected step
   discard-upload             Parallel         ──▶ S3 deletes (rejected uploads)
   mark-failed                step             (when any stage above gives up)
```

An upload ends in one of three statuses. `READY` is published and indexed.
`REJECTED` is gone: a reviewer turned it down, nobody reviewed it in time, or
it is not an image, and the original and its thumbnail are deleted. `FAILED`
could not be processed, and the original is kept.

S3 delivers events at least once, and a user can upload the same key again
while the first execution still waits for a reviewer. So every execution names
its run (`name-run`) and writes the name into the file record. The record
belongs to the run that registered last, and every later write is conditional
on still owning it. An execution that was overtaken changes and deletes
nothing; it ends with the result `SUPERSEDED`.

## Where each SDK idea lives

From the [Go SDK guide](https://docs.aws.amazon.com/durable-execution/sdk-reference/languages/go/):

| Idea | Code |
| --- | --- |
| The handler and `durable.Start` | [cmd/pipeline/main.go](../cmd/pipeline/main.go), [pipeline.go `Handle`](../internal/pipeline/pipeline.go) |
| Steps and checkpointing | [record.go `save`](../internal/pipeline/record.go) |
| Retry strategies | [retry.go `retryTransient`](../internal/pipeline/retry.go) |
| Goroutines and `durable.Go` | [analysis.go `analyzeAndThumbnail`](../internal/pipeline/analysis.go) |
| Futures and combinators (`InvokeAsync`, `Join`) | [analysis.go `analyzeAndThumbnail`](../internal/pipeline/analysis.go) |
| Invoking another function | [analysis.go](../internal/pipeline/analysis.go), [cmd/thumbnailer](../cmd/thumbnailer/main.go) |
| Callbacks (`WaitForCallback`) | [review.go](../internal/pipeline/review.go), [aws/callbacks](../internal/aws/callbacks/callbacks.go) |
| Typed errors | [review.go](../internal/pipeline/review.go) (`CallbackTimeoutError`), [pipeline.go `Handle`](../internal/pipeline/pipeline.go) (`OperationError`) |
| A business outcome as a step result, not an error | [record.go `save`](../internal/pipeline/record.go) |
| Nondeterminism inside a step | [register.go `register`](../internal/pipeline/register.go) (the run's random name) |
| `Map` and `Parallel` | [publish.go](../internal/pipeline/publish.go) (`Map`), [discard.go](../internal/pipeline/discard.go) (`Parallel`) |
| Replay-aware logging | `ctx.Logger()` in [record.go `settle`](../internal/pipeline/record.go), `sc.Logger()` in [analysis.go](../internal/pipeline/analysis.go) |
| Static analysis with durablelint | `make lint`, pinned in [tools/go.mod](../tools/go.mod) |
| Testing with durabletest | [harness_test.go](../internal/pipeline/harness_test.go), [pipeline_test.go](../internal/pipeline/pipeline_test.go) |
| Cloud runner | [cloud_test.go](../internal/pipeline/cloud_test.go) |

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
non-retryable failure, an upload that ends as `FAILED`, a review that is
approved, rejected, or never answered, and an upload that is replaced while
its first execution is still running.

```bash
make lint
```

`lint` runs golangci-lint (cognitive and cyclomatic complexity, nesting,
function length and more, see [.golangci.yml](../.golangci.yml)) and durablelint,
which finds code that would break on replay.

```bash
make check
```

`check` runs all of it in one go, with the frontend's lint, the tests of the
CDK app and a `cdk synth` that compiles the five Go functions. It needs no AWS
credentials.
