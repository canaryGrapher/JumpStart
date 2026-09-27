import { useSyncExternalStore } from "react";
import { getWindows, subscribeTerminalDock } from "../terminalDock";

// Subscribes a component to the terminal dock's current windows (open,
// minimized, running, and finished). Actions (openTerminal, closeTerminal,
// etc.) are plain functions imported directly from ../terminalDock — they
// don't need a subscription, only the list does.
export default function useTerminalDock() {
  const windows = useSyncExternalStore(subscribeTerminalDock, getWindows);
  return { windows };
}
