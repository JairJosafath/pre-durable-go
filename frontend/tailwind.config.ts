import type { Config } from "tailwindcss";

// One dark theme. The names say what a colour is for, not what it looks like.
const config: Config = {
  content: ["./src/components/**/*.{js,ts,jsx,tsx}", "./src/app/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        canvas: "#090A0C",
        surface: "#101216",
        raised: "#16191F",
        hover: "#1C2028",
        line: "#22262E",
        "line-strong": "#313743",
        fg: "#E9EBEE",
        muted: "#8D95A3",
        faint: "#596170",
        accent: { DEFAULT: "#5DC9E2", strong: "#8ADBEE", ink: "#04252E" },
        ok: "#3DD68C",
        warn: "#F5B54A",
        bad: "#F26A6E",
      },
      fontFamily: {
        sans: ["var(--font-sans)", "system-ui", "sans-serif"],
        mono: ["var(--font-mono)", "ui-monospace", "monospace"],
      },
      keyframes: {
        shimmer: { "0%": { backgroundPosition: "200% 0" }, "100%": { backgroundPosition: "-200% 0" } },
        rise: { "0%": { opacity: "0", transform: "translateY(8px)" }, "100%": { opacity: "1", transform: "none" } },
        fade: { "0%": { opacity: "0" }, "100%": { opacity: "1" } },
        pop: { "0%": { opacity: "0", transform: "scale(.97)" }, "100%": { opacity: "1", transform: "none" } },
        breathe: { "0%, 100%": { opacity: "1" }, "50%": { opacity: ".35" } },
        slide: { "0%": { opacity: "0", transform: "translateX(24px)" }, "100%": { opacity: "1", transform: "none" } },
      },
      animation: {
        shimmer: "shimmer 1.8s linear infinite",
        rise: "rise .35s cubic-bezier(.2,.7,.2,1) both",
        fade: "fade .25s ease-out both",
        pop: "pop .2s cubic-bezier(.2,.7,.2,1) both",
        breathe: "breathe 1.6s ease-in-out infinite",
        slide: "slide .3s cubic-bezier(.2,.7,.2,1) both",
      },
    },
  },
  plugins: [],
};

export default config;
