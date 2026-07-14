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
        { "title": "Auth flow", "done": true },
        { "title": "Billing page", "done": false }
      ]
    }
  ]
}

Rules:
- Omit "id" fields; they are generated on import.
- Use absolute paths only.
- Inspect my repositories to fill in accurate names, dirs, start commands and env vars.
- For tasks, list features that are already built (done: true) and features still to build (done: false).
- Reply with the JSON only, wrapped in a \`\`\`json code block, and nothing else.

After you print the JSON, I will paste it into JumpStart's "Import config" dialog.`;
}
