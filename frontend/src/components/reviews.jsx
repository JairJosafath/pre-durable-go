import { useState } from "react";
import { CheckIcon, CloseIcon, CodeIcon, Spinner } from "@/components/icons";

/**
 * Uploads the pipeline could not publish on its own. Each one is a durable
 * execution that is suspended until one of these buttons is pressed.
 */
export default function Reviews({ pending, thumbnail, decide, inspect, narrow }) {
  // key -> the answer that is on its way.
  const [answering, setAnswering] = useState({});

  if (pending.length === 0) return null;

  async function answer(file, approved) {
    setAnswering((current) => ({ ...current, [file.key]: approved }));
    await decide(file, approved);
    setAnswering(({ [file.key]: _, ...rest }) => rest);
  }

  return (
    <section className="mb-10 animate-rise">
      <div className="mb-3 flex items-baseline gap-3">
        <h2 className="text-lg font-semibold tracking-tight">Waiting for your review</h2>
        <span className="rounded-full bg-warn/15 px-2 py-0.5 text-xs font-medium text-warn">{pending.length}</span>
      </div>

      <ul className={`grid gap-3 ${narrow ? "2xl:grid-cols-2" : "xl:grid-cols-2"}`}>
        {pending.map((file) => {
          const busy = file.key in answering;
          const src = thumbnail(file) || file.preview;

          return (
            <li
              key={file.key}
              className={`flex animate-pop flex-col gap-3 rounded-2xl border bg-surface p-3 sm:flex-row sm:items-center sm:gap-4 sm:pr-4 ${
                inspect?.selected === file.title ? "border-accent/70" : "border-line"
              }`}
            >
              <div className="flex min-w-0 flex-1 items-center gap-4">
                <div className="h-20 w-20 shrink-0 overflow-hidden rounded-xl bg-raised">
                  {src ? <img src={src} alt={file.title} className="h-full w-full object-cover" /> : <div className="sheen h-full w-full" />}
                </div>

                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">{file.title}</p>
                  <p className="mt-0.5 truncate text-sm text-warn" title={file.reasons.join(" · ")}>
                    {file.reasons.join(" · ")}
                  </p>
                  <p className="mt-1 truncate text-xs text-muted">{file.labels.slice(0, 5).join(" · ")}</p>
                </div>
              </div>

              <div className="flex shrink-0 items-center justify-end gap-2">
                {inspect && (
                  <button
                    type="button"
                    title="Show the execution that is waiting"
                    className="rounded-xl border border-line p-2 text-accent transition hover:border-accent/50 hover:bg-accent/10"
                    onClick={() => inspect.select(file.title)}
                  >
                    <CodeIcon className="h-[18px] w-[18px]" />
                  </button>
                )}
                <button
                  type="button"
                  disabled={busy}
                  className="flex items-center gap-1.5 rounded-xl border border-line px-3 py-2 text-sm text-muted transition hover:border-bad/50 hover:bg-bad/10 hover:text-bad active:scale-[.97] disabled:opacity-40"
                  onClick={() => answer(file, false)}
                >
                  {answering[file.key] === false ? <Spinner /> : <CloseIcon className="h-4 w-4" />}
                  Reject
                </button>
                <button
                  type="button"
                  disabled={busy}
                  className="flex items-center gap-1.5 rounded-xl bg-ok/15 px-3 py-2 text-sm font-medium text-ok transition hover:bg-ok/25 active:scale-[.97] disabled:opacity-40"
                  onClick={() => answer(file, true)}
                >
                  {answering[file.key] === true ? <Spinner /> : <CheckIcon className="h-4 w-4" />}
                  Approve
                </button>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
