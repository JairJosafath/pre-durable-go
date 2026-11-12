"use client";
import { useCallback, useEffect, useMemo, useState } from "react";
import SignIn from "@/components/signin";
import TopBar from "@/components/topbar";
import Photos, { Empty, PhotosLoading } from "@/components/photos";
import Reviews from "@/components/reviews";
import Lightbox from "@/components/lightbox";
import Inspector from "@/components/inspector";
import Toasts, { useToasts } from "@/components/toasts";
import useAuth from "@/hooks/useAuth";
import useDrive from "@/hooks/useDrive";
import useSearch from "@/hooks/useSearch";
import useThumbnails from "@/hooks/useThumbnails";
import useHistories from "@/hooks/useHistories";
import useStored from "@/hooks/useStored";
import { current } from "@/lib/timeline";
import { plural } from "@/lib/format";

// Dev mode: how often to read an execution's history, in milliseconds.
const WORKING_MS = 1500;
const WAITING_MS = 6000;
// Other uploads being processed are watched too, to name their step on the
// tile. Only a few: every look is an invocation of a Lambda function, and it
// should not compete with the pipeline for the account's concurrency.
const MAX_WATCHED = 2;

export default function Home() {
  const { auth, checked, signIn, signOut, canSignOut } = useAuth();

  if (!checked) return null;
  if (!auth.authenticated) return <SignIn signIn={signIn} />;

  return <Drive auth={auth} signOut={signOut} canSignOut={canSignOut} />;
}

function Drive({ auth, signOut, canSignOut }) {
  const token = auth.idToken;
  const { toasts, notify } = useToasts();
  // A token the API no longer accepts signs the user out.
  const drive = useDrive(token, { onExpired: signOut, notify });

  const [search, setSearch] = useState("");
  const term = search.trim();
  const found = useSearch(token, term);

  const [opened, setOpened] = useState(null);
  const [dev, setDev] = useStored("dev", false);
  // The upload the dev panel shows: the one the user picked, or else the one
  // that was last seen being processed.
  const [pinned, setPinned] = useState("");
  const [followed, setFollowed] = useState("");

  // A search result is an index entry: a key and a label. The drive knows
  // the rest about the files that are on screen.
  const results = useMemo(
    () => found.results.map((result) => drive.ready.find((file) => file.key === result.key) || { ...result, run: "", labels: [] }),
    [found.results, drive.ready],
  );

  // Thumbnails exist once the pipeline is past analysis.
  const pictured = useMemo(() => [...drive.ready, ...drive.reviews, ...results], [drive.ready, drive.reviews, results]);
  const thumbnail = useThumbnails(token, pictured);

  const live = drive.working[0]?.title || "";
  useEffect(() => {
    if (live) setFollowed(live);
  }, [live]);

  const selected = pinned || followed || drive.reviews[0]?.title || drive.all[0]?.title || "";

  const watched = useMemo(() => {
    if (!dev) return [];

    const every = (file) => (drive.working.includes(file) ? WORKING_MS : file.phase === "PENDING_REVIEW" ? WAITING_MS : 0);
    const chosen = drive.all.filter((file) => file.title === selected);
    const others = drive.working.filter((file) => file.title !== selected).slice(0, MAX_WATCHED);

    return [...chosen, ...others].map((file) => ({ title: file.title, every: every(file), stamp: `${file.run}|${file.phase}` }));
  }, [dev, drive.all, drive.working, selected]);
  const histories = useHistories(token, watched);

  const inspect = dev
    ? {
        selected,
        select: setPinned,
        // The operations an upload is in right now, to put on its tile.
        step: (file) => current(histories[file.title]?.history?.operations || []).map((operation) => operation.name).join(" · "),
      }
    : null;

  const openLabel = useCallback((label) => {
    setOpened(null);
    setSearch(label);
  }, []);
  const close = useCallback(() => setOpened(null), []);

  const nothingYet = drive.loaded && drive.all.every((file) => file.phase === "REJECTED");

  return (
    <>
      <TopBar
        username={auth.username}
        onSignOut={canSignOut ? signOut : undefined}
        search={search}
        setSearch={setSearch}
        onUpload={drive.upload}
        dev={dev}
        setDev={setDev}
        offline={drive.offline}
      />

      <main className={`px-5 pb-24 pt-8 transition-[padding] duration-300 sm:px-8 ${dev ? "pb-[62vh] lg:pb-24 lg:pr-[492px]" : ""}`}>
        {term ? (
          <section>
            <div className="mb-5 flex flex-wrap items-center gap-x-4 gap-y-3">
              <h1 className="text-2xl font-semibold tracking-tight">
                <span className="text-muted">Photos with</span> {term}
              </h1>
              <div className="flex flex-wrap gap-2">
                {found.labels
                  .filter((label) => label.toLowerCase() !== term.toLowerCase())
                  .slice(0, 8)
                  .map((label) => (
                    <button key={label} type="button" className="chip" onClick={() => setSearch(label)}>
                      {label}
                    </button>
                  ))}
              </div>
            </div>

            {found.searching && results.length === 0 ? (
              <PhotosLoading />
            ) : results.length === 0 ? (
              <Empty title={`Nothing labelled "${term}"`}>
                The search looks for labels that start with what you type. Try a shorter word, or pick one of the labels above.
              </Empty>
            ) : (
              <Photos ready={results} thumbnail={thumbnail} onOpen={setOpened} hasMore={found.hasMore} loadMore={found.loadMore} inspect={inspect} />
            )}
          </section>
        ) : (
          <>
            <Reviews pending={drive.reviews} thumbnail={thumbnail} decide={drive.decide} inspect={inspect} narrow={dev} />

            <section>
              <div className="mb-5 flex items-baseline gap-3">
                <h1 className="text-2xl font-semibold tracking-tight">Photos</h1>
                {drive.loaded && drive.ready.length > 0 && (
                  <span className="text-sm text-muted">
                    {plural(drive.ready.length, "photo")}
                    {drive.hasMore ? " and counting" : ""}
                  </span>
                )}
              </div>

              {!drive.loaded ? (
                <PhotosLoading />
              ) : nothingYet ? (
                <Empty title="Your drive is empty">
                  Upload a photo. It is labelled, checked and indexed on its own, and shows up here when it is ready.
                </Empty>
              ) : (
                <Photos
                  working={drive.working}
                  failed={drive.failed}
                  ready={drive.ready}
                  thumbnail={thumbnail}
                  onOpen={setOpened}
                  hasMore={drive.hasMore}
                  loadMore={drive.loadMore}
                  inspect={inspect}
                />
              )}
            </section>
          </>
        )}
      </main>

      {dev && (
        <Inspector
          files={drive.all}
          histories={histories}
          selected={selected}
          pinned={Boolean(pinned)}
          select={setPinned}
          follow={() => setPinned("")}
          onClose={() => setDev(false)}
        />
      )}

      {opened && <Lightbox token={token} file={opened} thumbnail={thumbnail(opened)} onClose={close} onLabel={openLabel} />}
      <Toasts toasts={toasts} />
    </>
  );
}
