import { useEffect, useRef, useState } from "react";
import { GetLogs, EventsOn } from "../api";
import { parseAnsi } from "../ansi";

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
      {lines.length
        ? parseAnsi(lines.join("\n")).map((seg, i) => (
            <span key={i} style={seg.style}>
              {seg.text}
            </span>
          ))
        : "No output yet."}
    </div>
  );
}
