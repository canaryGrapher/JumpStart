import RocketLogo from "./RocketLogo";
import { DOWNLOADS, RELEASES_PAGE } from "../downloads";
import { track } from "../analytics";

export default function Footer() {
  return (
    <footer id="download">
      <div class="container foot-grid">
        <div class="foot-cta reveal">
          <h2>Are You Interested<br />In JumpStart?</h2>
          <div class="foot-dl">
            <a
              href={DOWNLOADS.macos}
              class="btn btn-dark"
              onClick={() => track("download", { platform: "macos", location: "footer" })}
            >
              Download for macOS <span class="arrow">→</span>
            </a>
            <a
              href={DOWNLOADS.windows}
              class="btn btn-outline"
              onClick={() => track("download", { platform: "windows", location: "footer" })}
            >
              Download for Windows
            </a>
          </div>
          <p class="foot-dl-note">
            Also on Linux — see <a href={RELEASES_PAGE}>all releases</a>. Free and open source.
          </p>
        </div>
        <div class="foot-cols">
          <div>
            <h4>Product</h4>
            <a href="#features">Features</a>
            <a href="#ship">Ship</a>
            <a href="#ai">AI Board</a>
            <a href="#faq">FAQ</a>
          </div>
          <div>
            <h4>Resources</h4>
            <a href="#">Documentation</a>
            <a href="https://github.com/canaryGrapher/JumpStart/releases">Changelog</a>
            <a href="https://github.com/canaryGrapher/JumpStart">Source</a>
          </div>
          <div>
            <h4>Legal</h4>
            <a href="#">Privacy Policy</a>
            <a href="#">Terms of Service</a>
          </div>
        </div>
      </div>
      <div class="container foot-bar">
        <span class="logo"><RocketLogo /> JumpStart<span class="logo-end">.</span></span>
        <span class="foot-copy">
          © 2026 JumpStart · Made with Wails · A <a href="https://workvar.com">workvar.com</a> project
        </span>
      </div>
      <div class="watermark" aria-hidden="true">JUMPSTART</div>
    </footer>
  );
}
