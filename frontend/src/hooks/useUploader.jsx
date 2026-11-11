

import { useState, useEffect, useRef } from "react";
const baseUrl = process.env.NEXT_PUBLIC_ENDPOINT||"";

export default function useUploader(token){
  
  const input = useRef(null);  
    const [reload, setReload] = useState(false);
    const [file, setFile] = useState(null);
    const [error, setError] = useState(null);
    const [loading, setLoading] = useState(false);

  
    async function getPresignedUrl(key){
      if (!key) return;
      const response = await fetch(`${baseUrl}presigned?action=put&key=${key}`,
      {
        method: "GET",
        headers: {
          "Authorization": `Bearer ${token}`
        }
      })
      const result = await response.json();
      const url = JSON.parse(result.body).url;
      console.log({url})
      return url;
    }
      
  
    async function uploadFile(){
      if (!file) return;
      const title = file.name;
      const presignedUrl = await getPresignedUrl(title);
      if (!presignedUrl) return;
      const response = await fetch(presignedUrl, {
        method: "PUT",
        body: file,
        headers:{
          "Content-Type": file.type
        }
      });
  
      if (response.status === 200) {
        console.log("success");
        setReload(true);
      } else {
        console.log("error");
      }
  
      setLoading(false);
      setFile(null);
  
    }
  
    useEffect(() => {
      uploadFile();
    }, [file, token]);
  
  
    return {
      file, setFile, error, loading, reload, setReload, input, 
    }
  }