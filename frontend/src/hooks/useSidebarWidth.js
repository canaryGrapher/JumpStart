import { useCallback, useEffect, useRef, useState } from "react";

const KEY = "sidebarWidth";
export const SIDEBAR_MIN = 180;
export const SIDEBAR_MAX = 420;
export const SIDEBAR_DEFAULT = 240;

const clamp = (w) => Math.min(SIDEBAR_MAX, Math.max(SIDEBAR_MIN, w));

/* Sidebar width with drag-to-resize, persisted across restarts. */
export default function useSidebarWidth() {
  const [width, setWidth] = useState(() => {
    const saved = parseInt(localStorage.getItem(KEY), 10);
    return Number.isFinite(saved) ? clamp(saved) : SIDEBAR_DEFAULT;
  });
  const [resizing, setResizing] = useState(false);
  const drag = useRef(null); // { startX, startW }

  useEffect(() => {
    localStorage.setItem(KEY, String(width));
  }, [width]);

  const onResizeStart = useCallback(
    (e) => {
      e.preventDefault();
      drag.current = { startX: e.clientX, startW: width };
      setResizing(true);

      const onMove = (ev) =>
        setWidth(clamp(drag.current.startW + ev.clientX - drag.current.startX));
      const onUp = () => {
        setResizing(false);
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onUp);
      };
      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onUp);
    },
    [width]
  );

  const reset = useCallback(() => setWidth(SIDEBAR_DEFAULT), []);

  return { width, resizing, onResizeStart, reset };
}
