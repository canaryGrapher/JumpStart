import { For } from "solid-js";
import IconFeat from "./IconFeat";

// Real backlog cards from the app's kanban board.
const TODO = [
  "Real AI report generation (currently hardcoded placeholder stats)",
  "Passing rate computed from real task data",
  "Notifications system",
];

export default function AiBoard() {
  return (
    <section class="section" id="ai">
      <div class="container">
        <div class="split-head reveal">
          <div>
            <span class="eyebrow-dot">● Built For Builders</span>
            <h2>JumpStart Helps You<br />Ship Faster</h2>
          </div>
          <div class="split-right">
            <p>
              A per-project Kanban board with a local AI assistant that drafts stories, tasks,
              and acceptance criteria. Powered by Ollama, nothing leaves your machine.
            </p>
          </div>
        </div>

        <div class="build-grid">
          <div class="icon-list reveal">
            <IconFeat icon="◈" color="orange" title="Smart Auto-Detection">
              Node, Go, Python, Ruby, PHP, Java, Gradle, Rust, and Docker Compose, even in
              nested monorepos.
            </IconFeat>
            <IconFeat icon="▤" color="purple" title="Built-In Kanban Board">
              Stories, tasks, and bugs with labels, priorities, story points, subtasks, and
              progress tracking.
            </IconFeat>
            <IconFeat icon="✦" color="teal" title="Local AI Assistant">
              Fill with AI drafts descriptions and subtasks using your own Ollama model at
              localhost:11434. Nothing leaves your machine.
            </IconFeat>
            <IconFeat icon="⇥" color="orange" title="Import Builder">
              Add projects by hand, by JSON, or with an interactive block builder and AI chat
              that assembles the config for you.
            </IconFeat>
          </div>
          <div class="panel kanban-frag reveal tilt">
            <div class="kf-progress">
              <span>11/16 done · 0 stories</span>
              <div class="kf-bar"><i></i></div>
              <span>69%</span>
            </div>
            <div class="kf-cols">
              <div class="kf-col">
                <h5>To Do <em>5</em></h5>
                <For each={TODO}>{(t) => <div class="kf-card">{t}</div>}</For>
              </div>
              <div class="kf-col">
                <h5>In Progress <em>0</em></h5>
                <div class="kf-drop">Drop items here</div>
              </div>
            </div>
            <div class="kf-ai">
              <span>✨ Ask AI to generate user stories…</span>
              <span class="model">llama3.1:8b</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
