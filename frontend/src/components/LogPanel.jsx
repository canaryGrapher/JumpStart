import { useEffect, useRef, useState } from "react";
import { GetLogs, EventsOn } from "../api";

export default function LogPanel({ procId }) {
  const [lines, setLines] = useState([]);
  const boxRef = useRef(null);

  useEffect(() => {
    GetLogs(procId).then((l) => setLines(l || []));
    const off = EventsOn(`log:${procId}`, (line) =>
      setLines((prev) => [...prev.slice(-1999), line])
    );
    return off;
  }, [procId]);

  useEffect(() => {
    const el = boxRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lines]);

  return (
    <div className="logs" ref={boxRef} onClick={(e) => e.stopPropagation()}>
      {lines.length ? lines.join("\n") : "No output yet."}
    </div>
  );
}
