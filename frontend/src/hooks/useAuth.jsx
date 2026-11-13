import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import { localAuth } from "@/local/auth";

const STORAGE_KEY = "auth";
const SESSION_MS = 60 * 60 * 1000; // a Cognito ID token is valid for an hour

const signedOut = { authenticated: false };

export default function useAuth() {
  const [auth, setAuth] = useState(localAuth || signedOut);
  // Until the stored session has been looked at, nobody knows whether to show
  // the sign-in form. Showing nothing for a moment beats flashing the form.
  const [checked, setChecked] = useState(Boolean(localAuth));

  useEffect(() => {
    if (localAuth) return;

    try {
      const stored = JSON.parse(localStorage.getItem(STORAGE_KEY));
      if (stored?.exp > Date.now()) setAuth(stored);
      else localStorage.removeItem(STORAGE_KEY);
    } catch {
      localStorage.removeItem(STORAGE_KEY);
    }

    setChecked(true);
  }, []);

  /** Signs in and returns an error message, or nothing when it worked. */
  const signIn = useCallback(async (username, password) => {
    let response;
    try {
      response = await api("/auth/login", { method: "POST", body: { username, password } });
    } catch {
      return "The API cannot be reached. Is it running?";
    }

    const result = await response.json().catch(() => ({}));
    const token = result.AuthenticationResult?.IdToken;
    if (!token) return result.message || "That did not work. Check the username and password.";

    const session = { username, authenticated: true, idToken: token, exp: Date.now() + SESSION_MS };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    setAuth(session);

    return "";
  }, []);

  /**
   * Forgets the session, which brings the form back. It is what the avatar's
   * "Sign out" does, and what happens when the API no longer accepts the token.
   */
  const signOut = useCallback(() => {
    if (localAuth) return;

    localStorage.removeItem(STORAGE_KEY);
    setAuth(signedOut);
  }, []);

  // Locally nobody signed in, so there is nothing to sign out of.
  return { auth, checked, signIn, signOut, canSignOut: !localAuth };
}
