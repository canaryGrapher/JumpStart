import { For } from "solid-js";

// Graphified process card from the real Processes tab.
export default function ProcessCard(props) {
  return (
    <div class="panel proc-card reveal tilt">
      <div class="proc-head">
        <strong>{props.name}</strong>
        <span class="status-chip" classList={{ run: props.running }}>
          {props.running ? "Running" : "Stopped"}
        </span>
      </div>
      <div class="cmd-box">{props.cmd}</div>
      <small class="proc-path">{props.path}</small>
      <div class="proc-actions">
        <span class="pill-btn">Deps</span>
        <span class="pill-btn">Logs</span>
        <span class="pill-btn dark">Start</span>
      </div>
      <div class="scripts-row">
        <small>SCRIPTS</small>
        <div class="chips">
          <For each={props.scripts}>{(s) => <span class="pill-btn">{s}</span>}</For>
        </div>
      </div>
    </div>
  );
}
