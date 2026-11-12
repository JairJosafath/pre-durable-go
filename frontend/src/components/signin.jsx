import { useState } from "react";
import { Logo, Spinner, AlertIcon } from "@/components/icons";

export default function SignIn({ signIn }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(event) {
    event.preventDefault();
    if (!username || !password || busy) return;

    setBusy(true);
    setError(await signIn(username.trim(), password));
    setBusy(false);
  }

  const field =
    "w-full rounded-xl border border-line bg-canvas px-3.5 py-2.5 text-fg placeholder:text-faint transition-colors hover:border-line-strong focus:border-accent focus:outline-none";

  return (
    <div className="flex min-h-screen items-center justify-center p-6">
      <form onSubmit={submit} className="w-full max-w-sm animate-rise rounded-3xl border border-line bg-surface/80 p-8 shadow-2xl shadow-black/40 backdrop-blur">
        <div className="mb-8 flex items-center gap-3">
          <Logo className="h-10 w-10" />
          <div>
            <h1 className="text-xl font-semibold tracking-tight">PrimeDrive</h1>
            <p className="text-sm text-muted">Photos that sort themselves.</p>
          </div>
        </div>

        <label className="mb-1.5 block text-sm text-muted" htmlFor="username">
          Username
        </label>
        <input
          id="username"
          className={field}
          autoComplete="username"
          autoFocus
          value={username}
          onChange={(event) => setUsername(event.target.value)}
        />

        <label className="mb-1.5 mt-4 block text-sm text-muted" htmlFor="password">
          Password
        </label>
        <input
          id="password"
          type="password"
          className={field}
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />

        {error && (
          <p role="alert" className="mt-4 flex animate-fade items-start gap-2 rounded-xl border border-bad/30 bg-bad/10 px-3 py-2 text-sm text-bad">
            <AlertIcon className="mt-0.5 h-4 w-4 shrink-0" />
            {error}
          </p>
        )}

        <button
          type="submit"
          disabled={!username || !password || busy}
          className="mt-6 flex w-full items-center justify-center gap-2 rounded-xl bg-accent px-4 py-2.5 font-medium text-accent-ink transition hover:bg-accent-strong active:scale-[.98] disabled:cursor-not-allowed disabled:opacity-40"
        >
          {busy && <Spinner />}
          {busy ? "Signing in" : "Sign in"}
        </button>
      </form>
    </div>
  );
}
