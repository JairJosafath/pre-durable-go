"use client";
import Auth from "@/components/auth";
import PhotosContainer from "@/components/photos.container";
import SearchContainer from "@/components/search.container";
import Topbar from "@/components/topbar";
import { useState, useRef, useEffect } from "react";
import useUploader from "@/hooks/useUploader";
import useAuth from "@/hooks/useAuth";

export default function Home() {
  const [search, setSearch] = useState("");
  const searchInput = useRef(null);

  const { auth, setAuth } = useAuth();
  const { file, setFile, loading, reload, setReload, input } =
    useUploader(auth?.idToken);

      
    useEffect(() => {
      if(!search&&searchInput?.current?.value)
      searchInput.current.value = "";
    },[search])
  return (
    <main>
      {auth.authenticated ? (
        <>
          <Topbar
            searchInput={searchInput}
            input={input}
            setFile={setFile}
            auth={auth}
            setSearch={setSearch}
          />
          {!search ? (
            <PhotosContainer
              token={auth?.idToken}
              reload={reload}
              setReload={setReload}
            />
          ) : (
            <SearchContainer
              token={auth?.idToken}
              search={search}
              setSearch={setSearch}
            />
          )}
        </>
      ) : (
        <Auth setAuth={setAuth} />
      )}
    </main>
  );
}
