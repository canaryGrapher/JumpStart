// Live dashboard UI (Solid) — each region is a real component with
// data-layer so GSAP can fly pieces in from past 100vw/100vh.
import { For, Show } from "solid-js";
import Icon, { ICONS } from "../Icon";
import ProjIcon from "./ProjIcon";
import {
  ALL_PROJECTS,
  FAVORITES,
  MOST_USED,
  PORTS,
  RECENT,
  STATS,
} from "./mockData";

/**
 * Flight poses for animations.js. Layout positions come from CSS —
 * only `id` + `from` are consumed by the scroll timeline.
 */
export const SHOT_PIECES = [
  { id: "chrome", from: { x: "-120vw", y: "-30vh", z: 40, rotate: -6, rotateY: 10 } },
  { id: "search", from: { x: "-125vw", y: "-8vh", z: 36, rotate: -4, rotateY: 8 } },
  { id: "nav", from: { x: "-130vw", y: "4vh", z: 32, rotate: -3, rotateY: 10 } },
  { id: "favorites", from: { x: "-140vw", y: "20vh", z: 28, rotate: 3, rotateY: 12 } },
  { id: "all", from: { x: "-135vw", y: "75vh", z: 24, rotate: 4, rotateY: 10 } },
  { id: "foot", from: { x: "-110vw", y: "120vh", z: 20, rotate: 5, rotateY: 6 } },
  { id: "title", from: { x: "20vw", y: "-120vh", z: 40, rotate: -2, rotateX: 8 } },
  { id: "stat1", from: { x: "-35vw", y: "-125vh", z: 50, rotate: -6, rotateX: 6 } },
  { id: "stat2", from: { x: "8vw", y: "-130vh", z: 52, rotate: 3, rotateX: 8 } },
  { id: "stat3", from: { x: "60vw", y: "-128vh", z: 50, rotate: -3, rotateX: 7 } },
  { id: "stat4", from: { x: "130vw", y: "-110vh", z: 48, rotate: 5, rotateX: 6 } },
  { id: "recent", from: { x: "-125vw", y: "50vh", z: 36, rotate: -5, rotateY: 8 } },
  { id: "used", from: { x: "130vw", y: "35vh", z: 36, rotate: 5, rotateY: -8 } },
  { id: "port", from: { x: "10vw", y: "130vh", z: 28, rotate: 2, rotateX: -8 } },
];

function Piece(props) {
  return (
    <div
      class={`win-piece win-piece-${props.id}`}
      data-layer={props.id}
      aria-label={props.label}
    >
      {props.children}
    </div>
  );
}

function StatTile(props) {
  const s = props.stat;
  return (
    <Piece id={s.id} label={s.label}>
      <div class="hd-tile">
        <div class="hd-tile-label">{s.label}</div>
        <div class="hd-tile-num">
          <span class="hd-tile-value">{s.value}</span>
          <span class={`hd-pill ${s.pillKind === "neutral" ? "neutral" : ""}`}>{s.pill}</span>
        </div>
        <Show when={s.sub}>
          <div class="hd-tile-sub">{s.sub}</div>
        </Show>
        <Show when={s.meter != null}>
          <div class="hd-meter">
            <i style={{ width: `${s.meter}%` }} />
          </div>
        </Show>
      </div>
    </Piece>
  );
}

function ProjectList(props) {
  return (
    <Piece id={props.id} label={props.title}>
      <div class="hd-panel">
        <div class="hd-panel-head">
          <h4>{props.title}</h4>
          <Show when={props.link}>
            <span class="hd-link">
              {props.link} <Icon d={ICONS.chevron} />
            </span>
          </Show>
        </div>
        <p class="hd-panel-sub">{props.sub}</p>
        <For each={props.items}>
          {(p) => (
            <div class="hd-row">
              <ProjIcon n={p.n} c={p.c} g={p.g} />
              <div class="hd-row-text">
                <strong>{p.n}</strong>
                <small>{p.s}</small>
                <small class="hd-desc">{p.d}</small>
              </div>
              <span class="hd-meta">{p.m}</span>
            </div>
          )}
        </For>
      </div>
    </Piece>
  );
}

export default function ExplodedShot() {
  return (
    <div class="win win-shot" id="win" aria-label="JumpStart dashboard">
      <div class="shot-plate" aria-hidden="true" />

      <div class="win-frame">
        <aside class="win-side">
          <Piece id="chrome" label="Window chrome">
            <div class="hd-chrome">
              <span class="tl red" />
              <span class="tl yellow" />
              <span class="tl green" />
            </div>
          </Piece>

          <Piece id="search" label="Search">
            <div class="hd-search">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="11" cy="11" r="7" />
                <path d="M20 20l-3.5-3.5" />
              </svg>
              <span>Search</span>
            </div>
          </Piece>

          <Piece id="nav" label="Navigation">
            <nav class="hd-nav">
              <span class="hd-nav-row active">
                <Icon d={ICONS.dashboard} />
                <span>Dashboard</span>
              </span>
              <span class="hd-nav-row">
                <Icon d={ICONS.ports} />
                <span>Ports</span>
              </span>
            </nav>
          </Piece>

          <Piece id="favorites" label="Favorites">
            <div class="hd-side-block">
              <div class="hd-side-section with-icon">
                <Icon d={ICONS.star} filled />
                <span>Favorites</span>
              </div>
              <For each={FAVORITES}>
                {(p) => (
                  <div class="hd-side-wrap">
                    <div class="hd-side-row" classList={{ active: p.sel }}>
                      <ProjIcon n={p.n} c={p.c} g={p.g} />
                      <span class="hd-side-text">
                        <span class="hd-side-name">{p.n}</span>
                        <span class="hd-side-sub">{p.s}</span>
                      </span>
                    </div>
                    <span class="hd-star" aria-hidden="true">
                      <Icon d={ICONS.star} filled />
                    </span>
                  </div>
                )}
              </For>
            </div>
          </Piece>

          <Piece id="all" label="All projects">
            <div class="hd-side-block">
              <div class="hd-side-section">All projects</div>
              <For each={ALL_PROJECTS}>
                {(p) => (
                  <div class="hd-side-row">
                    <ProjIcon n={p.n} c={p.c} g={p.g} />
                    <span class="hd-side-text">
                      <span class="hd-side-name">{p.n}</span>
                      <span class="hd-side-sub">{p.s}</span>
                    </span>
                  </div>
                )}
              </For>
            </div>
          </Piece>

          <Piece id="foot" label="Sidebar footer">
            <div class="hd-foot">
              <span class="hd-nav-row">
                <Icon d={ICONS.plus} />
                <span>Add Project</span>
              </span>
              <span class="hd-gear" aria-hidden="true">
                <Icon d={ICONS.gear} />
              </span>
            </div>
          </Piece>
        </aside>

        <main class="win-main">
          <Piece id="title" label="Dashboard title">
            <header class="hd-title">
              <div class="hd-title-lead">
                <span class="hd-title-icon">
                  <Icon d={ICONS.dashboard} />
                </span>
                <div>
                  <h3>Dashboard</h3>
                  <small>Development build · vdev</small>
                </div>
              </div>
            </header>
          </Piece>

          <div class="hd-tiles">
            <For each={STATS}>{(s) => <StatTile stat={s} />}</For>
          </div>

          <div class="hd-lists">
            <ProjectList
              id="recent"
              title="Recent projects"
              sub="Pick up where you left off"
              link="All projects"
              items={RECENT}
            />
            <ProjectList
              id="used"
              title="Most used"
              sub="Your go-to projects"
              items={MOST_USED}
            />
          </div>

          <Piece id="port" label="Port usage">
            <div class="hd-panel hd-ports">
              <h4>Port usage &amp; mapping</h4>
              <p class="hd-panel-sub">Live view of every port used by managed subprocesses</p>
              <table class="hd-table">
                <thead>
                  <tr>
                    <th>Port</th>
                    <th>Project</th>
                    <th>Subprocess</th>
                    <th>PID</th>
                  </tr>
                </thead>
                <tbody>
                  <For each={PORTS}>
                    {(r) => (
                      <tr>
                        <td class="hd-port">{r.port}</td>
                        <td>{r.project}</td>
                        <td>{r.sub}</td>
                        <td class="hd-pid">{r.pid}</td>
                      </tr>
                    )}
                  </For>
                </tbody>
              </table>
            </div>
          </Piece>
        </main>
      </div>
    </div>
  );
}
