import { useState, useEffect, useCallback, useRef } from "react";
import { api, fileName } from "@/lib/api";

const DEBOUNCE_MS = 220;

/** Keeps the first occurrence of each file: a prefix search can match one file under several labels. */
function unique(found) {
  const seen = new Set();

  return found.filter((file) => !seen.has(file.key) && seen.add(file.key));
}

/**
 * Finds the user's photos by label, as they type. A search is one DynamoDB
 * query on a sort-key prefix, so "ca" finds cats, cars and castles.
 *
 * Also suggests the user's own labels that start with what was typed.
 */
export default function useSearch(token, term) {
  const [results, setResults] = useState([]);
  const [labels, setLabels] = useState([]);
  const [searching, setSearching] = useState(false);
  const [next, setNext] = useState(null);
  // Answers can arrive out of order. Only the latest question counts.
  const question = useRef(0);

  const ask = useCallback(
    async (after) => {
      const params = new URLSearchParams({ index: term });
      if (after) params.set("next", after);

      const response = await api(`/query?${params}`, { token });
      const data = response.ok ? await response.json() : {};
      // A search returns one index entry per label and file.
      const found = (data.Items || []).map((item) => ({ key: item.key.S, title: fileName(item.key.S), label: item.label?.S || "" }));

      return { found, next: data.LastEvaluatedKey?.sk?.S || null };
    },
    [token, term],
  );

  useEffect(() => {
    if (!token || !term) {
      setResults([]);
      setLabels([]);
      setNext(null);
      setSearching(false);

      return undefined;
    }

    const mine = ++question.current;
    setSearching(true);

    const timer = setTimeout(async () => {
      const [page, suggestions] = await Promise.all([
        ask(null),
        api(`/labels?prefix=${encodeURIComponent(term)}`, { token })
          .then((response) => (response.ok ? response.json() : {}))
          .catch(() => ({})),
      ]).catch(() => [{ found: [], next: null }, {}]);

      if (mine !== question.current) return;

      setResults(unique(page.found));
      setNext(page.next);
      setLabels((suggestions.Items || []).map((item) => item.label.S));
      setSearching(false);
    }, DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [token, term, ask]);

  const loadMore = useCallback(async () => {
    if (!next) return;

    const mine = question.current;
    const page = await ask(next);
    if (mine !== question.current) return;

    setResults((current) => unique([...current, ...page.found]));
    setNext(page.next);
  }, [next, ask]);

  return { results, labels, searching, hasMore: Boolean(next), loadMore };
}
