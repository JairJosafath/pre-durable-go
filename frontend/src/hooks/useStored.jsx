import { useState, useEffect, useCallback } from "react";

/** A setting that survives a reload, such as whether dev mode is on. */
export default function useStored(name, fallback) {
  const [value, setValue] = useState(fallback);

  useEffect(() => {
    try {
      const stored = localStorage.getItem(name);
      if (stored !== null) setValue(JSON.parse(stored));
    } catch {
      // A value nobody can read is as good as no value.
    }
  }, [name]);

  const store = useCallback(
    (next) => {
      setValue((current) => {
        const resolved = typeof next === "function" ? next(current) : next;
        localStorage.setItem(name, JSON.stringify(resolved));

        return resolved;
      });
    },
    [name],
  );

  return [value, store];
}
