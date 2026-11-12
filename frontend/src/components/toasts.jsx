import { useCallback, useState } from "react";
import { AlertIcon } from "@/components/icons";

const SHOWN_MS = 6000;

const TONES = {
  warn: "border-warn/30 text-warn",
  bad: "border-bad/30 text-bad",
};

/** Short messages about things that happened off screen, such as a rejection. */
export function useToasts() {
  const [toasts, setToasts] = useState([]);

  const notify = useCallback(({ tone, text }) => {
    const id = `${Date.now()}-${Math.random()}`;

    setToasts((current) => [...current, { id, tone, text }]);
    setTimeout(() => setToasts((current) => current.filter((toast) => toast.id !== id)), SHOWN_MS);
  }, []);

  return { toasts, notify };
}

export default function Toasts({ toasts }) {
  return (
    <div className="pointer-events-none fixed bottom-5 left-1/2 z-50 flex -translate-x-1/2 flex-col items-center gap-2" aria-live="polite">
      {toasts.map((toast) => (
        <p
          key={toast.id}
          className={`flex max-w-[90vw] animate-rise items-center gap-2 rounded-xl border bg-raised px-4 py-2.5 text-sm shadow-xl shadow-black/50 ${TONES[toast.tone]}`}
        >
          <AlertIcon className="h-4 w-4 shrink-0" />
          <span className="text-fg">{toast.text}</span>
        </p>
      ))}
    </div>
  );
}
