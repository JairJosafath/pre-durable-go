import { useState, useEffect, useCallback } from "react";
import { api, fileName } from "@/lib/api";

const POLL_INTERVAL_MS = 3000;

/** Reads a DynamoDB file item, as returned by GET /files. */
function toFile(item) {
  return {
    key: item.key.S,
    title: fileName(item.key.S),
    status: item.status?.S,
    labels: (item.labels?.L || []).map((label) => label.M.name.S),
    reasons: (item.review?.M?.reasons?.L || []).map((reason) => reason.S),
  };
}

/** Keeps the first occurrence of each file: a prefix search can match one file under several labels. */
function unique(files) {
  const seen = new Set();
  return files.filter((file) => !seen.has(file.key) && seen.add(file.key));
}

/**
 * Lists the user's files, or the files matching a label search, and loads
 * their thumbnails.
 */
export default function useThumber(token, search) {
  const [files, setFiles] = useState([]);
  const [thumbs, setThumbs] = useState({});
  const [loading, setLoading] = useState(true);
  const [photo, setPhoto] = useState({ title: "", url: "" });
  const [reload, setReload] = useState(false);

  // The API returns one page at a time. `next` is the sort key of the last
  // item received; sending it back asks for the page after it.
  const [next, setNext] = useState(null);

  // Keys of uploads the user just approved or rejected. The decision resumes
  // the suspended execution in the background, so the catalog still says
  // PENDING_REVIEW for a moment. Until the outcome is in, such a file counts
  // as being processed, not as waiting for a review.
  const [decided, setDecided] = useState([]);

  const fetchPage = useCallback(
    async (after) => {
      const params = new URLSearchParams();
      if (search) params.set("index", search);
      if (after) params.set("next", after);

      const response = await api(`${search ? "/query" : "/files"}?${params}`, { token });
      const data = await response.json();
      const items = data.Items || [];

      // A search returns one index entry per label and file; a listing returns
      // the file records themselves, with their pipeline status.
      const found = search
        ? items.map((item) => ({ key: item.key.S, title: fileName(item.key.S), status: "READY", labels: [], reasons: [] }))
        : items.map(toFile);

      return { found, next: data.LastEvaluatedKey?.sk?.S || null };
    },
    [token, search],
  );

  const load = useCallback(async () => {
    if (!token) return;

    const page = await fetchPage(null);

    const awaitsReview = (key) => page.found.some((file) => file.key === key && file.status === "PENDING_REVIEW");

    setFiles(unique(page.found));
    setDecided((keys) => keys.filter(awaitsReview));
    setNext(page.next);
    setLoading(false);
    setReload(false);
  }, [token, fetchPage]);

  const loadMore = useCallback(async () => {
    if (!token || !next) return;

    const page = await fetchPage(next);

    setFiles((current) => unique([...current, ...page.found]));
    setNext(page.next);
  }, [token, next, fetchPage]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (reload) load();
  }, [reload, load]);

  const expectOutcome = useCallback((key) => {
    setDecided((keys) => [...keys, key]);
    setReload(true);
  }, []);

  const awaitingOutcome = (file) => file.status === "PENDING_REVIEW" && decided.includes(file.key);
  const inProgress = (file) => file.status === "PROCESSING" || awaitingOutcome(file);

  // While the pipeline is still working on an upload, keep looking.
  const processing = files.some(inProgress);
  useEffect(() => {
    if (!processing) return undefined;

    const timer = setInterval(load, POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [processing, load]);

  // Thumbnails exist once the pipeline is past analysis.
  const withThumbnail = files
    .filter((file) => file.status === "READY" || file.status === "PENDING_REVIEW")
    .map((file) => file.title)
    .join("|");

  useEffect(() => {
    if (!token) return;

    withThumbnail
      .split("|")
      .filter(Boolean)
      .forEach(async (title) => {
        // API Gateway passes an image through as bytes only when the request
        // accepts one of its binary media types (image/*). A browser's default,
        // */*, does not count, and the image would arrive as mangled text.
        const response = await api(`/object/${encodeURIComponent(title)}`, { token, accept: "image/*" });
        if (!response.ok) return;

        const url = URL.createObjectURL(await response.blob());
        setThumbs((current) => ({ ...current, [title]: url }));
      });
  }, [withThumbnail, token]);

  // Opening a photo fetches a presigned URL for the original.
  useEffect(() => {
    if (!photo?.title || photo.url) return;

    async function getOriginal() {
      const response = await api(`/presigned?action=get&key=${encodeURIComponent(photo.title)}`, { token });
      const { url } = await response.json();
      setPhoto((current) => ({ ...current, url }));
    }

    getOriginal();
  }, [photo?.title, photo?.url, token]);

  const withUrl = (file) => ({ ...file, url: thumbs[file.title] });

  return {
    thumbs: files.filter((file) => file.status === "READY").map(withUrl),
    pending: files.filter((file) => file.status === "PENDING_REVIEW" && !awaitingOutcome(file)).map(withUrl),
    processing: files.filter(inProgress),
    failed: files.filter((file) => file.status === "FAILED"),
    expectOutcome,
    loading,
    hasMore: Boolean(next),
    loadMore,
    photo,
    setPhoto,
    setReload,
  };
}
