# PrimeDrive frontend

A Next.js page for the drive: sign in, upload, search by label, review what
the pipeline could not decide on its own, and, in dev mode, watch the
durable function work.

Run it from the repository root:

```bash
make local
```

```bash
make web
```

`make local` runs it against the local stack, signed in as `alice`.
`make web` runs it against the deployed stack. Both read two settings,
described in [.env.example](.env.example).

## Where things are

| Path | What it is |
| --- | --- |
| `src/app/page.jsx` | The one page: what is shown when |
| `src/hooks/useDrive.jsx` | The drive, kept live by polling; upload and review |
| `src/hooks/useSearch.jsx` | Search by label, as you type |
| `src/hooks/useThumbnails.jsx` | Thumbnails, fetched with the user's token |
| `src/hooks/useHistories.jsx` | Dev mode: execution histories, kept live |
| `src/lib/timeline.js` | Dev mode: the operation tree and the squeezed time axis |
| `src/components/` | Top bar, photo grid, review queue, lightbox, dev panel |
| `src/local/` | Local flow only: start signed in, without the sign-in form |
| `tailwind.config.ts` | The colours, fonts and animations of the one dark theme |

The page has no state library and no component library: React, Tailwind and
the API.

## The page, and dev mode

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
upload (`ctx.ExecutionArn()`, in [register.go](../internal/pipeline/register.go)).
The history function ([internal/api/history.go](../internal/api/history.go))
takes that ARN from the caller's own record, never from the request, reads
the events and folds them into a list of operations
([internal/execution](../internal/execution/doc.go)).

Two limits. Lambda keeps a history for the execution's retention period, 14
days here; after that the panel says the history is gone. And an upload from
before this route existed has no execution in its record: upload it again to
watch it.
