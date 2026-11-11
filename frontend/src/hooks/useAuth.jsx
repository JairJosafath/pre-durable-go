import { useState, useEffect } from "react";


export default function useAuth(){
  
    const [auth,setAuth] = useState({ authenticated: false });
  
    useEffect(() => {
      const auth = localStorage.getItem("auth");
      const exp = JSON.parse(auth)?.exp;
      if (exp < Date.now()) {
        setAuth(() => ({ authenticated: false }));
        localStorage.removeItem("auth");
      }
      else if (auth) {
        setAuth(() => JSON.parse(auth));
      }
    },[])
  
    return {auth, setAuth};
  }