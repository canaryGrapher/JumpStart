// Chat session helpers. Threads are persisted by the Go side in
// ~/.jumpstart/chats/<projectId>.json, so history survives restarts and a
// user can pick up an old conversation or start a fresh one.
import {
  ListChats,
  GetChat,
  NewChat,
  RenameChat,
  DeleteChat,
  SendChatMessage,
} from "./api";
import { getAISettings } from "./ai";

export const listChats = (projectId) => ListChats(projectId);
export const getChat = (projectId, sessionId) => GetChat(projectId, sessionId);
export const newChat = (projectId, title = "") => NewChat(projectId, title);
export const renameChat = (projectId, sessionId, title) =>
  RenameChat(projectId, sessionId, title);
export const deleteChat = (projectId, sessionId) =>
  DeleteChat(projectId, sessionId);

// Send one turn. Passing an empty sessionId starts a new thread server-side,
// so the caller never needs two round trips. Resolves to the whole updated
// session (messages included).
export const sendMessage = (projectId, sessionId, text, opts = {}) => {
  const { host, model } = getAISettings();
  return SendChatMessage(host, model, projectId, sessionId || "", text, {
    requestId: opts.requestId || "",
    think: opts.think || "",
  });
};

// Parse the stories blob stored alongside an assistant message.
export const storiesOf = (message) => {
  const raw = message?.stories;
  if (!raw) return [];
  if (Array.isArray(raw)) return raw;
  try {
    return JSON.parse(raw) || [];
  } catch {
    return [];
  }
};
