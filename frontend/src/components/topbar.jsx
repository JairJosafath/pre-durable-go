import { useEffect, useRef, useState } from "react";
import { Logo, SearchIcon, UploadIcon, CloseIcon, CodeIcon, SignOutIcon } from "@/components/icons";

/** The user's initial. Where there is a session to end, it opens a menu to end it. */
function Account({ username, onSignOut }) {
  const [open, setOpen] = useState(false);
  const menu = useRef(null);

  useEffect(() => {
    if (!open) return undefined;

    const away = (event) => !menu.current?.contains(event.target) && setOpen(false);
    const escape = (event) => event.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", away);
    document.addEventListener("keydown", escape);

    return () => {
      document.removeEventListener("mousedown", away);
      document.removeEventListener("keydown", escape);
    };
  }, [open]);

  const face = "flex h-10 w-10 items-center justify-center rounded-full border border-line bg-gradient-to-br from-raised to-hover text-sm font-semibold text-fg";
  const initial = username?.[0]?.toUpperCase();

  if (!onSignOut) {
    return (
      <span title={`Signed in as ${username}`} className={`hidden sm:flex ${face}`}>
        {initial}
      </span>
    );
  }

  return (
    <div ref={menu} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        title={`Signed in as ${username}`}
        className={`${face} transition hover:border-line-strong ${open ? "border-accent/60" : ""}`}
        onClick={() => setOpen((shown) => !shown)}
      >
        {initial}
      </button>

      {open && (
        <div role="menu" className="absolute right-0 top-12 z-40 w-56 animate-pop rounded-2xl border border-line bg-raised p-1.5 shadow-2xl shadow-black/60">
          <p className="truncate px-3 pb-2 pt-1.5 text-sm text-muted">
            Signed in as <span className="font-medium text-fg">{username}</span>
          </p>
          <button
            type="button"
            role="menuitem"
            className="flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-left text-sm transition hover:bg-hover"
            onClick={onSignOut}
          >
            <SignOutIcon className="h-[18px] w-[18px] text-muted" />
            Sign out
          </button>
        </div>
      )}
    </div>
  );
}

/** The bar that stays on top: search in the middle, upload and dev mode on the right. */
export default function TopBar({ username, onSignOut, search, setSearch, onUpload, dev, setDev, offline }) {
  const picker = useRef(null);
  const box = useRef(null);

  return (
    <header className="sticky top-0 z-30 border-b border-line/70 bg-canvas/75 backdrop-blur-xl">
      <div className="flex h-16 items-center gap-4 px-5 sm:px-8">
        <button type="button" className="flex shrink-0 items-center gap-2.5" onClick={() => setSearch("")} title="All photos">
          <Logo />
          <span className="hidden text-[17px] font-semibold tracking-tight sm:block">PrimeDrive</span>
        </button>

        <div className="mx-auto flex w-full max-w-xl items-center gap-2 rounded-2xl border border-line bg-surface px-3.5 transition-colors focus-within:border-accent/70 hover:border-line-strong">
          <SearchIcon className="h-[18px] w-[18px] shrink-0 text-faint" />
          <input
            ref={box}
            type="text"
            value={search}
            placeholder="Search your photos by what is in them"
            aria-label="Search by label"
            className="h-10 w-full bg-transparent text-[15px] placeholder:text-faint focus:outline-none"
            onChange={(event) => setSearch(event.target.value)}
            onKeyDown={(event) => event.key === "Escape" && setSearch("")}
          />
          {search && (
            <button
              type="button"
              aria-label="Clear the search"
              className="rounded-full p-1 text-muted transition-colors hover:bg-hover hover:text-fg"
              onClick={() => {
                setSearch("");
                box.current?.focus();
              }}
            >
              <CloseIcon className="h-4 w-4" />
            </button>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-2">
          {offline && (
            <span className="hidden items-center gap-1.5 rounded-full border border-warn/30 bg-warn/10 px-2.5 py-1 text-xs text-warn md:flex">
              <span className="h-1.5 w-1.5 rounded-full bg-warn" />
              Reconnecting
            </span>
          )}

          <button
            type="button"
            aria-pressed={dev}
            title="Dev mode: watch the durable function work"
            className={`flex h-10 items-center gap-2 rounded-xl border px-3 font-mono text-[13px] transition ${
              dev ? "border-accent/50 bg-accent/10 text-accent" : "border-line text-muted hover:border-line-strong hover:text-fg"
            }`}
            onClick={() => setDev((on) => !on)}
          >
            <CodeIcon className="h-[18px] w-[18px]" />
            <span className="hidden md:inline">dev</span>
            <span className={`h-1.5 w-1.5 rounded-full ${dev ? "animate-breathe bg-accent" : "bg-faint"}`} />
          </button>

          <button
            type="button"
            className="flex h-10 items-center gap-2 rounded-xl bg-accent px-4 font-medium text-accent-ink transition hover:bg-accent-strong active:scale-[.97]"
            onClick={() => picker.current.click()}
          >
            <UploadIcon className="h-[18px] w-[18px]" />
            <span className="hidden sm:inline">Upload</span>
          </button>
          <input
            ref={picker}
            type="file"
            className="hidden"
            onChange={(event) => {
              if (event.target.files[0]) onUpload(event.target.files[0]);
              event.target.value = "";
            }}
          />

          <Account username={username} onSignOut={onSignOut} />
        </div>
      </div>
    </header>
  );
}
