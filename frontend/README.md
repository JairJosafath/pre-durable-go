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
| `tailwind.config.ts` | The colours, fonts and animations of the one dark theme |

The page has no state library and no component library: React, Tailwind and
the API.
