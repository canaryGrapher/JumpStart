import { For, Show } from "solid-js";
import Icon, { ICONS } from "./Icon";

// Graphified process card from the real Processes tab (macOS 27 redesign).
export default function ProcessCard(props) {
  const running = !!props.running;
  return (
    <div class="panel proc-card reveal tilt" classList={{ running }}>
      <div class="proc-head">
        <span class="proc-glyph" classList={{ on: running }} aria-hidden="true">
          <Icon d={ICONS.terminal} />
        </span>
        <div class="proc-title">
          <h3>{props.name}</h3>
          <span class="proc-state" classList={{ on: running }}>
            <span class="dot" />
            {running ? `Running · PID ${props.pid || 4821}` : "Stopped"}
          </span>
        </div>
        <span class="btn proc-run" classList={{ danger: running, primary: !running }}>
          <Icon d={running ? ICONS.stop : ICONS.play} filled />
          {running ? "Stop" : "Run"}
        </span>
      </div>

      <div class="cmd">
        <span class="cmd-prompt">$</span>
        {props.cmd}
      </div>
      <Show when={props.path}>
        <div class="dir">
          <Icon d={ICONS.folder} />
          <span>{props.path}</span>
        </div>
      </Show>

      <Show when={running}>
        <div class="proc-live">
          <div class="ports">
            <For each={props.ports || ["5173", "34115"]}>
              {(p) => (
                <span class="port-badge">
                  localhost:{p}
                  <Icon d={ICONS.chevron} />
                </span>
              )}
            </For>
          </div>
          <div class="usage-badges">
            <span class="usage-badge">CPU <b>{props.cpu || "2.4"}%</b></span>
            <span class="usage-badge">RAM <b>{props.ram || "186"} MB</b></span>
          </div>
        </div>
      </Show>

      <Show when={props.scripts?.length}>
        <div class="scripts-row">
          <small>Scripts</small>
          <div class="chips">
            <For each={props.scripts}>{(s) => <span class="pill-btn">{s}</span>}</For>
          </div>
        </div>
      </Show>

      <div class="proc-footer">
        <div class="card-actions-utility" />
        <div class="proc-footer-toggles">
          <span class="chip-toggle">
            <Icon d={ICONS.layers} />
            Dependencies
          </span>
          <span class="chip-toggle">
            <Icon d={ICONS.fileText} />
            Logs
          </span>
        </div>
      </div>
    </div>
  );
}
