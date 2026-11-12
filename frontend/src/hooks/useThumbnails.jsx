import { useEffect, useReducer, useRef, useCallback } from "react";
import { api } from "@/lib/api";

// A thumbnail that is not there yet is asked for again, but not forever.
const MAX_ATTEMPTS = 6;

// Uploading a file name again makes a new thumbnail, so the run is part of
// what identifies one.
const idOf = (file) => `${file.key}|${file.run || ""}`;

/**
 * Loads the thumbnails of the given files and remembers them. Returns a
 * function that gives the URL of a file's thumbnail, or "" while there is none.
 *
 * The API wants an Authorization header, which an <img> cannot send. So each
 * thumbnail is fetched and handed to the image as an object URL.
 */
export default function useThumbnails(token, files) {
  const urls = useRef(new Map());
  const attempts = useRef(new Map());
  const [, redraw] = useReducer((count) => count + 1, 0);

  useEffect(() => {
    if (!token) return;

    for (const file of files) {
      const id = idOf(file);
      const tried = attempts.current.get(id) || 0;
      if (urls.current.has(id) || tried >= MAX_ATTEMPTS) continue;

      urls.current.set(id, "");
      attempts.current.set(id, tried + 1);

      // API Gateway passes an image through as bytes only when the request
      // accepts one of its binary media types (image/*). A browser's default,
      // */*, does not count, and the image would arrive as mangled text.
      api(`/object/${encodeURIComponent(file.title)}`, { token, accept: "image/*" })
        .then(async (response) => {
          if (!response.ok) throw new Error("no thumbnail yet");
          urls.current.set(id, URL.createObjectURL(await response.blob()));
          redraw();
        })
        .catch(() => urls.current.delete(id));
    }
  }, [files, token]);

  return useCallback((file) => urls.current.get(idOf(file)) || "", []);
}
