import LegalLayout from "./LegalLayout";

export default function Docs() {
  return (
    <LegalLayout title="Documentation" updated="October 5, 2026">
      <p>
        Guides for using JumpStart on your machine. For contributor architecture
        notes, see the{" "}
        <a
          href="https://github.com/canaryGrapher/JumpStart/wiki"
          target="_blank"
          rel="noreferrer"
        >
          GitHub Wiki
        </a>
        .
      </p>

      <h2>Guides</h2>
      <ul>
        <li>
          <a href="#/docs/mcp">Connect Agents (MCP)</a> — wire Cursor, Claude,
          Codex, ChatGPT, Hermes, Paperclip, and other harnesses to JumpStart.
        </li>
        <li>
          <a href="#/downloads">Downloads</a> — macOS, Windows, and Linux builds.
        </li>
        <li>
          <a href="#/privacy">Privacy Policy</a> — what the app and site collect.
        </li>
      </ul>

      <h2>On GitHub</h2>
      <ul>
        <li>
          <a
            href="https://github.com/canaryGrapher/JumpStart/wiki"
            target="_blank"
            rel="noreferrer"
          >
            Developer wiki
          </a>
        </li>
        <li>
          <a
            href="https://github.com/canaryGrapher/JumpStart/wiki/Agents-MCP"
            target="_blank"
            rel="noreferrer"
          >
            Agents MCP (full tool list + architecture)
          </a>
        </li>
        <li>
          <a
            href="https://github.com/canaryGrapher/JumpStart/releases"
            target="_blank"
            rel="noreferrer"
          >
            Releases / changelog
          </a>
        </li>
      </ul>
    </LegalLayout>
  );
}
