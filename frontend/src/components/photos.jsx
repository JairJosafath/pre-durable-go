import { useEffect, useRef, useState } from "react";
import { AlertIcon, CodeIcon, PhotoIcon, Spinner } from "@/components/icons";

// What the tile of an upload says while the pipeline is not done with it.
const PHASES = {
  UPLOADING: "Uploading",
  STARTING: "Starting",
  PROCESSING: "Analysing",
  PUBLISHING: "Publishing",
  REMOVING: "Removing",
};

/** A photo, fading in once its bytes are there. */
function Picture({ src, alt, className = "" }) {
  const [shown, setShown] = useState(false);

  return (
    <img
      src={src}
      alt={alt}
      draggable={false}
      onLoad={() => setShown(true)}
      className={`h-full w-full object-cover transition duration-500 ${shown ? "opacity-100" : "opacity-0"} ${className}`}
    />
  );
}

/** An upload the pipeline is still working on. In dev mode it names the step. */
function WorkingTile({ file, step, selected, onInspect }) {
  const Tag = onInspect ? "button" : "div";

  return (
    <Tag
      type={onInspect ? "button" : undefined}
      onClick={onInspect}
      className={`relative aspect-square animate-pop overflow-hidden rounded-2xl border bg-gradient-to-br from-hover to-surface text-left ${
        selected ? "border-accent/70" : "border-line"
      }`}
    >
      {file.preview ? (
        <Picture src={file.preview} alt="" className="scale-105 blur-[2px] brightness-50" />
      ) : (
        <PhotoIcon className="absolute left-1/2 top-[38%] h-9 w-9 -translate-x-1/2 -translate-y-1/2 text-faint" />
      )}
      <div className="sheen absolute inset-0" />
      <div className="absolute inset-x-0 bottom-0 flex flex-col gap-1 p-3">
        <span className="flex items-center gap-2 text-sm font-medium">
          <Spinner className="h-3.5 w-3.5 text-accent" />
          {PHASES[file.phase]}
        </span>
        {step && <span className="truncate font-mono text-[11px] text-accent">{step}</span>}
        <span className="truncate text-xs text-muted">{file.title}</span>
      </div>
    </Tag>
  );
}

/** An upload the pipeline gave up on. The original is kept, so uploading again retries. */
function FailedTile({ file, selected, onInspect }) {
  return (
    <div
      className={`relative flex aspect-square animate-pop flex-col justify-between overflow-hidden rounded-2xl border bg-bad/[.06] p-3 ${
        selected ? "border-accent/70" : "border-bad/30"
      }`}
    >
      <div className="flex items-start justify-between">
        <AlertIcon className="h-5 w-5 text-bad" />
        {onInspect && <InspectButton onClick={onInspect} shown />}
      </div>
      <div>
        <p className="text-sm font-medium text-bad">Could not be processed</p>
        <p className="mt-0.5 text-xs text-muted">Upload it again to retry.</p>
        <p className="mt-2 truncate text-xs text-muted">{file.title}</p>
      </div>
    </div>
  );
}

function InspectButton({ onClick, shown }) {
  return (
    <button
      type="button"
      title="Show this upload's execution"
      className={`rounded-lg border border-white/10 bg-black/60 p-1.5 text-accent backdrop-blur transition hover:bg-black/80 ${
        shown ? "" : "opacity-0 focus:opacity-100 group-hover:opacity-100"
      }`}
      onClick={(event) => {
        event.stopPropagation();
        onClick();
      }}
    >
      <CodeIcon className="h-4 w-4" />
    </button>
  );
}

function ReadyTile({ file, src, selected, onOpen, onInspect }) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(event) => (event.key === "Enter" || event.key === " ") && onOpen()}
      className={`group relative aspect-square cursor-zoom-in overflow-hidden rounded-2xl border bg-raised transition duration-300 hover:-translate-y-0.5 hover:shadow-xl hover:shadow-black/50 ${
        selected ? "border-accent/70" : "border-line hover:border-line-strong"
      }`}
    >
      {src || file.preview ? (
        <Picture src={src || file.preview} alt={file.title} className="group-hover:scale-[1.04]" />
      ) : (
        <div className="sheen h-full w-full" />
      )}

      {onInspect && (
        <div className="absolute right-2 top-2">
          <InspectButton onClick={onInspect} shown={selected} />
        </div>
      )}

      <div className="pointer-events-none absolute inset-x-0 bottom-0 translate-y-1 bg-gradient-to-t from-black/85 via-black/45 to-transparent p-3 pt-10 opacity-0 transition duration-300 group-hover:translate-y-0 group-hover:opacity-100 group-focus-visible:opacity-100">
        <p className="truncate text-sm font-medium">{file.title}</p>
        {file.labels?.length > 0 && <p className="truncate text-xs text-white/65">{file.labels.slice(0, 4).join(" · ")}</p>}
      </div>
    </div>
  );
}

/** Loads the next page when the end of the grid scrolls into view. */
function MoreWhenSeen({ onSeen }) {
  const marker = useRef(null);

  useEffect(() => {
    const observer = new IntersectionObserver(([entry]) => entry.isIntersecting && onSeen(), { rootMargin: "400px" });
    observer.observe(marker.current);

    return () => observer.disconnect();
  }, [onSeen]);

  return (
    <div ref={marker} className="flex justify-center py-8 text-muted">
      <Spinner className="h-5 w-5" />
    </div>
  );
}

/**
 * The photos, with the uploads that are not photos yet in front: the ones
 * the pipeline is working on, then the ones it gave up on.
 */
export default function Photos({ working = [], failed = [], ready, thumbnail, onOpen, hasMore, loadMore, inspect }) {
  return (
    <>
      <div className="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3 sm:grid-cols-[repeat(auto-fill,minmax(190px,1fr))]">
        {working.map((file) => (
          <WorkingTile
            key={file.key}
            file={file}
            step={inspect?.step(file)}
            selected={inspect?.selected === file.title}
            onInspect={inspect ? () => inspect.select(file.title) : undefined}
          />
        ))}
        {failed.map((file) => (
          <FailedTile
            key={file.key}
            file={file}
            selected={inspect?.selected === file.title}
            onInspect={inspect ? () => inspect.select(file.title) : undefined}
          />
        ))}
        {ready.map((file) => (
          <ReadyTile
            key={file.key}
            file={file}
            src={thumbnail(file)}
            selected={inspect?.selected === file.title}
            onOpen={() => onOpen(file)}
            onInspect={inspect ? () => inspect.select(file.title) : undefined}
          />
        ))}
      </div>
      {hasMore && <MoreWhenSeen onSeen={loadMore} />}
    </>
  );
}

/** Shown instead of the grid while the first answer is on its way. */
export function PhotosLoading() {
  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3 sm:grid-cols-[repeat(auto-fill,minmax(190px,1fr))]">
      {Array.from({ length: 12 }, (_, index) => (
        <div key={index} className="sheen aspect-square rounded-2xl border border-line bg-raised" style={{ animationDelay: `${index * 60}ms` }} />
      ))}
    </div>
  );
}

export function Empty({ title, children }) {
  return (
    <div className="mx-auto flex max-w-md animate-rise flex-col items-center py-24 text-center">
      <div className="mb-5 flex h-16 w-16 items-center justify-center rounded-2xl border border-line bg-surface text-muted">
        <PhotoIcon className="h-7 w-7" />
      </div>
      <h2 className="text-lg font-semibold">{title}</h2>
      <p className="mt-1.5 text-[15px] leading-relaxed text-muted">{children}</p>
    </div>
  );
}
