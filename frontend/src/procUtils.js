// Shared helpers for working with a project's process list.

// isComposeProc identifies subprocesses that are really "docker compose up"
// commands. These are managed from the Containers tab instead of the
// Processes/Tests tabs, so they are hidden there.
export const isComposeProc = (proc) => {
  const cmd = (proc.command || "").toLowerCase();
  return cmd.includes("docker compose") || cmd.includes("docker-compose");
};
