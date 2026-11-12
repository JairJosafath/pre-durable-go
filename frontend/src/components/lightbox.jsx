import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { CloseIcon, DownloadIcon, Spinner } from "@/components/icons";

/**
 * One photo at full size. The thumbnail is shown at once, and the original
 * replaces it when its presigned URL has loaded.
 */
export default function Lightbox({ token, file, thumbnail, onClose, onLabel }) {
  const [original, setOriginal] = useState("");
  const [arrived, setArrived] = useState(false);

  useEffect(() => {
    let current = true;

    api(`/presigned?action=get&key=${encodeURIComponent(file.title)}`, { token })
      .then((response) => response.json())
      .then(({ url }) => current && setOriginal(url || ""))
      .catch(() => {});

    return () => {
      current = false;
    };
  }, [file.title, token]);

  useEffect(() => {
    const onKey = (event) => event.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex animate-fade flex-col bg-black/90 backdrop-blur-md" role="dialog" aria-modal="true" aria-label={file.title} onClick={onClose}>
      <div className="flex items-center gap-3 px-5 py-4" onClick={(event) => event.stopPropagation()}>
        <p className="min-w-0 flex-1 truncate font-medium">{file.title}</p>
        <button
          type="button"
          disabled={!original}
          className="flex items-center gap-2 rounded-xl border border-white/15 px-3 py-2 text-sm transition hover:bg-white/10 disabled:opacity-40"
          onClick={() => window.open(original, "_blank")}
        >
          <DownloadIcon className="h-4 w-4" />
          Download
        </button>
        <button type="button" aria-label="Close" className="rounded-xl border border-white/15 p-2 transition hover:bg-white/10" onClick={onClose}>
          <CloseIcon className="h-5 w-5" />
        </button>
      </div>

      <div className="relative flex min-h-0 flex-1 items-center justify-center px-5">
        {/* The thumbnail holds the place, blurred, until the original is in. */}
        {thumbnail && !arrived && <img src={thumbnail} alt="" className="absolute max-h-full max-w-full scale-[1.02] rounded-lg object-contain blur-md" />}
        {!arrived && <Spinner className="absolute h-7 w-7 text-white/80" />}
        {original && (
          <img
            src={original}
            alt={file.title}
            onLoad={() => setArrived(true)}
            onClick={(event) => event.stopPropagation()}
            className={`relative max-h-full max-w-full animate-pop rounded-lg object-contain shadow-2xl transition-opacity duration-300 ${arrived ? "opacity-100" : "opacity-0"}`}
          />
        )}
      </div>

      <div className="flex min-h-[64px] flex-wrap items-center justify-center gap-2 px-5 py-4" onClick={(event) => event.stopPropagation()}>
        {(file.labels || []).map((label) => (
          <button key={label} type="button" className="chip" title={`Find photos with ${label}`} onClick={() => onLabel(label)}>
            {label}
          </button>
        ))}
      </div>
    </div>
  );
}
