import React from "react";
import { api } from "@/lib/api";

export default function Auth({ setAuth }) {
  const username = React.useRef(null);
    const password = React.useRef(null);
  const [error, setError] = React.useState(null);
  async function signIn (){
    if (!username || !password) return;
    const user = username?.current?.value
    const pass = password?.current?.value

    const response = await api("/auth/login", { method: "POST", body: { username: user, password: pass } });

    const result = await response.json();
    const token = result.AuthenticationResult?.IdToken;

    if (token) {
    localStorage.setItem("auth", JSON.stringify({ username: user, authenticated: true, idToken: token, exp: Date.now() + 60 * 60 * 1000}));

      setAuth(() => {
        return { username: user, authenticated: true, idToken: token ,  exp: Date.now() + 60 * 60 * 1000};
      });
    } else {
      console.log("error");
    }
  };
  return (
    <div className="flex h-screen justify-center items-center">
      <div className="flex flex-col gap-2">
        <label>Username</label>
        <input ref={username} type="text" className="p-2 rounded text-black" />
        <label>Password</label>
        <input ref={password} type="password" className="p-2 rounded text-black" />
        <button className="bg-primary-500 text-white p-2 rounded" onClick={()=>signIn()}>
          Sign In
        </button>
      </div>
    </div>
  );
}