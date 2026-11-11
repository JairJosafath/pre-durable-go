import { useState, useRef, useCallback } from "react";
import { api } from "@/lib/api";

const DEBOUNCE_MS = 200;

/** Suggests the user's own labels that start with what they typed. */
export default function useLabels(token) {
  const [labels, setLabels] = useState([]);
  const timer = useRef(null);

  const suggest = useCallback(
    (prefix) => {
      clearTimeout(timer.current);

      timer.current = setTimeout(async () => {
        if (!token) return;

        const response = await api(`/labels?prefix=${encodeURIComponent(prefix)}`, { token });
        if (!response.ok) return;

        const data = await response.json();
        setLabels((data.Items || []).map((item) => item.label.S));
      }, DEBOUNCE_MS);
    },
    [token],
  );

  return { labels, suggest };
}
