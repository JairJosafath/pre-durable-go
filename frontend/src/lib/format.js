/** A duration as a person says it: 40 ms, 1.2 s, 3m 05s, 2h 10m. */
export function duration(ms) {
  if (ms == null || Number.isNaN(ms)) return "";
  if (ms < 1000) return `${Math.max(0, Math.round(ms))} ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(ms < 10_000 ? 1 : 0)} s`;

  const minutes = Math.floor(ms / 60_000);
  if (minutes < 60) return `${minutes}m ${String(Math.floor((ms % 60_000) / 1000)).padStart(2, "0")}s`;

  return `${Math.floor(minutes / 60)}h ${String(minutes % 60).padStart(2, "0")}m`;
}

/** A moment as a clock time with milliseconds, which is what matters in a trace. */
export function clock(time) {
  if (!time) return "";
  const date = new Date(time);

  return `${date.toLocaleTimeString([], { hour12: false })}.${String(date.getMilliseconds()).padStart(3, "0")}`;
}

/** The last part of an ARN or a long id, which is the part that differs. */
export function tail(id, length = 12) {
  if (!id) return "";
  const last = id.split("/").pop();

  return last.length > length ? `…${last.slice(-length)}` : last;
}

export function plural(count, word) {
  return `${count} ${word}${count === 1 ? "" : "s"}`;
}
