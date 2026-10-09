// AI helpers wrapping the Ollama Wails bindings. Model + host live in
// localStorage and are edited from Preferences → AI.
import {
  OllamaListModels,
  OllamaEnrichTask,
  OllamaChat,
  OllamaGenerateCommitMessage,
  OllamaModelInfo,
  CancelAIRequest,
  EventsOn,
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
// draft is the task as currently edited; the backend shows the model its due
// date, links, attachments, checklists, and other fields, and reads text
// attachments as extra context.
// Returns { description, acceptance[], subtasks[], priority, labels[], storyPoints, dueDate }.
export const enrichTask = (title, body, kind, projectId = "", draft = {}, opts = {}) => {
  const { host, model } = getAISettings();
  return OllamaEnrichTask(host, model, title, body || "", kind, projectId, draft, {
    requestId: opts.requestId || "",
    think: opts.think || "",
  });
};

// --- request ids, live progress and Stop ---

export const newRequestId = () =>
  (crypto.randomUUID && crypto.randomUUID()) || `${Date.now()}-${Math.random().toString(16).slice(2)}`;

export const cancelAIRequest = (requestId) => requestId && CancelAIRequest(requestId);

// Subscribe to streamed reasoning/answer text for one request. Returns an
// unsubscribe function.
export const onAIDelta = (requestId, fn) => {
  const off = EventsOn("ai:delta", (d) => d && d.requestId === requestId && fn(d));
  return typeof off === "function" ? off : () => {};
};

// What the selected model supports: { thinking, thinkLevels, vision }.
export const modelInfo = (model) => {
  const { host } = getAISettings();
  return OllamaModelInfo(host, model || getAISettings().model);
};

// One-shot chat with no persistence. The chat dock uses sendMessage in
// chats.js instead, which saves the thread; this stays for ad-hoc calls.
// Returns { reply, stories[], sources[] }; stories may carry a dueDate.
export const chat = (history, projectId = "", opts = {}) => {
  const { host, model } = getAISettings();
  return OllamaChat(host, model, history, projectId, { requestId: opts.requestId || "", think: opts.think || "" });
};

// Draft a Conventional Commits message from a project's pending changes.
// The backend picks the diff: staged changes if anything is staged,
// otherwise everything uncommitted.
// Returns { subject, body, message, staged, fileCount, truncated }.
export const generateCommitMessage = (projectRoot) => {
  const { host, model } = getAISettings();
  return OllamaGenerateCommitMessage(host, model, projectRoot);
};
