import { useState } from "react";
import { api } from "@/lib/api";

/**
 * Uploads the pipeline could not publish on its own. Each one is a durable
 * execution that is suspended until one of these buttons is pressed.
 */
export default function ReviewPanel({ token, pending, onDecided }) {
  const [busy, setBusy] = useState("");

  if (!pending?.length) return null;

  async function decide(file, approved) {
    setBusy(file.key);
    await api("/reviews", { token, method: "POST", body: { key: file.key, approved } });
    setBusy("");
    onDecided();
  }

  return (
    <section className="mx-12 my-4 rounded bg-neutral-900 p-4">
      <h2 className="mb-3 text-xl font-bold">Waiting for your review</h2>
      <ul className="flex flex-col gap-3">
        {pending.map((file) => (
          <li key={file.key} className="flex items-center gap-4">
            {file.url && <img src={file.url} alt={file.title} className="h-16 w-16 object-cover" />}
            <div className="flex-1">
              <p className="truncate">{file.title}</p>
              <p className="text-sm text-neutral-400">
                {file.labels.join(", ")} — {file.reasons.join("; ")}
              </p>
            </div>
            <button
              className="rounded bg-green-700 px-3 py-1 disabled:opacity-50"
              disabled={busy === file.key}
              onClick={() => decide(file, true)}
            >
              Approve
            </button>
            <button
              className="rounded bg-red-700 px-3 py-1 disabled:opacity-50"
              disabled={busy === file.key}
              onClick={() => decide(file, false)}
            >
              Reject
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}
