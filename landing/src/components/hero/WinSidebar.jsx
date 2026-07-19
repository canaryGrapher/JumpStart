import { For } from "solid-js";

// Graphified from the real JumpStart sidebar.
const PROJECTS = [
  { n: "AnyLM", s: "2 subprocesses", d: "AnyLM is a local-first workspace for large lan…" },
  { n: "Carousel Studio", s: "1 subprocess", d: "Carousel Studio is a local application that use…" },
  { n: "Consent Manager Platform", s: "5 subprocesses", d: "The Consent Manager Platform is a modular…" },
  { n: "CronCompose", s: "4 subprocesses", d: "CronCompose is a web tool that allows users…" },
  { n: "Defiance Capital Assessment", s: "3 subprocesses", d: "This project is a technical assessment for full…" },
  { n: "JumpStart", s: "2 subprocesses", d: "JumpStart is a macOS control panel that help…", sel: true },
  { n: "Patty", s: "3 subprocesses", d: "Patty is an AI-focused project management a…" },
];

export default function WinSidebar() {
  return (
    <aside class="win-layer win-side" data-layer="side">
      <div class="side-tiles">
        <span class="side-tile active"><i>▦</i>Dashboard</span>
        <span class="side-tile"><i>❯_</i>Ports</span>
      </div>
      <div class="side-search">Search projects…</div>
      <span class="side-label">PROJECTS</span>
      <For each={PROJECTS}>
        {(p) => (
          <div class="proj-row" classList={{ sel: p.sel }}>
            <span class="proj-av">{p.n[0]}</span>
            <div>
              <strong>{p.n}</strong>
              <small>{p.s}</small>
              <small>{p.d}</small>
            </div>
          </div>
        )}
      </For>
    </aside>
  );
}
