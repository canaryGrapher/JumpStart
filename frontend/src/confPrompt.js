import { TASK_FIELD_DOCS } from "./taskFields.js";

// Builds the prompt users paste into their AI agent tool so it can
// generate the JumpStart config JSON. The agent prints the JSON as text
// (it does NOT write a file); the user pastes that text into JumpStart's
// Import config dialog.
export function buildConfPrompt() {
  const taskFields = TASK_FIELD_DOCS.map(
    (f) => `          "${f.key}": …, // ${f.note}`
  ).join("\n");

  return `You are helping me build the import config for JumpStart, my local dev project manager.

Do NOT write any file. Instead, output ONLY a single JSON code block that I can copy and paste into JumpStart's "Import config" dialog.

The JSON must contain a "projects" array. Schema per project:
{
  "projects": [
    {
      "name": "Project Alpha",            // required, unique; used to update an existing project
      "root": "/absolute/path/to/repo",   // project root folder
      "tasksEnabled": true,                // enables the per-project task tracker
      "processes": [
        {
          "name": "frontend (Next.js)",   // display name
          "dir": "/absolute/path",        // working directory
          "command": "npm run dev",       // start command run with sh -c
          "env": { "PORT": "3000" },      // extra env vars (optional)
          "testCommand": "npm test",      // optional override for the Run tests button
          "scripts": [                     // optional one-off commands shown as buttons
            {
              "name": "Migrate",           // required, button label
              "command": "npm run db:migrate", // required, runs once and exits
              "dir": "/absolute/path",     // optional, defaults to the process dir
              "env": { "NODE_ENV": "development" }, // optional, merged over the process env
              "source": "package.json"     // optional, where the command came from
            }
          ]
        }
      ],
      "sprints": [                         // optional; tasks may reference by name via "sprint"
        { "name": "Sprint 1", "status": "active" }
      ],
      "tasks": [                           // optional feature tracker seed — same fields as CSV import
        {
${taskFields}
        },
        {
          "title": "Billing page",
          "type": "task",
          "status": "todo",
          "done": false,
          "priority": "medium",
          "description": "Subscription plans and invoice history.",
          "labels": ["billing"],
          "storyPoints": 3,
          "subtasks": [
            { "title": "Plan cards", "done": false },
            { "title": "Stripe webhook", "done": false }
          ],
          "acceptance": [
            { "title": "User can change plan", "done": false }
          ]
        }
      ]
    }
  ]
}

Rules:
- Omit "id" fields (including process, script, subtask and acceptance item ids); they are generated on import.
- Use absolute paths only.
- Inspect my repositories to fill in accurate names, dirs, start commands and env vars.
- For each process, add "scripts" for the one-off commands that repo actually has: package.json scripts other than the start command (build, lint, test, db:migrate, seed), Makefile targets, task/just recipes, Django or Rails management commands, and standalone shell scripts in the repo.
- A script must be a command that runs and exits. Long-lived watchers or servers belong in "processes", not "scripts".
- Give scripts short, human button labels ("Migrate", "Seed DB", "Lint") and set "source" to the file the command came from.
- Omit a script's "dir" and "env" unless they differ from the parent process.
- For tasks, use the same field set as JumpStart's task CSV import (header order): ${TASK_FIELD_DOCS.map((f) => f.key).join(", ")}.
- List features that are already built (done: true, status: "done") and features still to build (done: false, status matching their real progress: backlog, todo, or inprogress).
- Write a real "description" for each task/story: what it does and any relevant context, based on the code you find.
- Break non-trivial items into "subtasks" (implementation checklist) using {title, done}.
- Fill "acceptance" with concrete, testable acceptance criteria for any type (story, task, or bug) — same {title, done} shape as subtasks.
- Link child tasks to a parent story with "parentId" only when you also include that parent in the same tasks array with a stable temporary id you invent; otherwise omit parentId and let the user link later.
- Prefer "sprint" (name) over "sprintId" when seeding; unknown names create the sprint.
- Set "priority", "labels", "storyPoints", "assignee", "milestone", "issueType", "parentKey", "reviewers", and "linkedPrs" when you can reasonably infer them; omit fields you're not confident about rather than guessing.
- Reply with the JSON only, wrapped in a \`\`\`json code block, and nothing else.

After you print the JSON, I will paste it into JumpStart's "Import config" dialog.`;
}
