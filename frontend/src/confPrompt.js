// Builds the prompt users paste into their AI agent tool so it can
// generate the JumpStart conf file.
export function buildConfPrompt(confPath) {
  return `You are helping me populate the config file for JumpStart, my local dev project manager.

Write a JSON file to this exact path (create parent folders if needed):
${confPath}

The file must contain a "projects" array. Schema per project:
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

After writing the file, tell me to open JumpStart's Dashboard and click "Import config".`;
}
