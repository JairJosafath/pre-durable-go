import { useState, useEffect } from "react";

// Local development only. When NEXT_PUBLIC_LOCAL_USER names a user, the app
// starts signed in as that user and never shows the sign-in form. The local
// API (`make local-api`) has no Cognito: it takes the token to be the user's
// name. Against AWS this variable must stay unset; a deployed API would
// reject such a token, so setting it by mistake cannot let anyone in.
const localUser = process.env.NEXT_PUBLIC_LOCAL_USER;
const localAuth = localUser
  ? { username: localUser, authenticated: true, idToken: localUser }
  : null;

export default function useAuth() {
  const [auth, setAuth] = useState(localAuth || { authenticated: false });

  useEffect(() => {
    if (localAuth) return;

    const auth = localStorage.getItem("auth");
    const exp = JSON.parse(auth)?.exp;
    if (exp < Date.now()) {
      setAuth(() => ({ authenticated: false }));
      localStorage.removeItem("auth");
    } else if (auth) {
      setAuth(() => JSON.parse(auth));
    }
  }, []);

  return { auth, setAuth };
}
