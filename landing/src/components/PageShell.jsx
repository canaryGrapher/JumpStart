import { onMount } from "solid-js";
import RocketLogo from "./RocketLogo";

// Shared chrome for standalone (non-home) pages: minimal top bar and a
// compact footer. Home nav anchors don't apply here, so the logo simply
// returns to the landing page.
export default function PageShell(props) {
  onMount(() => window.scrollTo(0, 0));
  return (
    <div class="legal">
      <header class="legal-bar">
        <a class="logo" href="#top" onClick={() => (window.location.hash = "")}>
          <RocketLogo /> JumpStart<span class="logo-end">.</span>
        </a>
        <a class="legal-back" href="#top" onClick={() => (window.location.hash = "")}>
          ← Back to site
        </a>
      </header>
      {props.children}
      <footer class="legal-foot">
        <span>
          © 2026 JumpStart · A <a href="https://workvar.com">workvar.com</a> project
        </span>
        <span class="legal-foot-links">
          <a href="#/docs">Docs</a>
          <a href="#/downloads">Downloads</a>
          <a href="#/privacy">Privacy</a>
          <a href="#/terms">Terms</a>
        </span>
      </footer>
    </div>
  );
}
