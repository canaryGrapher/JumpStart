import { onMount } from "solid-js";
import RocketLogo from "../components/RocketLogo";

// Shared shell for legal pages: minimal top bar, centered prose column,
// and a compact footer. Home nav anchors don't apply here, so the logo
// simply returns to the landing page.
export default function LegalLayout(props) {
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
      <main class="legal-doc container narrow">
        <h1>{props.title}</h1>
        <p class="legal-updated">Last updated: {props.updated}</p>
        {props.children}
      </main>
      <footer class="legal-foot">
        <span>© 2026 JumpStart · A <a href="https://workvar.com">workvar.com</a> project</span>
        <span class="legal-foot-links">
          <a href="#/privacy">Privacy</a>
          <a href="#/terms">Terms</a>
        </span>
      </footer>
    </div>
  );
}
