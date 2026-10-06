# Agents MCP

JumpStart can run a local [Model Context Protocol](https://modelcontextprotocol.io) server so external AI agents can drive the same projects you manage in the app: start processes, edit files inside project roots, inspect git changes, and work with kanban tasks.

A shorter, user-facing copy of this guide also lives on the website at
[#/docs/mcp](https://jumpstart.workvar.com/#/docs/mcp).

## What agents can do

| Tool | Purpose |
|------|---------|
| `list_projects` / `get_project` | Discover JumpStart projects |
| `list_processes` / `start_process` / `stop_process` / `start_all_processes` / `stop_all_processes` | Control subprocesses |
| `get_process_status` / `get_process_logs` | Inspect live status and logs |
| `update_process` / `add_process` | Edit or add subprocess definitions |
| `list_tasks` / `get_task` / `upsert_task` / `delete_task` | Read and change kanban tasks (`backlog` \| `todo` \| `inprogress` \| `testing` \| `done`) |
| `list_directory` / `read_file` / `write_file` | Browse and edit files inside a project root |
| `git_status` / `git_diff` / `git_working_changes` | Inspect repository changes |

File tools are sandboxed to each project's `root`. Absolute paths and `..` escapes are rejected.

## Enable it

1. Open **Settings → Agents**.
2. Turn on **Enable MCP server**.
3. Copy the **Server URL** and **Bearer token** (or the whole client config block).

The server binds to `127.0.0.1` only (default port `8787`). Config lives in `~/.jumpstart/mcp.json` with mode `0600`. JumpStart must be running with Agents MCP enabled for clients to connect.

Replace `<token>` below with the token from Settings → Agents.

---

## Connect Cursor

Cursor accepts streamable HTTP MCP directly:

```json
{
  "mcpServers": {
    "jumpstart": {
      "url": "http://127.0.0.1:8787/mcp",
      "headers": {
        "Authorization": "Bearer <token>"
      }
    }
  }
}
```

Reload MCP servers after toggling JumpStart.

---

## Connect Claude Code

Claude Code prefers an explicit `type: "http"` entry. Project scope (`.mcp.json` in the repo) is usually best:

```bash
claude mcp add --transport http --scope project jumpstart \
  http://127.0.0.1:8787/mcp \
  --header "Authorization: Bearer <token>"
```

Or write `.mcp.json` yourself:

```json
{
  "mcpServers": {
    "jumpstart": {
      "type": "http",
      "url": "http://127.0.0.1:8787/mcp",
      "headers": {
        "Authorization": "Bearer <token>"
      }
    }
  }
}
```

Verify with `claude mcp list`. A JSON entry with `url` but no `type` is treated as stdio and will fail.

---

## Connect Claude Desktop

**Do not paste JumpStart's `http://127.0.0.1:…/mcp` URL into Claude Desktop's
connector UI or as a `"type": "http"` / `"url"` entry.** That path only accepts
**public `https://` URLs** (reached from Anthropic's servers), so localhost HTTP
is rejected on purpose — even with a self-signed certificate.

For a local JumpStart server, use **stdio + [`mcp-remote`](https://www.npmjs.com/package/mcp-remote)**
in Developer settings (`claude_desktop_config.json`). Claude launches the bridge
on your machine; the bridge talks HTTP to JumpStart.

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`  
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

Requires Node.js/`npx` on `PATH`. JumpStart must already be running with Agents MCP enabled.

```json
{
  "mcpServers": {
    "jumpstart": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote@latest",
        "http://127.0.0.1:8787/mcp",
        "--allow-http",
        "--header",
        "Authorization:${AUTH_HEADER}"
      ],
      "env": {
        "AUTH_HEADER": "Bearer <token>"
      }
    }
  }
}
```

Put the Bearer value in `env` (not in `args`) so spaces are not mangled on
Windows. Fully quit and reopen Claude Desktop after saving. You should see a
hammer / tools icon once the server connects.

If you need the connector UI instead, expose JumpStart behind a **public HTTPS**
URL (e.g. a tunnel) and add that — not `http://localhost`.

---

## Connect Codex / ChatGPT desktop

[Codex](https://developers.openai.com/codex/config-reference) (CLI, IDE extension) and the ChatGPT desktop app share `~/.codex/config.toml` (or a project `.codex/config.toml`):

```toml
[mcp_servers.jumpstart]
url = "http://127.0.0.1:8787/mcp"
http_headers = { Authorization = "Bearer <token>" }
```

Safer: keep the token out of the file:

```toml
[mcp_servers.jumpstart]
url = "http://127.0.0.1:8787/mcp"
bearer_token_env_var = "JUMPSTART_MCP_TOKEN"
```

```sh
export JUMPSTART_MCP_TOKEN="<token>"
```

---

## Connect Hermes Agent

In `~/.hermes/config.yaml`:

```yaml
mcp_servers:
  jumpstart:
    transport: http
    url: http://127.0.0.1:8787/mcp
    headers:
      Authorization: "Bearer ${JUMPSTART_MCP_TOKEN}"
```

Export `JUMPSTART_MCP_TOKEN` in the environment Hermes runs under. Confirm with `hermes mcp list`.

If you use Hermes's optional Codex app-server runtime, migrate or mirror this server into `~/.codex/config.toml` as well so the Codex subprocess sees the same tools.

---

## Connect Paperclip

Paperclip does not host MCP itself — the agent's **runtime** does.

### Claude Code adapter

On the host / `cwd` the agent uses:

```bash
claude mcp add --transport http --scope project jumpstart \
  http://127.0.0.1:8787/mcp \
  --header "Authorization: Bearer <token>"
```

Prefer project `.mcp.json` so every agent pointed at that workspace gets JumpStart.

### Hermes adapter

1. Include `mcp` in the adapter `toolsets` (e.g. `"toolsets": "terminal,file,mcp"`).
2. Add the Hermes YAML block above on the heartbeat host.
3. Put `JUMPSTART_MCP_TOKEN` in the agent's Paperclip `env` / secrets — never commit the raw token.

Other Paperclip adapters (Codex, Cursor Local, Gemini CLI, OpenCode, Pi) inherit whatever MCP their CLI already supports; use the matching section on this page.

---

## Other harnesses

JumpStart speaks **streamable HTTP MCP** on `http://127.0.0.1:<port>/mcp` with an `Authorization: Bearer …` header.

| If the client supports… | Do this |
|-------------------------|---------|
| HTTP / streamable HTTP + headers (Cursor, Claude Code, Codex) | Use `url` + `headers` / `http_headers` |
| Claude Desktop connectors UI | Public `https://` only — use stdio + `mcp-remote` for localhost instead |
| Stdio only | Bridge with `npx -y mcp-remote@latest http://127.0.0.1:8787/mcp --allow-http --header Authorization:${AUTH_HEADER}` |
| Query-string tokens | `http://127.0.0.1:8787/mcp?token=<token>` (fallback; prefer the header) |

After connecting, ask the agent to `list_projects` or “list JumpStart MCP tools” to confirm the session is live.

---

## Security

- Disabled by default.
- Localhost-only listener.
- Every request requires the bearer token (or `?token=`).
- Rotate the token anytime from Settings → Agents.
- Agents with the token can start processes and write files inside registered project roots — treat the token like a local admin key.

## Architecture

- Package: `internal/mcpserver`
- Bindings: `mcp_api.go` (`GetMCPSettings`, `SetMCPSettings`, `RotateMCPToken`)
- Transport: MCP streamable HTTP via `github.com/modelcontextprotocol/go-sdk`
- Lifecycle: started in `App.Startup` when enabled; stopped in `App.Shutdown`
- Website docs: `landing/src/pages/DocsMcp.jsx` (`#/docs/mcp`)
