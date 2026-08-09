import { useEffect, useRef, useState } from "react";
import { GetLogs, EventsOn } from "../api";
import { parseAnsi } from "../ansi";
import { captureOnce } from "../analytics";

export default function LogPanel({ procId, source = "process" }) {
  const [lines, setLines] = useState([]);
  const boxRef = useRef(null);

  useEffect(() => {
    GetLogs(procId).then((l) => {
      const list = l || [];
      setLines(list);
      const src = ["process", "script", "test"].includes(source) ? source : "process";
      captureOnce(`logs:${src}:${procId}`, "logs_opened", {
        line_count: list.length,
        source: src,
      });
    });
    const off = EventsOn(`log:${procId}`, (line) =>
      setLines((prev) => [...prev.slice(-1999), line])
    );
    return off;
  }, [procId, source]);

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
