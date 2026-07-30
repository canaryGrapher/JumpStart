# UX Copy: JumpStart — Project detail & Tasks board

Reviewed: the project header, the Tasks tab (sprint filter, columns, empty states), the AI dock, and the task detail modal.

---

## 1. AI dock (collapsed)

**Current**: `Ask about this project or generate user stories…`

**Recommended**: `Ask about this project, or plan a feature` ✅ applied

| Option | Copy | Tone | Best for |
|---|---|---|---|
| A | Ask about this project, or plan a feature | Neutral, capability-forward | Default. Covers both jobs (Q&A + story drafting) without listing features. |
| B | Ask anything about this codebase | Confident, short | If code context indexing is the headline feature. |
| C | Plan a feature, or ask about the code | Task-first | If most users arrive to generate stories, not to ask questions. |

**Rationale**: The old string named an internal artifact ("user stories") before the user knows what the dock does. "Plan a feature" is the user's goal; stories are the output. Also shorter, which matters now that the dock is a fixed-width pill.

**Model chip**: `no model` reads like an error with no fix. Use `Set up AI` as a clickable affordance instead.

---

## 2. Column empty states

**Current**: `Drop items here`

Problem: drag is the only implied action, but every column also has `+ Add item`. A first-time user with an empty board sees three identical grey boxes.

**Recommended**, per column:

| Column | Copy |
|---|---|
| Backlog (board empty) | `Nothing here yet. Add a story, or ask AI to draft a few.` |
| Backlog (board has items) | `Drag items here to park them` |
| To Do | `Drag from Backlog when it's ready to start` |
| In Progress | `Drag here when you pick it up` |
| Done | `Finished work lands here` |

**Rationale**: Empty-state pattern is *what this is + why it's empty + how to start*. Column-specific copy also teaches the workflow, which a generic "Drop items here" cannot.

**Search empty state**: `No matching tasks` → `No tasks match "{query}". Try a shorter search.`

---

## 3. Add affordance

**Current**: `+ Add item` (identical in all four columns)

**Recommended**: `+ Add story` in Backlog, `+ Add task` elsewhere — match the label to what gets created. The type picker already exists; the button should reflect its state rather than say "item".

**Placeholder**: `story title...` → `What needs to happen?` (Backlog) / `Task title` (elsewhere). Placeholders that restate the field label add nothing.

---

## 4. Toast after AI adds stories

**Current**: `Added ${stories.length} story${stories.length === 1 ? "" : "ies"} to the board.`

⚠️ This produces **"Added 3 storyies to the board."** Pluralization bug, not just a copy nit.

**Recommended**: `Added ${n} ${n === 1 ? "story" : "stories"} to Backlog.`

Naming the destination column is more useful than "the board" — the user has to go find them.

---

## 5. AI setup error

**Current**: `Pick an Ollama model in Preferences → AI first.`

**Recommended**: `No AI model selected. Choose one in Preferences → AI to start chatting.`

**Rationale**: Error pattern is *what happened + why + how to fix*. The current version opens with the fix and never states the problem, so it reads as a scolding. It also leaks the vendor name ("Ollama") into a message about the user's own config.

---

## 6. Destructive actions

`Delete` on the project header needs a confirmation that names the consequence:

- Title: `Delete {ProjectName}?`
- Body: `JumpStart will forget this project's processes, tasks, and sprints. Files on disk are not touched.`
- Buttons: `Delete project` / `Keep it`

**Rationale**: The single biggest fear with a tool that "manages projects on my PC" is that deleting the entry deletes the code. Say explicitly that it does not.

---

## 7. Consistency fixes

| Issue | Where | Fix |
|---|---|---|
| Mixed ellipsis: `…` vs `...` | `Search tasks by title or description…` vs `Name...`, `Add label...`, `Add criterion...` | Standardise on `…` everywhere |
| "item" vs "task" vs "story" used interchangeably | Columns, add buttons, toasts | Reserve **story** for parent, **task** for child, drop **item** entirely |
| `Start all` / `Stop all` | Project header | Fine. Keep verb-first pattern for any new bulk action |
| `+ Sprint` | Tasks toolbar | `+ New sprint` — the bare noun is ambiguous next to the `Sprint 1` filter pill |

---

## Localization notes

- "Backlog", "sprint", and "story points" are agile jargon with established translations. Don't paraphrase them.
- Empty-state strings expand ~30% in German and French. The column empty states above are short enough to survive, but avoid going longer.
- Avoid "park them" and "lands here" if you localize — they're idioms. Literal fallbacks: "Store items here for later" / "Completed work appears here".
- `Delete {ProjectName}?` — keep the project name as a variable, not string-concatenated, so RTL layouts place it correctly.
