import { For } from "solid-js";

// Graphified from the real "Recent projects" and "Most used" panels.
const RECENT = [
  { n: "JumpStart", s: "2 subprocesses", d: "JumpStart is a macOS control panel that helps m…", m: "29m ago" },
  { n: "PeepalRP", s: "2 subprocesses", d: "PeepalRP is a comprehensive sales and marketing p…", m: "9h ago" },
  { n: "Defiance Capital Assessment", s: "3 subprocesses", d: "This project is a technical assessment for full stack…", m: "1d ago" },
  { n: "Consent Manager Platform", s: "5 subprocesses", d: "The Consent Manager Platform is a modular monoli…", m: "3d ago" },
];
const MOST_USED = [
  { n: "Defiance Capital Assessment", s: "3 subprocesses", d: "This project is a technical assessment for full stac…", m: "4 starts" },
  { n: "PeepalRP", s: "2 subprocesses", d: "PeepalRP is a comprehensive sales and marketing…", m: "2 starts" },
  { n: "Consent Manager Platform", s: "5 subprocesses", d: "The Consent Manager Platform is a modular monoli…", m: "1 starts" },
  { n: "JumpStart", s: "2 subprocesses", d: "JumpStart is a macOS control panel that helps ma…", m: "1 starts" },
];

function Panel(props) {
  return (
    <div class="win-panel">
      <h4>{props.title}</h4>
      <small>{props.sub}</small>
      <For each={props.items}>
        {(p) => (
          <div class="wp-item">
            <span class="proj-av">{p.n[0]}</span>
            <div>
              <strong>{p.n}</strong>
              <small>{p.s}</small>
              <small>{p.d}</small>
            </div>
            <span class="wp-meta">{p.m}</span>
          </div>
        )}
      </For>
    </div>
  );
}

export default function WinPanels() {
  return (
    <div class="win-layer win-panels" data-layer="panels">
      <Panel title="Recent projects" sub="Pick up where you left off" items={RECENT} />
      <Panel title="Most used" sub="Your go-to projects" items={MOST_USED} />
    </div>
  );
}
