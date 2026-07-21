import RocketLogo from "./RocketLogo";
import { downloads, RELEASES_PAGE } from "../downloads";
import { track } from "../analytics";
import { AppleLogo, WindowsLogo, LinuxLogo } from "./OSIcon";

export default function Footer() {
  return (
    <footer id="download">
      <div class="container foot-grid">
        <div class="foot-cta reveal">
          <h2>Are You Interested<br />In JumpStart?</h2>
          <div class="foot-dl">
            <a
              href={downloads().macos}
              class="btn btn-dark"
              onClick={() => track("download", { platform: "macos", location: "footer" })}
            >
              <AppleLogo /> macOS
            </a>
            <a
              href={downloads().windows}
              class="btn btn-dark"
              onClick={() => track("download", { platform: "windows", location: "footer" })}
            >
              <WindowsLogo /> Windows
            </a>
            <a
              href={downloads().linux}
              class="btn btn-dark"
              onClick={() => track("download", { platform: "linux", location: "footer" })}
            >
              <LinuxLogo /> Linux
            </a>
          </div>
          <p class="foot-dl-note">
            Free and open source.
            <a
              href={RELEASES_PAGE}
              class="link-arrow"
              onClick={() => track("releases_redirect", { location: "footer" })}
            >
              Browse all releases <span class="arrow">→</span>
            </a>
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
            <a
              href="https://github.com/canaryGrapher/JumpStart/releases"
              onClick={() => track("releases_redirect", { location: "footer_changelog" })}
            >
              Changelog
            </a>
            <a
              href="https://github.com/canaryGrapher/JumpStart"
              onClick={() => track("outbound_github", { target: "source", location: "footer" })}
            >
              Source
            </a>
          </div>
          <div>
            <h4>Legal</h4>
            <a href="#/privacy">Privacy Policy</a>
            <a href="#/terms">Terms of Use</a>
          </div>
        </div>
      </div>
      <div class="container foot-bar">
        <span class="logo"><RocketLogo /> JumpStart<span class="logo-end">.</span></span>
        <span class="foot-copy">
          © 2026 JumpStart · Made with Wails · A <a href="https://workvar.com">workvar.com</a> project
        </span>
        <a
          href="https://www.foundrlist.com/product/jumpstart?utm_source=badge&utm_medium=embed"
          target="_blank"
          rel="noopener"
          class="foundr-badge"
        >
          <img
            src="https://www.foundrlist.com/api/badge/jumpstart"
            alt="Featured on FoundrList"
            width="150"
            height="48"
          />
        </a>
      </div>
      <div class="watermark" aria-hidden="true">JUMPSTART</div>
    </footer>
  );
}
