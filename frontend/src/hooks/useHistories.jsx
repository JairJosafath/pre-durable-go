import { useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";

const HEARTBEAT_MS = 300;
// While Lambda still says RUNNING, keep looking, whatever the caller thinks.
const RUNNING_MS = 1500;

/**
 * Dev mode: the durable execution history of some uploads, kept live.
 *
 * `wanted` lists the uploads to watch: { title, every, stamp }. `every` is how
 * often to ask again in milliseconds, 0 for once. A changed `stamp` (the
 * upload's run and status) asks again at once, so the last step of an
 * execution is not missed.
 *
 * Returns title -> { state: "loading" | "ok" | "missing" | "error", history, message, at }.
 */
export default function useHistories(token, wanted) {
  const [histories, setHistories] = useState({});
  const asked = useRef({}); // title -> { at, stamp, asking }
  const list = useRef(wanted);
  list.current = wanted;

  const watching = wanted.length > 0;

  useEffect(() => {
    if (!token || !watching) return undefined;

    const ask = async (title, stamp) => {
      asked.current[title] = { ...asked.current[title], asking: true, stamp };

      let entry;
      try {
        const response = await api(`/executions?key=${encodeURIComponent(title)}`, { token });
        const body = await response.json().catch(() => ({}));

        if (response.ok) entry = { state: "ok", history: body };
        else if (response.status === 404) entry = { state: "missing", message: body.message || "No history for this upload." };
        else entry = { state: "error", message: body.message || `GET /executions answered ${response.status}` };
      } catch {
        entry = { state: "error", message: "The API cannot be reached." };
      }

      asked.current[title] = { at: Date.now(), stamp, asking: false, running: entry.history?.execution?.status === "RUNNING" };
      // A failed look keeps the last good history on screen.
      setHistories((current) => ({
        ...current,
        [title]: entry.state === "error" && current[title]?.history ? { ...current[title], at: Date.now() } : { ...entry, at: Date.now() },
      }));
    };

    const beat = () => {
      if (document.hidden) return;

      for (const { title, every, stamp } of list.current) {
        const last = asked.current[title];
        if (last?.asking) continue;

        const interval = last?.running ? Math.min(every || RUNNING_MS, RUNNING_MS) : every;
        const due = !last || last.stamp !== stamp || (interval > 0 && Date.now() - last.at >= interval);
        if (due) ask(title, stamp);
      }
    };

    beat();
    const timer = setInterval(beat, HEARTBEAT_MS);

    return () => clearInterval(timer);
  }, [token, watching]);

  return histories;
}
