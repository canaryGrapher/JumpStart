// Code context helpers. The Go side indexes a project's source tree into
// ~/.jumpstart/context/<projectId>.json; the chat then retrieves the most
// relevant chunks per question so answers are grounded in real code.
import {
  BuildCodeContext,
  GetCodeContextStatus,
  ClearCodeContext,
  SearchCodeContext,
  EventsOn,
  EventsOff,
} from "./api";

export const buildContext = (projectId) => BuildCodeContext(projectId);
export const getContextStatus = (projectId) => GetCodeContextStatus(projectId);
export const clearContext = (projectId) => ClearCodeContext(projectId);
export const searchContext = (projectId, query, limit = 10) =>
  SearchCodeContext(projectId, query, limit);

// Subscribe to indexing progress. Returns an unsubscribe function.
export const onIndexProgress = (projectId, handler) => {
  const listener = (payload) => {
    if (!payload || payload.projectId !== projectId) return;
    handler(payload);
  };
  EventsOn("codectx:progress", listener);
  return () => EventsOff("codectx:progress");
};

// Human-readable summary of an index, e.g. "412 files · 1,830 chunks".
export const describeIndex = (status) => {
  if (!status?.indexed) return "Not indexed";
  const { files, chunks } = status.stats || {};
  return `${(files || 0).toLocaleString()} files · ${(chunks || 0).toLocaleString()} chunks`;
};

export const indexAge = (status) => {
  const t = status?.stats?.builtAt;
  if (!t) return "";
  const mins = Math.floor((Date.now() - t) / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
};
