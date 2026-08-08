# AI and Chat

Everything AI-related talks to a **local Ollama** server. Nothing is sent to cloud LLM APIs.

## Packages

| Piece | Location |
|-------|----------|
| HTTP client | `internal/ai/ollama.go` |
| Bindings | `ai_api.go`, `ai_commit.go`, `chat_api.go`, `codectx_api.go` |
| Retrieval | `internal/codectx` |
| Persistence | `internal/chatstore` |
| UI settings | `frontend/src/ai.js` (localStorage) |

## Ollama client

- Default `http://localhost:11434`, override via argument or `OLLAMA_HOST`
- Long HTTP timeout (minutes) for local generation
- Used for: task enrich, freeform chat, commit messages, project descriptions

## Code context (`codectx`)

- Walks the project (skips heavy dirs), chunks source, builds a **BM25** index — no embedding model required
- Overview (tree, manifests, scripts) is cheap and often included in prompts
- Search returns top chunks pasted into the system prompt
- Progress events: `codectx:progress`, `codectx:done`
- Index is per-project and removed on `DeleteProject`

## Chat store

- Path: `~/.jumpstart/chats/<safeProjectID>.json`
- Kept out of `config.json` so chat traffic does not rewrite the project file
- Cap: 50 sessions per project (oldest dropped)
- Messages may store structured `stories` JSON and `sources` file paths for UI previews

## UI surfaces

1. **TaskDetailModal** — “Fill with AI” → `OllamaEnrichTask`
2. **ChatDock** — story assistant; expand to fullscreen; `SendChatMessage`
3. **CommitBox** — `OllamaGenerateCommitMessage` using `gitops` commit context
4. **ProjectModal** — optional AI project description

## Privacy constraints

Analytics may record `model_family`, `param_size`, lengths, latency, success — **never** prompts, replies, paths, or raw model tags. Use `TrackModelSelected` so Go parses the tag safely.

## Changing prompts / retrieval

- Prompt assembly: `internal/codectx/prompt.go` and enrich paths in `ai_api.go`
- Chunking / walk limits: `chunk.go`, `walk.go`
- Keep retrieval local and deterministic unless you intentionally add a new dependency (and document the download UX)
