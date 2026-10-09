// App-wide preferences persisted by the Go side (~/.jumpstart/settings.json)
// so the MCP server and Raycast see the same values as the UI. Components
// read them through useAppSettings(); writes go through saveAppSettings().
import { useEffect, useState } from "react";
import { GetAppSettings, SetAppSettings } from "./api";

export const THINK_LEVELS = [
  { id: "auto", label: "Auto", hint: "Model default" },
  { id: "off", label: "Off", hint: "Answer directly, fastest" },
  { id: "low", label: "Low", hint: "Brief reasoning" },
  { id: "medium", label: "Medium", hint: "Balanced" },
  { id: "high", label: "High", hint: "Most thorough, slowest" },
];

const DEFAULTS = {
  autosave: false,
  thinkLevel: "auto",
  hotkeyMode: "app",
  hotkeyKey: "cmd+shift+j",
  ocrEngine: "vision",
  dateOrder: "mdy",
};

let cache = { ...DEFAULTS };
let loaded = false;
const listeners = new Set();
const emit = () => listeners.forEach((fn) => fn(cache));

export async function loadAppSettings() {
  try {
    cache = { ...DEFAULTS, ...((await GetAppSettings()) || {}) };
    loaded = true;
    emit();
  } catch {
    // Keep defaults; the backend may be unavailable in a bare browser.
  }
  return cache;
}

export async function saveAppSettings(patch) {
  const next = await SetAppSettings({ ...cache, ...patch });
  cache = { ...DEFAULTS, ...next };
  emit();
  return cache;
}

export const getAppSettingsSnapshot = () => cache;

export default function useAppSettings() {
  const [value, setValue] = useState(cache);
  useEffect(() => {
    listeners.add(setValue);
    if (!loaded) loadAppSettings();
    else setValue(cache);
    return () => listeners.delete(setValue);
  }, []);
  return value;
}
