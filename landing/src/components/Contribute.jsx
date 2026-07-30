import { track } from "../analytics";

const REPO_URL = "https://github.com/canaryGrapher/JumpStart";
const CONTRIBUTING_URL = `${REPO_URL}/issues`;

export default function Contribute() {
  return (
    <section class="section contribute" id="contribute">
      <div class="container narrow center">
        <span class="eyebrow-dot">● Open Source</span>
        <h2 class="reveal">Built In The Open</h2>
        <p class="contribute-sub reveal">
          JumpStart is open source and shaped by the people who use it. Found a bug, want a
          feature, or fancy shipping one yourself? The repo is the place to start, and small
          fixes are every bit as welcome as big ones.
        </p>
        <div class="contribute-cta reveal">
          <a
            href={REPO_URL}
            class="btn btn-dark"
            target="_blank"
            rel="noopener noreferrer"
            onClick={() => track("outbound_github", { target: "repo", location: "contribute" })}
          >
            View on GitHub <span class="arrow">→</span>
          </a>
          <a
            href={CONTRIBUTING_URL}
            class="btn btn-outline"
            target="_blank"
            rel="noopener noreferrer"
            onClick={() =>
              track("outbound_github", { target: "issues", location: "contribute" })
            }
          >
            Contribute
          </a>
        </div>
      </div>
    </section>
  );
}
