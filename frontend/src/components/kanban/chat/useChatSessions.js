import { useCallback, useEffect, useRef, useState } from "react";
import useAIProgress from "../../../hooks/useAIProgress";
import {
  listChats,
  getChat,
  deleteChat,
  sendMessage,
} from "../../../chats";

// Owns the saved conversations for one project: which thread is open, its
// messages, and sending a turn. Sessions are persisted by the Go side, so
// this hook only mirrors what came back from disk.
export default function useChatSessions(projectId, onError) {
  const [sessions, setSessions] = useState([]);
  const [activeId, setActiveId] = useState("");
  const [messages, setMessages] = useState([]);
  const [busy, setBusy] = useState(false);
  const [pending, setPending] = useState(null); // optimistic user turn
  const progress = useAIProgress();

  // Held in a ref so a caller passing an inline arrow doesn't retrigger
  // every callback (and, through them, the load effect) on each render.
  const errRef = useRef(onError);
  errRef.current = onError;
  const fail = (e) => errRef.current && errRef.current(String(e));

  const refreshList = useCallback(async () => {
    if (!projectId) return;
    try {
      setSessions((await listChats(projectId)) || []);
    } catch (e) {
      fail(e);
    }
  }, [projectId]);

  // Switching project resets the open thread as well as the list.
  useEffect(() => {
    setActiveId("");
    setMessages([]);
    setPending(null);
    refreshList();
  }, [projectId, refreshList]);

  // Open an existing thread and load its history.
  const open = useCallback(
    async (sessionId) => {
      if (!sessionId) return;
      try {
        const s = await getChat(projectId, sessionId);
        setActiveId(s.id);
        setMessages(s.messages || []);
        setPending(null);
      } catch (e) {
        fail(e);
      }
    },
    [projectId]
  );

  // Start a fresh thread. The session row is created lazily on first send,
  // so clicking "New chat" repeatedly never litters the list.
  const startNew = useCallback(() => {
    setActiveId("");
    setMessages([]);
    setPending(null);
  }, []);

  const remove = useCallback(
    async (sessionId) => {
      try {
        await deleteChat(projectId, sessionId);
        if (sessionId === activeId) startNew();
        await refreshList();
      } catch (e) {
        fail(e);
      }
    },
    [projectId, activeId, startNew, refreshList]
  );

  // Send one turn into the active thread, creating it if needed.
  // think overrides Settings > AI for this turn ("" = use the setting).
  const send = useCallback(
    async (text, think = "") => {
      const body = (text || "").trim();
      if (!body || busy) return;

      setPending({ id: "pending", role: "user", content: body });
      setBusy(true);
      const requestId = progress.begin();
      try {
        const session = await sendMessage(projectId, activeId, body, { requestId, think });
        setActiveId(session.id);
        setMessages(session.messages || []);
        setPending(null);
        await refreshList();
      } catch (e) {
        fail(e);
        setPending(null);
        // The backend saved the user turn plus an error reply, so reload
        // to keep what's on screen in step with what's on disk.
        if (activeId) await open(activeId);
        await refreshList();
      } finally {
        progress.end();
        setBusy(false);
      }
    },
    [projectId, activeId, busy, refreshList, open, progress]
  );

  return {
    sessions,
    activeId,
    messages: pending ? [...messages, pending] : messages,
    busy,
    progress,
    stop: progress.stop,
    open,
    startNew,
    remove,
    send,
  };
}
