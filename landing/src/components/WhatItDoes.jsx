import IconFeat from "./IconFeat";

export default function WhatItDoes() {
  return (
    <section class="section">
      <div class="container">
        <div class="feature-card reveal">
          <div class="mini-activity tilt">
            <h4>Config import</h4>
            <p>Add projects with the block builder, by pasting JSON, or from a file.</p>
            <div class="cmd-box">/Users/you/.jumpstart/import.json</div>
            <div class="proc-actions" style="justify-content:flex-start">
              <span class="pill-btn dark">Import config</span>
            </div>
            <div class="scripts-row">
              <small>PORT USAGE &amp; MAPPING</small>
              <div class="port-empty">Start a subprocess and its listening ports will show up here.</div>
            </div>
          </div>
          <div class="feature-copy">
            <h2>What Can JumpStart<br />Do For You?</h2>
            <IconFeat icon="▶" color="orange" title="Total Process Control">
              Start, stop, and inspect every subprocess with live PID, logs, CPU, and memory in
              one centralized place.
            </IconFeat>
            <IconFeat icon="⚓" color="purple" title="Automatic Port Detection">
              Ports pulled from logs and lsof in real time, shown as clickable localhost badges.
              No more guessing.
            </IconFeat>
            <IconFeat icon="↻" color="teal" title="Always Up To Date">
              JumpStart checks for new versions in the background and lets you grab the latest
              build with one click. In-app announcements keep you posted on what shipped.
            </IconFeat>
            <IconFeat icon="⚙" color="orange" title="Env & dotenv">
              Set per-process environment variables and import them straight from a project's
              .env file when JumpStart spots one.
            </IconFeat>
            <IconFeat icon="⬇" color="purple" title="Dependency Management">
              Inspect dependencies and run installs for common package managers without leaving
              the app.
            </IconFeat>
            <a href="#download" class="btn btn-dark">Learn More <span class="arrow">→</span></a>
          </div>
        </div>
      </div>
    </section>
  );
}
