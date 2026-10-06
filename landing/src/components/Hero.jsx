import ExplodedShot from "./hero/ExplodedShot";
import { onMount } from "solid-js";
import { downloads } from "../downloads";
import { totalDownloads, loadTotalDownloads } from "../downloadCount";
import { trackDownload } from "../analytics";
import { AppleLogo, WindowsLogo, LinuxLogo } from "./OSIcon";

export default function Hero() {
  onMount(() => loadTotalDownloads());

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

      {/* Dashboard UI pieces start outside the viewport and assemble as you
          scroll (animations.js). Vector components stay sharp at any scale. */}
      <div class="stage" id="stage">
        <div class="win-fit">
          <ExplodedShot />
        </div>
        <span class="hero-hint">Scroll to assemble ↓</span>
      </div>

      <div class="hero-content">
        <span class="eyebrow-dot">● Native control panel · macOS, Windows &amp; Linux</span>
        <h1 class="hero-title">
          <span class="brand">JumpStart</span>
          <span class="hero-line">Run. Test. Ship.<br />One Window.</span>
        </h1>
        <p class="hero-sub">
          Starts, stops, and monitors every project on your machine — then goes further:
          run your tests, wrangle Docker, commit and push, and publish a tagged release without
          touching a terminal.
        </p>
        <div class="hero-dl">
          <a
            href={downloads().macos}
            class="btn btn-dark btn-lg"
            onClick={() => trackDownload("macos", "hero")}
          >
            <AppleLogo /> Download for macOS <span class="arrow">→</span>
          </a>
          <a
            href={downloads().windows}
            class="btn btn-dark btn-lg"
            onClick={() => trackDownload("windows", "hero")}
          >
            <WindowsLogo /> Download for Windows <span class="arrow">→</span>
          </a>
          <a
            href={downloads().linux}
            class="btn btn-dark btn-lg"
            onClick={() => trackDownload("linux", "hero")}
          >
            <LinuxLogo /> Download for Linux <span class="arrow">→</span>
          </a>
        </div>
        {totalDownloads() > 0 && (
          <p class="hero-downloads">
            <strong>{totalDownloads().toLocaleString()}</strong> downloads and counting
          </p>
        )}
      </div>
    </header>
    </>
  );
}
