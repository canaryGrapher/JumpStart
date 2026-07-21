// Tiny hash-based router. Works on any static host (Vercel, GitHub Pages,
// the Wails webview) with no server rewrites. Legal pages live at
// #/privacy and #/terms; everything else renders the home page.
import { createSignal } from "solid-js";

function parse() {
  const h = window.location.hash;
  if (h === "#/privacy") return "privacy";
  if (h === "#/terms") return "terms";
  return "home";
}

const [route, setRoute] = createSignal(parse());

window.addEventListener("hashchange", () => {
  setRoute(parse());
  if (parse() !== "home") window.scrollTo(0, 0);
});

export { route };
