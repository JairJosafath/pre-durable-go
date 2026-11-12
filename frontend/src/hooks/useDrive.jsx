import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { api, toFile } from "@/lib/api";

// The page keeps itself up to date by asking again: often while the pipeline
// is working on something, rarely when nothing is going on, and not at all
// while the tab is in the background.
const BUSY_MS = 1500;
const IDLE_MS = 8000;
const HEARTBEAT_MS = 400;

// The API answers one page of 24 files at a time, in the order of their
// names. The first few pages are loaded straight away, so a drive of ordinary
// size is complete: every review that waits is on screen, wherever its name
// sorts. More pages load as the user scrolls.
const FIRST_PAGES = 4;
// A file that was just uploaded can sort onto a page that is not loaded yet.
// Looking for it goes this far.
const SEARCH_PAGES = 25;

// An upload that the pipeline has not picked up after this long went wrong.
const PICKUP_TIMEOUT_MS = 60_000;

const UNAUTHORIZED = 401;

/**
 * The user's drive, kept live: every upload with where it is in the
 * pipeline, plus the two things a user can do to it, upload and review.
 */
export default function useDrive(token, { onExpired, notify }) {
  const [files, setFiles] = useState([]);
  const [loaded, setLoaded] = useState(false);
  const [offline, setOffline] = useState(false);
  const [hasMore, setHasMore] = useState(false);

  // Uploads on their way: file name -> { preview, previousRun, phase, since }.
  const [sending, setSending] = useState({});
  // Reviews the user just answered: key -> approved. The decision resumes the
  // suspended execution in the background, so the catalog says PENDING_REVIEW
  // for a moment longer.
  const [decided, setDecided] = useState({});
  // A picture of every file the user uploaded in this session, to show until
  // the pipeline has made the thumbnail.
  const previews = useRef({});

  const pages = useRef(FIRST_PAGES);
  const expected = useRef(sending);
  expected.current = sending;
  const asking = useRef(false);
  const askedAt = useRef(0);
  const busy = useRef(false);
  const known = useRef(files);
  known.current = files;

  const refresh = useCallback(async () => {
    if (!token || asking.current) return;
    asking.current = true;

    try {
      const found = [];
      let after = null;

      // Uploads of this session whose new record has not been seen yet.
      const missing = () =>
        Object.entries(expected.current).some(
          ([title, mine]) => !found.some((file) => file.title === title && file.run !== mine.previousRun),
        );

      // Asking for as many pages as are on screen keeps everything on screen
      // up to date. A fresh upload is looked for beyond that.
      for (let page = 0; page < pages.current || (page < SEARCH_PAGES && missing()); page++) {
        const response = await api(`/files${after ? `?next=${encodeURIComponent(after)}` : ""}`, { token });
        if (response.status === UNAUTHORIZED) return onExpired();
        if (!response.ok) throw new Error(`GET /files answered ${response.status}`);

        const data = await response.json();
        found.push(...(data.Items || []).map(toFile));
        after = data.LastEvaluatedKey?.sk?.S || null;
        // A page that was needed to find an upload stays loaded.
        pages.current = Math.max(pages.current, page + 1);
        if (!after) break;
      }

      setFiles(found);
      setHasMore(Boolean(after));
      setLoaded(true);
      setOffline(false);
    } catch {
      setOffline(true);
    } finally {
      asking.current = false;
      askedAt.current = Date.now();
    }
  }, [token, onExpired]);

  /** Asks again at the next heartbeat instead of waiting for the interval. */
  const soon = useCallback(() => {
    askedAt.current = 0;
  }, []);

  useEffect(() => {
    if (!token) return undefined;

    refresh();

    const timer = setInterval(() => {
      if (document.hidden) return;
      if (Date.now() - askedAt.current >= (busy.current ? BUSY_MS : IDLE_MS)) refresh();
    }, HEARTBEAT_MS);

    const onVisible = () => !document.hidden && refresh();
    document.addEventListener("visibilitychange", onVisible);

    return () => {
      clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [token, refresh]);

  const loadMore = useCallback(() => {
    pages.current += 1;
    soon();
  }, [soon]);

  const forget = useCallback((title) => {
    setSending(({ [title]: _, ...rest }) => rest);
  }, []);

  // The browser uploads straight to S3. The object landing in the bucket is
  // what starts the durable function.
  const upload = useCallback(
    async (file) => {
      const title = file.name;
      const preview = file.type.startsWith("image/") ? URL.createObjectURL(file) : "";
      const previousRun = known.current.find((other) => other.title === title)?.run || "";

      previews.current[title] = preview;
      setSending((current) => ({ ...current, [title]: { preview, previousRun, phase: "UPLOADING", since: Date.now() } }));

      try {
        const response = await api(`/presigned?action=put&key=${encodeURIComponent(title)}`, { token });
        if (response.status === UNAUTHORIZED) return onExpired();
        const { url } = await response.json();

        const put = await fetch(url, {
          method: "PUT",
          body: file,
          headers: { "Content-Type": file.type || "application/octet-stream" },
        });
        if (!put.ok) throw new Error(`upload answered ${put.status}`);

        setSending((current) => ({ ...current, [title]: { ...current[title], phase: "STARTING", since: Date.now() } }));
        soon();
      } catch {
        forget(title);
        notify({ tone: "bad", text: `${title} could not be uploaded.` });
      }
    },
    [token, onExpired, notify, soon, forget],
  );

  // Answering a review resumes the durable execution that waits for it.
  const decide = useCallback(
    async (file, approved) => {
      let response;
      try {
        response = await api("/reviews", { token, method: "POST", body: { key: file.key, approved } });
      } catch {
        response = null;
      }

      if (response?.status === UNAUTHORIZED) return onExpired();
      if (!response?.ok) {
        notify({ tone: "bad", text: `The decision about ${file.title} did not arrive. Try again.` });
        return;
      }

      setDecided((current) => ({ ...current, [file.key]: approved }));
      soon();
    },
    [token, onExpired, notify, soon],
  );

  // What is on screen: the catalog's records, with the uploads that are still
  // on their way laid over them.
  const view = useMemo(() => {
    const shown = [];
    const covered = new Set();

    for (const file of files) {
      const mine = sending[file.title];
      // A record with the run from before this upload is the old photo.
      if (mine && file.run === mine.previousRun) continue;

      covered.add(file.title);
      const answered = file.status === "PENDING_REVIEW" && file.key in decided;
      let phase = file.status;
      if (answered) phase = decided[file.key] ? "PUBLISHING" : "REMOVING";

      shown.push({ ...file, phase, preview: previews.current[file.title] || "" });
    }

    for (const [title, mine] of Object.entries(sending)) {
      if (covered.has(title)) continue;
      shown.push({ key: `sending:${title}`, title, status: "", run: "", labels: [], reasons: [], note: "", phase: mine.phase, preview: mine.preview });
    }

    return shown;
  }, [files, sending, decided]);

  const working = (file) => ["UPLOADING", "STARTING", "PROCESSING", "PUBLISHING", "REMOVING"].includes(file.phase);
  busy.current = view.some(working);

  // Tidy up once the catalog has caught up with what the user did.
  useEffect(() => {
    for (const [title, mine] of Object.entries(sending)) {
      const record = files.find((file) => file.title === title && file.run !== mine.previousRun);

      if (record && record.status !== "PROCESSING") forget(title);
      else if (!record && mine.phase === "STARTING" && Date.now() - mine.since > PICKUP_TIMEOUT_MS) {
        forget(title);
        notify({ tone: "bad", text: `${title} was uploaded, but the pipeline did not pick it up.` });
      }
    }

    const stillWaiting = (key) => files.some((file) => file.key === key && file.status === "PENDING_REVIEW");
    if (Object.keys(decided).some((key) => !stillWaiting(key))) {
      setDecided((current) => Object.fromEntries(Object.entries(current).filter(([key]) => stillWaiting(key))));
    }
  }, [files, sending, decided, forget, notify]);

  // A rejected upload is deleted and leaves the page. Say so, or it would
  // just vanish.
  const before = useRef(null);
  useEffect(() => {
    if (!loaded) return;

    const statuses = Object.fromEntries(files.map((file) => [`${file.key}|${file.run}`, file]));
    if (before.current) {
      for (const [id, file] of Object.entries(statuses)) {
        const was = before.current[id]?.status ?? (file.title in previews.current ? "NEW" : null);
        if (file.status === "REJECTED" && was && was !== "REJECTED") {
          notify({ tone: "warn", text: `${file.title} was rejected and removed${file.note ? `: ${file.note}` : ""}.` });
        }
      }
    }
    before.current = statuses;
  }, [files, loaded, notify]);

  return {
    // Everything the catalog knows, for dev mode.
    all: view,
    working: view.filter(working),
    reviews: view.filter((file) => file.phase === "PENDING_REVIEW"),
    failed: view.filter((file) => file.phase === "FAILED"),
    ready: view.filter((file) => file.phase === "READY"),
    loaded,
    offline,
    hasMore,
    loadMore,
    upload,
    decide,
  };
}
