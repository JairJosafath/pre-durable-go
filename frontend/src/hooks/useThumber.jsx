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

  const load = useCallback(async () => {
    if (!token) return;

    const path = search ? `/query?index=${encodeURIComponent(search)}` : "/files";
    const response = await api(path, { token });
    const data = await response.json();
    const items = data.Items || [];

    // A search returns one index entry per label and file; a listing returns
    // the file records themselves, with their pipeline status.
    const found = search
      ? [...new Set(items.map((item) => item.key.S))].map((key) => ({ key, title: fileName(key), status: "READY", labels: [], reasons: [] }))
      : items.map(toFile);

    setFiles(found);
    setLoading(false);
    setReload(false);
  }, [token, search]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (reload) load();
  }, [reload, load]);

  // While the pipeline is still working on an upload, keep looking.
  const processing = files.some((file) => file.status === "PROCESSING");
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
        const response = await api(`/object/${encodeURIComponent(title)}`, { token });
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
    pending: files.filter((file) => file.status === "PENDING_REVIEW").map(withUrl),
    processing: files.filter((file) => file.status === "PROCESSING"),
    loading,
    photo,
    setPhoto,
    setReload,
  };
}
