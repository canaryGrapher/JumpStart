// AI helpers wrapping the Ollama Wails bindings. Model + host live in
// localStorage and are edited from Preferences → AI.
import { OllamaListModels, OllamaEnrichTask, OllamaChat } from "./api";

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

// Ask the model to flesh out one item. Returns
// { description, acceptance[], subtasks[], priority, labels[] }.
export const enrichTask = (title, kind, projectContext = "") => {
  const { host, model } = getAISettings();
  return OllamaEnrichTask(host, model, title, kind, projectContext);
};

// Drive the story-generating chat. history is [{role, content}, ...].
// Returns { reply, stories[] }.
export const chat = (history, projectContext = "") => {
  const { host, model } = getAISettings();
  return OllamaChat(host, model, history, projectContext);
};
