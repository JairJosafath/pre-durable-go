import { useState, useEffect, useRef } from "react";
import { api } from "@/lib/api";

export default function useUploader(token) {
  const input = useRef(null);
  const [reload, setReload] = useState(false);
  const [file, setFile] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!file || !token) return;

    async function uploadFile() {
      setLoading(true);

      // The browser uploads straight to S3. The object landing in the bucket
      // is what starts the durable function.
      const response = await api(`/presigned?action=put&key=${encodeURIComponent(file.name)}`, { token });
      const { url } = await response.json();

      const upload = await fetch(url, {
        method: "PUT",
        body: file,
        headers: { "Content-Type": file.type },
      });

      if (upload.ok) setReload(true);

      setLoading(false);
      setFile(null);
    }

    uploadFile();
  }, [file, token]);

  return { file, setFile, loading, reload, setReload, input };
}
