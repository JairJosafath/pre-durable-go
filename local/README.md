# PrimeDrive on your machine, without AWS

Everything that exists only for this local flow is in a directory named
`local`. Nothing that runs in AWS imports or reads any of it.

| Path | What it is |
| --- | --- |
| `local/` | This folder: the Docker Compose file of the stand-in services, and the notice `make local` prints |
| `cmd/local/` | The program that runs the workflow here: the composition root with open-source implementations, and the API that stands in for API Gateway and Cognito |
| `internal/local/` | Go packages only that program uses: the open classifier that stands in for Rekognition |
| `tools/local/` | The scripts behind `make local` and `make local-seed` |
| `frontend/src/local/` | The frontend's shortcut past the sign-in form |

The workflow, the domain, the S3 and DynamoDB adapters and the three API
handlers are the same code that runs in AWS.

## Run it

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

## With the frontend

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
give you something to review. They are kept in `tools/samples/`; `FRESH=1`
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
reads two settings ([frontend/.env.example](../frontend/.env.example)), which
`make local` and `make local-web` set:

| Setting | AWS | Local |
| --- | --- | --- |
| `NEXT_PUBLIC_ENDPOINT` | the `ApiUrl` stack output | `http://localhost:8080` |
| `NEXT_PUBLIC_LOCAL_USER` | unset: users sign in | `alice`: no sign-in |

With `NEXT_PUBLIC_LOCAL_USER` set, the frontend skips the sign-in form and
sends the user's name as its token. The local API takes that name where AWS
takes the `sub` claim of a verified Cognito token. A deployed API rejects
such a token, so the setting cannot open a real drive.

## What the local stack is not

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
