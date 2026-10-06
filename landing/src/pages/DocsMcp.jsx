import LegalLayout from "./LegalLayout";

// User-facing guide for connecting JumpStart's Agents MCP server to
// common AI harnesses. Keep in sync with docs/wiki/Agents-MCP.md.
export default function DocsMcp() {
  return (
    <LegalLayout title="Connect Agents (MCP)" updated="October 5, 2026">
      <p>
        JumpStart can run a local{" "}
        <a href="https://modelcontextprotocol.io" target="_blank" rel="noreferrer">
          Model Context Protocol
        </a>{" "}
        server so AI agents on your machine can list projects, start and edit
        processes, read and write files inside project roots, inspect git
        changes, and work with kanban tasks — including the Testing column.
      </p>

      <h2>1. Enable the server</h2>
      <ol>
        <li>Open JumpStart → <strong>Settings → Agents</strong>.</li>
        <li>Turn on <strong>Enable MCP server</strong>.</li>
        <li>Copy the server URL and bearer token (or the whole client config).</li>
      </ol>
      <p>
        The server listens on <code>127.0.0.1</code> only (default port{" "}
        <code>8787</code>). JumpStart must stay running while agents connect.
        Replace <code>&lt;token&gt;</code> in every snippet below with your
        token from Settings.
      </p>

      <h2>2. Cursor</h2>
      <pre>{`{
  "mcpServers": {
    "jumpstart": {
      "url": "http://127.0.0.1:8787/mcp",
      "headers": {
        "Authorization": "Bearer <token>"
      }
    }
  }
}`}</pre>

      <h2>3. Claude Code</h2>
      <pre>{`claude mcp add --transport http --scope project jumpstart \\
  http://127.0.0.1:8787/mcp \\
  --header "Authorization: Bearer <token>"`}</pre>
      <p>
        Or add a project <code>.mcp.json</code> with{" "}
        <code>"type": "http"</code>, the URL above, and the Authorization
        header. An entry with <code>url</code> but no <code>type</code> is
        treated as stdio and will fail.
      </p>

      <h2>4. Claude Desktop</h2>
      <p>
        <strong>Do not</strong> paste{" "}
        <code>http://127.0.0.1:…/mcp</code> into Claude Desktop's connector UI
        or as a <code>"url"</code> / <code>"type": "http"</code> JSON entry.
        That path only accepts public <code>https://</code> URLs (Anthropic's
        servers reach them), so localhost HTTP is rejected on purpose.
      </p>
      <p>
        For local JumpStart, use <strong>stdio + mcp-remote</strong> in{" "}
        <code>claude_desktop_config.json</code> (Developer settings). Requires
        Node.js / <code>npx</code>. Copy this block from Settings → Agents →
        Claude Desktop as well:
      </p>
      <pre>{`{
  "mcpServers": {
    "jumpstart": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote@latest",
        "http://127.0.0.1:8787/mcp",
        "--allow-http",
        "--header",
        "Authorization:\${AUTH_HEADER}"
      ],
      "env": {
        "AUTH_HEADER": "Bearer <token>"
      }
    }
  }
}`}</pre>
      <p>
        Paths: macOS{" "}
        <code>~/Library/Application Support/Claude/claude_desktop_config.json</code>
        · Windows <code>%APPDATA%\Claude\claude_desktop_config.json</code>. Keep
        the Bearer value in <code>env</code>. Fully quit and reopen Claude
        Desktop after saving.
      </p>

      <h2>5. Codex / ChatGPT desktop</h2>
      <p>
        Shared config file: <code>~/.codex/config.toml</code> (or project{" "}
        <code>.codex/config.toml</code>).
      </p>
      <pre>{`[mcp_servers.jumpstart]
url = "http://127.0.0.1:8787/mcp"
http_headers = { Authorization = "Bearer <token>" }`}</pre>
      <p>Or keep the token out of the file:</p>
      <pre>{`[mcp_servers.jumpstart]
url = "http://127.0.0.1:8787/mcp"
bearer_token_env_var = "JUMPSTART_MCP_TOKEN"`}</pre>

      <h2>6. Hermes Agent</h2>
      <p>
        In <code>~/.hermes/config.yaml</code>:
      </p>
      <pre>{`mcp_servers:
  jumpstart:
    transport: http
    url: http://127.0.0.1:8787/mcp
    headers:
      Authorization: "Bearer \${JUMPSTART_MCP_TOKEN}"`}</pre>
      <p>
        Export <code>JUMPSTART_MCP_TOKEN</code> for the Hermes process. Confirm
        with <code>hermes mcp list</code>.
      </p>

      <h2>7. Paperclip</h2>
      <p>
        Paperclip agents inherit MCP from their runtime. For a{" "}
        <strong>Claude Code</strong> adapter, run the Claude Code command in
        section 3 from the agent's <code>cwd</code>. For a{" "}
        <strong>Hermes</strong> adapter, include <code>mcp</code> in{" "}
        <code>toolsets</code> and add the Hermes YAML above; store the token as
        a Paperclip secret on the agent's env. Other adapters (Codex, Cursor
        Local, Gemini CLI, OpenCode, Pi) use the matching section for their CLI.
      </p>

      <h2>8. Other harnesses</h2>
      <p>
        JumpStart exposes <strong>streamable HTTP MCP</strong> at{" "}
        <code>http://127.0.0.1:&lt;port&gt;/mcp</code> with{" "}
        <code>Authorization: Bearer …</code>. If the client only speaks stdio,
        use <code>mcp-remote</code> as in the Claude Desktop section. As a
        last resort, <code>?token=&lt;token&gt;</code> on the URL is accepted.
      </p>

      <h2>Security</h2>
      <ul>
        <li>Off by default; localhost only.</li>
        <li>Rotate the token anytime in Settings → Agents.</li>
        <li>
          Anyone with the token can start processes and write files inside
          registered project roots — treat it like a local admin key.
        </li>
      </ul>

      <p class="legal-updated">
        Developer notes and tool list:{" "}
        <a
          href="https://github.com/canaryGrapher/JumpStart/wiki/Agents-MCP"
          target="_blank"
          rel="noreferrer"
        >
          GitHub Wiki → Agents MCP
        </a>
        .
      </p>
    </LegalLayout>
  );
}
