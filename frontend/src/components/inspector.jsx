import { useEffect, useMemo, useState } from "react";
import { axis, elapsed, tree } from "@/lib/timeline";
import { clock, duration, plural, tail } from "@/lib/format";
import { AlertIcon, CheckIcon, ClockIcon, CloseIcon, PinIcon, RetryIcon, Spinner } from "@/components/icons";

// Dev mode: what the durable function did for an upload, read from the
// history Lambda keeps of every durable execution.

const PHASES = {
  UPLOADING: { text: "uploading", tone: "text-accent bg-accent/10" },
  STARTING: { text: "starting", tone: "text-accent bg-accent/10" },
  PROCESSING: { text: "processing", tone: "text-accent bg-accent/10" },
  PUBLISHING: { text: "publishing", tone: "text-accent bg-accent/10" },
  REMOVING: { text: "removing", tone: "text-accent bg-accent/10" },
  PENDING_REVIEW: { text: "awaiting review", tone: "text-warn bg-warn/10" },
  READY: { text: "ready", tone: "text-ok bg-ok/10" },
  REJECTED: { text: "rejected", tone: "text-muted bg-white/5" },
  FAILED: { text: "failed", tone: "text-bad bg-bad/10" },
};

// The order uploads are listed in: what is moving first.
const ORDER = ["UPLOADING", "STARTING", "PROCESSING", "PUBLISHING", "REMOVING", "PENDING_REVIEW", "FAILED", "READY", "REJECTED"];

const EXECUTIONS = {
  RUNNING: "text-accent",
  SUCCEEDED: "text-ok",
  FAILED: "text-bad",
  TIMED_OUT: "text-bad",
  STOPPED: "text-muted",
};

// How an operation is drawn, by status: its sign, and the colour of its bar.
const OPERATIONS = {
  RUNNING: { sign: <Spinner className="h-3.5 w-3.5 text-accent" />, bar: "bg-accent" },
  RETRYING: { sign: <RetryIcon className="h-3.5 w-3.5 animate-breathe text-warn" />, bar: "bg-warn" },
  SUCCEEDED: { sign: <CheckIcon className="h-3.5 w-3.5 text-ok" />, bar: "bg-ok/80" },
  FAILED: { sign: <CloseIcon className="h-3.5 w-3.5 text-bad" />, bar: "bg-bad" },
  TIMED_OUT: { sign: <ClockIcon className="h-3.5 w-3.5 text-bad" />, bar: "bg-bad" },
  STOPPED: { sign: <CloseIcon className="h-3.5 w-3.5 text-muted" />, bar: "bg-faint" },
  CANCELLED: { sign: <CloseIcon className="h-3.5 w-3.5 text-muted" />, bar: "bg-faint" },
};
// An open callback is not work: nothing runs while it waits for a person.
const WAITING = { sign: <ClockIcon className="h-3.5 w-3.5 animate-breathe text-warn" />, bar: "bg-warn/70" };

// What kind of operation a row is, in a word. Steps are the default and get
// no tag; a context says which SDK call opened it.
const KINDS = { INVOKE: "invoke", CALLBACK: "callback", WAIT: "wait" };
const CONTEXTS = {
  RunInChildContext: "context",
  Map: "map",
  MapIteration: "item",
  Parallel: "parallel",
  ParallelBranch: "branch",
  WaitForCallback: "wait for callback",
  WaitForCondition: "wait for condition",
};

function kindOf(operation) {
  if (operation.type !== "CONTEXT") return KINDS[operation.type] || "";

  return CONTEXTS[operation.subType] || (operation.subType || "context").toLowerCase();
}

// The SDK starts a few operations itself, without a name: the callback inside
// WaitForCallback, for one. Those go by what they are.
function nameOf(operation) {
  return operation.name || (operation.subType || operation.type).toLowerCase();
}

function Pill({ phase }) {
  const { text, tone } = PHASES[phase] || { text: phase.toLowerCase(), tone: "text-muted bg-white/5" };

  return <span className={`shrink-0 rounded-md px-1.5 py-0.5 font-mono text-[11px] ${tone}`}>{text}</span>;
}

/** Ticks while something is running, so open bars and durations keep growing. */
function useNow(running) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    setNow(Date.now());
    if (!running) return undefined;

    const timer = setInterval(() => setNow(Date.now()), 200);

    return () => clearInterval(timer);
  }, [running]);

  return now;
}

/** Where time was squeezed out of the axis, drawn through a lane. */
function Breaks({ breaks }) {
  return breaks.map((gap) => (
    <span
      key={gap.at}
      title={`${duration(gap.gap)} of waiting, not drawn to scale`}
      className="absolute inset-y-0 border-l border-dashed border-faint/70"
      style={{ left: `${gap.at}%` }}
    />
  ));
}

function Lane({ bar, tone, open, breaks }) {
  return (
    <div className="relative h-full">
      <Breaks breaks={breaks} />
      <span
        className={`absolute top-1/2 h-1.5 min-w-[3px] -translate-y-1/2 overflow-hidden rounded-full ${tone}`}
        style={{ left: `${Math.min(bar.left, 99)}%`, width: `${bar.width}%` }}
      >
        {open && <span className="sheen absolute inset-0" />}
      </span>
    </div>
  );
}

function Operation({ operation, scale, now }) {
  const waiting = operation.type === "CALLBACK" && operation.status === "RUNNING";
  const look = waiting ? WAITING : OPERATIONS[operation.status] || OPERATIONS.RUNNING;
  const open = !operation.endedAt;
  const kind = kindOf(operation);
  // A step that succeeded on a later attempt has nothing left to complain about.
  const error = operation.status === "SUCCEEDED" ? "" : operation.error;

  return (
    <li className="group">
      <div className="grid h-7 grid-cols-[minmax(0,1fr)_112px_56px] items-center gap-2 rounded-md px-2 hover:bg-white/[.04]">
        <div className="flex min-w-0 items-center gap-1.5" style={{ paddingLeft: operation.depth * 14 }}>
          <span className="flex h-4 w-4 shrink-0 items-center justify-center">{look.sign}</span>
          <span className={`truncate text-[12.5px] ${operation.depth === 0 ? "text-fg" : "text-fg/80"}`} title={nameOf(operation)}>
            {nameOf(operation)}
          </span>
          {kind && <span className="shrink-0 whitespace-nowrap rounded bg-white/5 px-1 text-[10px] text-faint">{kind}</span>}
          {operation.attempts > 1 && (
            <span className="shrink-0 rounded bg-warn/10 px-1 text-[10px] text-warn" title={`${operation.attempts} attempts`}>
              ×{operation.attempts}
            </span>
          )}
        </div>
        <Lane bar={scale.bar(operation)} tone={look.bar} open={open} breaks={scale.breaks} />
        <span className={`text-right text-[11px] tabular-nums ${open ? "text-accent" : "text-muted"}`}>
          {waiting ? "waiting" : duration(elapsed(operation, now))}
        </span>
      </div>
      {error && (
        <p className="mb-1 mr-2 break-words rounded-md bg-bad/10 px-2 py-1 text-[11px] leading-snug text-bad" style={{ marginLeft: 30 + operation.depth * 14 }}>
          {error}
        </p>
      )}
    </li>
  );
}

function Fact({ name, children, title }) {
  return (
    <div className="min-w-0">
      <dt className="text-[10px] uppercase tracking-wider text-faint">{name}</dt>
      <dd className="truncate text-[12.5px] text-fg" title={title}>
        {children}
      </dd>
    </div>
  );
}

function History({ history }) {
  const { execution, operations = [], invocations = [] } = history;
  const running = execution.status === "RUNNING";
  const now = useNow(running);

  const rows = useMemo(() => tree(operations), [operations]);
  const scale = axis(history, now);
  const replays = Math.max(0, invocations.length - 1);

  return (
    <>
      <dl className="grid grid-cols-3 gap-x-3 gap-y-2.5 border-y border-line py-3">
        <Fact name="Execution">
          <span className={EXECUTIONS[execution.status] || "text-fg"}>{execution.status.toLowerCase().replace("_", " ")}</span>
        </Fact>
        <Fact name="Started">{clock(execution.startedAt)}</Fact>
        <Fact name={running ? "Running for" : "Took"}>{duration(elapsed(execution, now))}</Fact>
        <Fact name="Invocations">{invocations.length}</Fact>
        <Fact name="Replays" title="How often the handler ran again from the top, skipping what was already checkpointed">
          {replays}
        </Fact>
        <Fact name="Id" title={execution.arn}>
          {tail(execution.arn)}
        </Fact>
      </dl>

      {execution.error && <p className="mt-3 break-words rounded-lg border border-bad/30 bg-bad/10 px-2.5 py-2 text-[12px] leading-snug text-bad">{execution.error}</p>}

      <div className="mt-3 grid h-6 grid-cols-[minmax(0,1fr)_112px_56px] items-center gap-2 px-2 text-[10px] uppercase tracking-wider text-faint">
        <span title="Every time Lambda ran the handler. Between invocations nothing runs.">Handler invocations</span>
        <div className="relative h-full">
          <Breaks breaks={scale.breaks} />
          {invocations.map((invocation, index) => {
            const bar = scale.bar(invocation);

            return (
              <span
                key={index}
                title={`Invocation ${index + 1}: ${duration(elapsed(invocation, now))}${invocation.error ? `, ${invocation.error}` : ""}`}
                className={`absolute top-1/2 h-2.5 min-w-[3px] -translate-y-1/2 rounded-sm ${invocation.error ? "bg-bad/70" : "bg-fg/45"}`}
                style={{ left: `${Math.min(bar.left, 99)}%`, width: `${bar.width}%` }}
              />
            );
          })}
        </div>
        <span />
      </div>

      <ol className="mt-1">
        {rows.map((operation) => (
          <Operation key={operation.id} operation={operation} scale={scale} now={now} />
        ))}
      </ol>

      {operations.length === 0 && <p className="px-2 py-3 text-[12px] text-muted">The execution has started. No operation is checkpointed yet.</p>}
      {scale.breaks.length > 0 && <p className="mt-3 px-2 text-[11px] text-faint">A dashed line is waiting time, squeezed so the work stays visible.</p>}
    </>
  );
}

function Detail({ file, entry }) {
  return (
    <section className="border-b border-line p-4">
      <div className="mb-3 flex items-center gap-2">
        <h3 className="min-w-0 flex-1 truncate font-sans text-[15px] font-semibold" title={file.key}>
          {file.title}
        </h3>
        <Pill phase={file.phase} />
      </div>

      {file.run && (
        <p className="mb-3 truncate text-[11px] text-faint" title="The name this run gave itself in the name-run step. The catalog record belongs to it.">
          run <span className="text-muted">{file.run}</span>
        </p>
      )}

      {!entry && (
        <div className="flex items-center gap-2 py-6 text-[12px] text-muted">
          <Spinner className="h-3.5 w-3.5" /> Reading the execution history
        </div>
      )}
      {entry?.history && <History history={entry.history} />}
      {entry && !entry.history && (
        <p className={`flex items-start gap-2 rounded-lg border px-2.5 py-2 text-[12px] leading-snug ${entry.state === "error" ? "border-bad/30 bg-bad/10 text-bad" : "border-line bg-raised text-muted"}`}>
          <AlertIcon className="mt-px h-3.5 w-3.5 shrink-0" />
          <span>
            {entry.message.replace(/\.?$/, ".")}
            {entry.state === "missing" && file.status && " Upload the file again to watch its execution."}
          </span>
        </p>
      )}
    </section>
  );
}

/**
 * The dev panel: the execution of one upload in detail, and every upload the
 * catalog knows below it. It follows whatever is being processed, until the
 * user picks an upload themselves.
 */
export default function Inspector({ files, histories, selected, pinned, select, follow, onClose }) {
  const sorted = useMemo(
    () => [...files].sort((a, b) => ORDER.indexOf(a.phase) - ORDER.indexOf(b.phase) || a.title.localeCompare(b.title)),
    [files],
  );
  const file = files.find((other) => other.title === selected);

  return (
    <aside className="fixed inset-x-0 bottom-0 z-20 flex h-[58vh] animate-slide flex-col border-t border-line bg-surface/95 font-mono backdrop-blur-xl lg:inset-x-auto lg:right-0 lg:top-16 lg:h-auto lg:w-[460px] lg:border-l lg:border-t-0">
      <header className="flex h-11 shrink-0 items-center gap-2 border-b border-line px-4 text-[12px]">
        <span className="h-1.5 w-1.5 animate-breathe rounded-full bg-accent" />
        <span className="flex-1 uppercase tracking-wider text-muted">Durable executions</span>
        <button
          type="button"
          title={pinned ? "Follow whatever is being processed" : "Following whatever is being processed"}
          className={`flex items-center gap-1 rounded-md px-2 py-1 transition ${pinned ? "text-muted hover:bg-white/5 hover:text-fg" : "bg-accent/10 text-accent"}`}
          onClick={follow}
        >
          <PinIcon className="h-3.5 w-3.5" />
          {pinned ? "pinned" : "following"}
        </button>
        <button type="button" aria-label="Close dev mode" className="rounded-md p-1 text-muted transition hover:bg-white/5 hover:text-fg" onClick={onClose}>
          <CloseIcon className="h-4 w-4" />
        </button>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto">
        {file ? (
          <Detail file={file} entry={histories[file.title]} />
        ) : (
          <p className="border-b border-line p-4 text-[12px] leading-relaxed text-muted">Upload a photo and its execution shows up here, one operation at a time.</p>
        )}

        <div className="p-2">
          <p className="px-2 py-2 text-[10px] uppercase tracking-wider text-faint">{plural(sorted.length, "upload")} in the catalog</p>
          <ul>
            {sorted.map((other) => (
              <li key={other.key}>
                <button
                  type="button"
                  className={`flex h-8 w-full items-center gap-2 rounded-md px-2 text-left text-[12.5px] transition ${
                    other.title === selected ? "bg-accent/10 text-fg" : "text-muted hover:bg-white/[.04] hover:text-fg"
                  }`}
                  onClick={() => select(other.title)}
                >
                  <span className="min-w-0 flex-1 truncate">{other.title}</span>
                  <Pill phase={other.phase} />
                </button>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </aside>
  );
}
