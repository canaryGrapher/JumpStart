import RocketLogo from "./RocketLogo";
import WinSidebar from "./hero/WinSidebar";
import WinStats from "./hero/WinStats";
import WinPanels from "./hero/WinPanels";
import { DOWNLOADS } from "../downloads";
import { track } from "../analytics";
import { AppleLogo, WindowsLogo, LinuxLogo } from "./OSIcon";

export default function Hero() {
  return (
    <>
      {/* Black opening screen at the root stacking context so it covers the
          navbar and the 3D window layers; scrolling scales it away. */}
      <div class="intro" id="intro" aria-hidden="true">
        <h2 class="intro-text">
          developers<span class="dot">.</span> ready<span class="dot">.</span>
        </h2>
        <span class="intro-hint">Scroll ↓</span>
      </div>

    <header class="hero" id="top">
      <div class="hero-bg" aria-hidden="true"></div>

      {/* The animation is the opening hook: the real dashboard, graphified
          into layers, starts exploded and fills the first viewport. GSAP
          composes it as you scroll; the text follows below (animations.js). */}
      <div class="stage" id="stage">
        <div class="win-fit">
        <div class="win" id="win">
          <div class="win-layer win-chrome" data-layer="chrome">
            <span class="tl r"></span><span class="tl y"></span><span class="tl g"></span>
            <span class="win-title"><RocketLogo size={20} /> Dashboard</span>
          </div>
          <WinSidebar />
          <WinStats />
          <WinPanels />
          <div class="win-layer win-port" data-layer="port">
            <div class="port-card">
              <h4>Port usage &amp; mapping</h4>
              <p>Live view of every port used by managed subprocesses</p>
              <div class="port-empty">Start a subprocess and its listening ports will show up here.</div>
            </div>
          </div>
        </div>
        </div>
        <span class="hero-hint">Scroll to assemble ↓</span>
      </div>

      <div class="hero-content">
        <span class="eyebrow-dot">● Native Control Panel · macOS &amp; Windows 🚀</span>
        <h1 class="hero-title">Run. Test. Ship.<br />One Window.</h1>
        <p class="hero-sub">
          JumpStart starts, stops, and monitors every project on your machine, then goes further:
          run your tests, wrangle Docker, commit and push, and publish a tagged release without
          touching a terminal. It even keeps itself up to date.
        </p>
        <div class="hero-dl">
          <a
            href={DOWNLOADS.macos}
            class="btn btn-dark btn-lg"
            onClick={() => track("download", { platform: "macos", location: "hero" })}
          >
            <AppleLogo /> Download for macOS <span class="arrow">→</span>
          </a>
          <a
            href={DOWNLOADS.windows}
            class="btn btn-outline btn-lg"
            onClick={() => track("download", { platform: "windows", location: "hero" })}
          >
            <WindowsLogo /> Download for Windows
          </a>
          <a
            href={DOWNLOADS.linux}
            class="btn btn-outline btn-lg"
            onClick={() => track("download", { platform: "linux", location: "hero" })}
          >
            <LinuxLogo /> Download for Linux
          </a>
        </div>
      </div>
    </header>
    </>
  );
}
