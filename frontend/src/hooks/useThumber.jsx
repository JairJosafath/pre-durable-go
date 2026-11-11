import { useState, useEffect } from "react";
const baseUrl = process.env.NEXT_PUBLIC_ENDPOINT||"";

export default function useThumber(token, search) {
    const [thumbs, setThumbs] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState();
    const [keys, setKeys] = useState([]);
    const [photo, setPhoto] = useState({ title: "", url: "" });
  const [reload, setReload] = useState(false);

  
    async function getKeys() {
        let url = ``;
        search ? url = `${baseUrl}query?index=${search}` : url = `${baseUrl}files`;
      const res = await fetch(url, {
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      const data = await res.json();
      const tmpKeys = [];
  
      data.Items.forEach((item) => {
        search?
        item?.keys.L.forEach((key) => {
          tmpKeys.push(key.M.key.S)
        })
        :tmpKeys.push(item.key.S);
      });
  
      setKeys([...tmpKeys]);
      console.log({keys})
    setReload(false);

    }
  
    useEffect(() => {
      async function fetchThumbs() {
  
        const thumbsTmp = [];
        keys.forEach(async (key) => {
          const t = key?.split("/").splice(0, 2).join("/");
          const title = key?.replace(t + "/", "");
          const res = await fetch(`${baseUrl}/object/${title}`, {
            method: "GET",
            headers: {
              Authorization: `Bearer ${token}`
            },
          });
  
          
          const blob = await res.blob();
          const url = URL.createObjectURL(blob);
          thumbsTmp.push({ title, url });
          setThumbs([...thumbsTmp]);
          console.log({thumbs})
          
        });
  
        if (keys.length === 0)
          setThumbs([])
  
        setLoading(false);
      }
      fetchThumbs();
    }, [keys.length]);
  
    useEffect(() => {
      getKeys();
    }, [token, search]);
    useEffect(()=>{
        if(reload)
        getKeys();
      }, [reload, token])
  
    useEffect(()=>{
      if(!photo?.title) return
      async function getPresignedUrl(){
        const res = await fetch(`${baseUrl}/presigned?action=get&key=${photo.title}`, {
          method: "GET",
          headers: {
            Authorization: `Bearer ${token}`
          }
        });
        const data = await res.json();
        const url = JSON.parse(data.body).url;
        setPhoto({...photo, url})
      }
      getPresignedUrl();
    },[photo?.title,token])
  
    return { thumbs, loading, error, photo, setPhoto, setReload};
  }
  