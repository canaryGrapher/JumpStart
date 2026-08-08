import RocketLogo from "./RocketLogo";
import { downloads } from "../downloads";
import { trackDownload, trackNav, trackOutbound } from "../analytics";
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
              onClick={() => trackDownload("macos", "footer")}
            >
              <AppleLogo /> macOS
            </a>
            <a
              href={downloads().windows}
              class="btn btn-dark"
              onClick={() => trackDownload("windows", "footer")}
            >
              <WindowsLogo /> Windows
            </a>
            <a
              href={downloads().linux}
              class="btn btn-dark"
              onClick={() => trackDownload("linux", "footer")}
            >
              <LinuxLogo /> Linux
            </a>
          </div>
          <p class="foot-dl-note">
            Free and open source.
            <a
              href="#/downloads"
              class="link-arrow"
              onClick={() => trackNav("downloads", "footer_note")}
            >
              Previous versions &amp; betas <span class="arrow">→</span>
            </a>
          </p>
        </div>
        <div class="foot-cols">
          <div>
            <h4>Product</h4>
            <a href="#features" onClick={() => trackNav("features", "footer")}>
              Features
            </a>
            <a href="#ship" onClick={() => trackNav("ship", "footer")}>
              Ship
            </a>
            <a href="#ai" onClick={() => trackNav("ai", "footer")}>
              AI Board
            </a>
            <a href="#faq" onClick={() => trackNav("faq", "footer")}>
              FAQ
            </a>
            <a href="#contribute" onClick={() => trackNav("contribute", "footer")}>
              Contribute
            </a>
          </div>
          <div>
            <h4>Resources</h4>
            <a href="#/downloads" onClick={() => trackNav("downloads", "footer")}>
              All downloads
            </a>
            <a href="#" onClick={() => trackNav("docs", "footer")}>
              Documentation
            </a>
            <a
              href="https://github.com/canaryGrapher/JumpStart/releases"
              onClick={() => trackOutbound("github_releases", "footer")}
            >
              Changelog
            </a>
            <a
              href="https://github.com/canaryGrapher/JumpStart"
              onClick={() => trackOutbound("github_source", "footer")}
            >
              Source
            </a>
          </div>
          <div>
            <h4>Legal</h4>
            <a href="#/privacy" onClick={() => trackNav("privacy", "footer")}>
              Privacy Policy
            </a>
            <a href="#/terms" onClick={() => trackNav("terms", "footer")}>
              Terms of Use
            </a>
          </div>
        </div>
      </div>
      <div class="container foot-bar">
        <span class="logo"><RocketLogo /> JumpStart<span class="logo-end">.</span></span>
        <span class="foot-copy">
          © 2026 JumpStart · Made with Wails · A{" "}
          <a href="https://workvar.com" onClick={() => trackOutbound("workvar", "footer_credit")}>
            workvar.com
          </a>{" "}
          project
        </span>
        <a
          href="https://www.foundrlist.com/product/jumpstart?utm_source=badge&utm_medium=embed"
          target="_blank"
          rel="noopener"
          class="foundr-badge"
          onClick={() => trackOutbound("foundrlist", "footer_badge")}
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
