import Icon, { ICONS } from "./Icon";
import IconFeat from "./IconFeat";

export default function Ship() {
  return (
    <section class="section" id="ship">
      <div class="container">
        <div class="split-head reveal">
          <div>
            <span class="eyebrow-dot">● From Commit To Release</span>
            <h2>Ship Without<br />Leaving The App</h2>
          </div>
          <div class="split-right">
            <p>
              The whole release path lives next to your processes: stage and commit, push to
              GitHub or GitLab with a securely stored token, run your test suite, and publish a
              tagged release. All from the same window your app runs in.
            </p>
          </div>
        </div>

        <div class="build-grid">
          <div class="icon-list reveal">
            <IconFeat icon="⎇" color="orange" title="Built-In Git Panel">
              Status, branch, ahead/behind, staged changes, commit, fetch, pull, and push.
              Tokens stay in your keychain, never in a config file.
            </IconFeat>
            <IconFeat icon="⏏" color="blue" title="One-Click Releases">
              Draft a tag, name it, write notes (or let AI draft them), and publish straight to
              GitHub or GitLab Releases.
            </IconFeat>
            <IconFeat icon="✓" color="green" title="Test Runner & Docker">
              Detects your test setup and runs it on demand. Compose up and down, and manage
              containers, images, and volumes in place.
            </IconFeat>
          </div>
          <div class="panel proj-head-card reveal tilt">
            <div class="ph-top">
              <div class="ph-identity">
                <span class="proj-av lg">J</span>
                <div>
                  <h3>JumpStart</h3>
                  <small class="ph-path">/Users/you/Projects/jumpstart</small>
                </div>
              </div>
              <div class="ph-actions">
                <span class="icon-btn outline primary" title="Start all">
                  <Icon d={ICONS.play} filled />
                </span>
                <span class="icon-btn outline" title="Stop all">
                  <Icon d={ICONS.stop} filled />
                </span>
                <span class="icon-btn outline" title="Edit">
                  <Icon d={ICONS.pencil} />
                </span>
                <span class="icon-btn outline danger" title="Delete">
                  <Icon d={ICONS.trash} />
                </span>
              </div>
            </div>
            <p class="ph-desc">
              JumpStart is a native control panel that helps manage applications and projects
              by detecting runnable parts and allowing one-click starting, stopping,
              inspecting, and organizing of processes.
            </p>
            <div class="ph-tabs">
              <span class="active">Processes</span>
              <span>Tasks</span>
              <span>Git</span>
              <span>Tests</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
