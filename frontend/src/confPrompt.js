// Builds the prompt users paste into their AI agent tool so it can
// generate the JumpStart config JSON. The agent prints the JSON as text
// (it does NOT write a file); the user pastes that text into JumpStart's
// Import config dialog.
export function buildConfPrompt() {
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
          "env": { "PORT": "3000" }       // extra env vars (optional)
        }
      ],
      "tasks": [                           // optional feature tracker seed
        {
          "title": "Auth flow",             // required
          "done": true,                     // kept for backward compat; mirror status === "done"
          "status": "done",                 // backlog | todo | inprogress | done
          "type": "story",                  // story | task | bug (default "task")
          "description": "Email/password login plus Google OAuth, session cookies, and password reset.",
          "priority": "high",               // low | medium | high
          "labels": ["backend", "auth"],
          "subtasks": [
            { "title": "Login form", "done": true },
            { "title": "Password reset email", "done": false }
          ],
          "acceptance": [                   // acceptance criteria; stories only
            { "title": "User can log in with email + password", "done": true },
            { "title": "User can reset a forgotten password", "done": false }
          ],
          "storyPoints": 5,
          "assignee": "Yash"
        },
        { "title": "Billing page", "done": false, "status": "todo" }
      ]
    }
  ]
}

Rules:
- Omit "id" fields (including subtask/acceptance item ids); they are generated on import.
- Use absolute paths only.
- Inspect my repositories to fill in accurate names, dirs, start commands and env vars.
- For tasks, list features that are already built (done: true, status: "done") and features still to build (done: false, status matching their real progress: backlog, todo, or inprogress).
- Write a real "description" for each task/story: what it does and any relevant context, based on the code you find.
- Break non-trivial tasks into "subtasks" (implementation checklist items) using the same {title, done} shape.
- For tasks with type "story", also fill "acceptance" with concrete, testable acceptance criteria.
- Set "priority", "labels", and "storyPoints" when you can reasonably infer them; omit fields you're not confident about rather than guessing.
- Reply with the JSON only, wrapped in a \`\`\`json code block, and nothing else.

After you print the JSON, I will paste it into JumpStart's "Import config" dialog.`;
}
