import { useCallback, useEffect, useRef, useState } from "react";
import { cancelAIRequest, newRequestId, onAIDelta } from "../ai";

// Tracks one in-flight AI request: elapsed time, the streamed reasoning
// trace, how much of the answer has arrived, and a stop() that cancels the
// backend call. begin() returns the request id to pass to the backend.
export default function useAIProgress() {
  const [state, setState] = useState(null); // { id, startedAt, thinking, chars }
  const [now, setNow] = useState(Date.now());
  const offRef = useRef(null);

  const end = useCallback(() => {
    offRef.current && offRef.current();
    offRef.current = null;
    setState(null);
  }, []);

  const begin = useCallback(() => {
    offRef.current && offRef.current();
    const id = newRequestId();
    setState({ id, startedAt: Date.now(), thinking: "", chars: 0 });
    offRef.current = onAIDelta(id, (d) =>
      setState((s) =>
        s && s.id === id
          ? { ...s, thinking: s.thinking + (d.thinking || ""), chars: s.chars + (d.content || "").length }
          : s
      )
    );
    return id;
  }, []);

  const stop = useCallback(() => {
    if (state) cancelAIRequest(state.id);
  }, [state]);

  useEffect(() => {
    if (!state) return undefined;
    const t = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(t);
  }, [state]);

  useEffect(() => () => offRef.current && offRef.current(), []);

  const elapsed = state ? Math.max(0, Math.round((now - state.startedAt) / 1000)) : 0;
  return { active: !!state, begin, end, stop, elapsed, thinking: state?.thinking || "", chars: state?.chars || 0 };
}

export const formatElapsed = (s) => (s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, "0")}s`);
