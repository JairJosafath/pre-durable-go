# PrimeDrive on Lambda durable functions

A port of [PrimeDrive](https://github.com/JairJosafath/PrimeDrive-CDK-Project) from
Step Functions to a single **durable Lambda function written in Go**.

A user uploads a photo. The pipeline labels it, checks it for moderation
concerns, makes a thumbnail, asks a person to review it when something looks
off, and indexes it so it can be found by what is in it.

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

## Quick start

`make` lists every target. These are the ones you need most:

| You want to | Run |
| --- | --- |
| Check everything before a push: lint, tests, synth | `make check` |
| Try the whole app on your machine, without AWS | `make local`, then `make local-seed` in a second terminal |
| Deploy to AWS | `make deploy` |
| Get a user to sign in with, and its password | `make user NAME=alice` |
| Put sample photos in that user's drive | `make seed NAME=alice` |
| Open the web page against the deployment | `make web` |

## Layout

| Path | What it is |
| --- | --- |
| `infra/` | CDK app: buckets, table, Cognito, REST API, the durable function |
| `internal/pipeline/` | The durable workflow and its tests; start at [pipeline.go](internal/pipeline/pipeline.go), one file per stage ([doc.go](internal/pipeline/doc.go) lists them) |
| `internal/drive/` | Domain types and the review policy; no AWS, no durable SDK |
| `internal/awsdrive/` | S3, DynamoDB, Rekognition and callback adapters |
| `internal/imagenet/` | Open-source vision adapter (ResNet-50 on TensorFlow Serving) |
| `internal/drivetest/` | In-memory fakes for tests |
| `internal/execution/` | What a durable execution did, folded from Lambda's history events |
| `internal/thumbnail/`, `internal/api/` | The thumbnailer and the three API handlers |
| `cmd/pipeline/` | Composition root for Lambda: AWS implementations |
| `cmd/local/` | Composition root for your laptop: open-source implementations |
| `frontend/` | The Next.js frontend, with a review queue and a dev mode |
| `local/` | Docker Compose file for the offline stack |
| `tools/` | The scripts behind the make targets |

## Where each SDK idea lives

From the [Go SDK guide](https://docs.aws.amazon.com/durable-execution/sdk-reference/languages/go/):

| Idea | Code |
| --- | --- |
| The handler and `durable.Start` | [cmd/pipeline/main.go](cmd/pipeline/main.go), [pipeline.go `Handle`](internal/pipeline/pipeline.go) |
| Steps and checkpointing | [record.go `save`](internal/pipeline/record.go) |
| Retry strategies | [retry.go `retryTransient`](internal/pipeline/retry.go) |
| Goroutines and `durable.Go` | [analysis.go `analyzeAndThumbnail`](internal/pipeline/analysis.go) |
| Futures and combinators (`InvokeAsync`, `Join`) | [analysis.go `analyzeAndThumbnail`](internal/pipeline/analysis.go) |
| Invoking another function | [analysis.go](internal/pipeline/analysis.go), [cmd/thumbnailer](cmd/thumbnailer/main.go) |
| Callbacks (`WaitForCallback`) | [review.go](internal/pipeline/review.go), [awsdrive/callbacks.go](internal/awsdrive/callbacks.go) |
| Typed errors | [review.go](internal/pipeline/review.go) (`CallbackTimeoutError`), [pipeline.go `Handle`](internal/pipeline/pipeline.go) (`OperationError`) |
| A business outcome as a step result, not an error | [record.go `save`](internal/pipeline/record.go) |
| Nondeterminism inside a step | [register.go `register`](internal/pipeline/register.go) (the run's random name) |
| `Map` and `Parallel` | [publish.go](internal/pipeline/publish.go) (`Map`), [discard.go](internal/pipeline/discard.go) (`Parallel`) |
| Replay-aware logging | `ctx.Logger()` in [record.go `settle`](internal/pipeline/record.go), `sc.Logger()` in [analysis.go](internal/pipeline/analysis.go) |
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

## The web page, and dev mode

The frontend shows the drive as it is: the photos, the uploads the pipeline
is still working on, and the ones that wait for a review. It keeps itself up
to date by asking `GET /files` again, every second and a half while
something is being processed and every eight seconds otherwise, and not at
all while the tab is in the background. There is nothing to refresh.

The **dev** switch in the top bar opens a panel that shows what the durable
function did for an upload: every operation by name, with its status,
duration, attempts and errors, and every invocation of the handler, so the
replays are visible. The panel follows whatever is being processed until you
pick an upload yourself.

```
browser ──GET /executions?key=cat.jpg──▶ history function ──▶ catalog: which execution owns cat.jpg?
                                                          └─▶ Lambda: GetDurableExecutionHistory
```

Lambda records an event for everything a durable execution does. The
workflow stores its execution ARN in the file record when it registers an
upload (`ctx.ExecutionArn()`, in [register.go](internal/pipeline/register.go)).
The history function ([internal/api/history.go](internal/api/history.go))
takes that ARN from the caller's own record, never from the request, reads
the events and folds them into a list of operations
([internal/execution](internal/execution/doc.go)).

Two limits. Lambda keeps a history for the execution's retention period, 14
days here; after that the panel says the history is gone. And an upload from
before this route existed has no execution in its record: upload it again to
watch it.

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
function length and more, see [.golangci.yml](.golangci.yml)) and durablelint,
which finds code that would break on replay.

```bash
make check
```

`check` runs all of it in one go, with the frontend's lint, the tests of the
CDK app and a `cdk synth` that compiles the four Go functions. It needs no AWS
credentials.

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

One command starts the containers, the API and the frontend. Open
http://localhost:3000 when it says so; Ctrl+C stops it.

```bash
make local
```

In a second terminal, fill the drive with sample photos:

```bash
make local-seed
```

`local-seed` downloads about twenty random photos from Wikimedia Commons, one
per topic (a cat, a bridge, a pizza), and four that show beer or wine, to
give you something to review. They are kept in `local/samples/`; `FRESH=1`
downloads a new random set, and `SAMPLES=<folder>` uploads your own photos
instead.

To watch the API and the frontend in separate terminals, run `make local-up`
and then `make local-api` and `make local-web`.

| AWS | Local stand-in |
| --- | --- |
| API Gateway | `cmd/local -serve`, the same routes on `localhost:8080` |
| Presigner, reviewer and history functions | The same handlers from `internal/api`, called in-process |
| Cognito | Nothing: you are signed in as `alice` |
| S3 event that starts the workflow | The upload goes through the local API, which starts it |

Nothing in the code that runs in AWS knows about local mode. The frontend
reads two settings ([frontend/.env.example](frontend/.env.example)), which
`make local` and `make local-web` set:

| Setting | AWS | Local |
| --- | --- | --- |
| `NEXT_PUBLIC_ENDPOINT` | the `ApiUrl` stack output | `http://localhost:8080` |
| `NEXT_PUBLIC_LOCAL_USER` | unset: users sign in | `alice`: no sign-in |

With `NEXT_PUBLIC_LOCAL_USER` set, the frontend skips the sign-in form and
sends the user's name as its token. The local API takes that name where AWS
takes the `sub` claim of a verified Cognito token. A deployed API rejects
such a token, so the setting cannot open a real drive.

### What the local stack is not

It is a stand-in to try the workflow with, not a copy of the cloud.
`make local` prints this list every time it starts:

- **Labels are not Rekognition's.** A small open classifier (ResNet-50,
  trained on ImageNet) knows 1,000 object classes and guesses which of them
  the whole photo is. Labels are often odd or plain wrong, and it cannot pick
  out several things in one picture.
- **There is no moderation model.** A photo is flagged only when the
  classifier sees beer, wine, a cocktail shaker or a gun. A photo it is less
  than 40% sure about goes to review too, so more photos wait for a reviewer
  than would in AWS.
- **Nobody signs in.** You are `alice`. There is no Cognito, and the API
  believes whatever name the browser sends.
- **Executions are not durable across restarts.** They live in the memory of
  the local API. Stop it and an upload that was waiting for a review can no
  longer be answered; upload it again.
- **Dev mode is coarser.** The local runner reports on an execution when it
  suspends or ends, so its operations appear all at once, not one by one as
  they do in AWS.
- **The catalog is kept in memory** by DynamoDB Local. `make local-down`
  deletes it, with every uploaded photo.
- **Missing altogether:** S3 events and their at-least-once delivery, the
  dead-letter queue, IAM permissions, API Gateway, and Lambda's limits and
  timeouts. Code that works here can still fail on those in AWS.

Stop the containers and delete their data with:

```bash
make local-down
```

## Deploy

Requires Node.js, Go, the AWS CLI, and AWS credentials for a region that
offers both Lambda durable functions and Amazon Rekognition. The targets
below use `eu-west-1`; add `AWS_REGION=<region>` to any of them for another.

```bash
make bootstrap
```

```bash
make deploy
```

`bootstrap` is needed once per account and region. `make outputs` shows the
stack's outputs again later, and `make destroy` deletes the stack with its
users, uploads and catalog.

Lambda does not retry a durable execution that fails. The S3 event that
started it is kept for 14 days in the queue named by the `FailedUploadsQueue`
stack output.

A later deploy can seem to hang at the end. Every deploy publishes a new
version of the durable function and deletes the old one, and Lambda does not
delete a version while an execution on it still waits, for a review for
instance. The new code is live as soon as the stack says
`UPDATE_COMPLETE_CLEANUP_IN_PROGRESS`; CloudFormation then keeps retrying
the delete, about ten minutes per attempt when this was seen. Answering the
open reviews first avoids the wait.

### A user, some photos, and the web page

```bash
make user NAME=alice
```

`user` creates the user in the stack's Cognito user pool and prints a
generated password to copy into the sign-in form. No email is sent and there
is no code to confirm. Run it again for the same name to get a new password.
It needs credentials that may administer the user pool, which the ones that
deployed the stack can. The API also has `POST /auth/signup` and
`POST /auth/confirm`, as in the original project, for signing up with a code
sent by email.

```bash
make seed NAME=alice
```

`seed` uploads the same sample photos as `make local-seed` into the user's
drive: about twenty random ones and four with beer or wine. Each upload
starts an execution of the durable function. Rekognition flagged three of the
four as Alcohol when this was tried (two empty beer bottles passed), and
those wait for a review. Rekognition is billed per image: the set costs about
5 US cents outside the free tier.

The photos go up one every two seconds. A new AWS account allows only 10
Lambda executions at once, and every upload runs the durable function and the
thumbnailer side by side. Uploaded in one burst, 14 of the 24 photos ended as
`FAILED`: the thumbnailer was refused with `Rate Exceeded`. `PACE=0` uploads
without the pause, for an account with the usual limit of 1,000.

```bash
make web
```

`web` runs the frontend on http://localhost:3000 against the deployed API,
which it reads from the stack's `ApiUrl` output. Only the web page is local.
