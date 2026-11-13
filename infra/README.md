# PrimeDrive on AWS

The CDK app that deploys PrimeDrive: the buckets, the catalog table, Cognito,
the REST API and the durable function. One construct per part, in
[lib/constructs](lib/constructs); [lib/prime-drive-stack.ts](lib/prime-drive-stack.ts)
puts them together.

Run everything from the repository root, with `make`.

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
version of the durable function and deletes the old one. The new code is
live as soon as the stack says `UPDATE_COMPLETE_CLEANUP_IN_PROGRESS`; what
follows is only that delete. Once it failed twice with "Exceeded attempts to
wait" and went through on the third try, 27 minutes later. Two executions on
the old version were still waiting for a review at the time, which may be
why. `make empty-all` stops open executions, should you want none around
before a deploy.

## A user, some photos, and the web page

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
make empty-all
```

`empty-all` starts from scratch: it deletes every user's photos, thumbnails
and catalog records, and stops the executions that are still open. The users
stay. `make empty NAME=alice` empties one drive. Both show what they are
about to delete and ask first, because nothing can be brought back; `DRY=1`
only shows.

```bash
make web
```

`web` runs the frontend on http://localhost:3000 against the deployed API,
which it reads from the stack's `ApiUrl` output. Only the web page is local.

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
