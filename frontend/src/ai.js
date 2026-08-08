// AI helpers wrapping the Ollama Wails bindings. Model + host live in
// localStorage and are edited from Preferences → AI.
import {
  OllamaListModels,
  OllamaEnrichTask,
  OllamaChat,
  OllamaGenerateCommitMessage,
} from "./api";

const HOST_KEY = "ollamaHost";
const MODEL_KEY = "ollamaModel";

export const DEFAULT_HOST = "http://localhost:11434";

export const getAISettings = () => ({
  host: localStorage.getItem(HOST_KEY) || DEFAULT_HOST,
  model: localStorage.getItem(MODEL_KEY) || "",
});

export const setAISettings = ({ host, model }) => {
  if (host !== undefined) localStorage.setItem(HOST_KEY, host);
  if (model !== undefined) localStorage.setItem(MODEL_KEY, model);
};

export const aiConfigured = () => !!getAISettings().model;

export const listModels = (host) => OllamaListModels(host || getAISettings().host);

// Ask the model to flesh out one item from what the user already wrote.
// body is the description typed so far, if any; the backend treats it as
// the stronger signal and expands on it rather than replacing it.
// Returns { description, acceptance[], subtasks[], priority, labels[] }.
export const enrichTask = (title, body, kind, projectId = "") => {
  const { host, model } = getAISettings();
  return OllamaEnrichTask(host, model, title, body || "", kind, projectId);
};

// One-shot chat with no persistence. The chat dock uses sendMessage in
// chats.js instead, which saves the thread; this stays for ad-hoc calls.
// Returns { reply, stories[], sources[] }.
export const chat = (history, projectId = "") => {
  const { host, model } = getAISettings();
  return OllamaChat(host, model, history, projectId);
};

// Draft a Conventional Commits message from a project's pending changes.
// The backend picks the diff: staged changes if anything is staged,
// otherwise everything uncommitted.
// Returns { subject, body, message, staged, fileCount, truncated }.
export const generateCommitMessage = (projectRoot) => {
  const { host, model } = getAISettings();
  return OllamaGenerateCommitMessage(host, model, projectRoot);
};
