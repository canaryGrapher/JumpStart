import { useEffect, useMemo, useRef, useState } from "react";
import { openInApp } from "../../navigate";
import { useWidgetData } from "./TaskWidget";

// Content Security Policy for widget code: inline script and style only.
// No network (fetch, XHR, WebSocket, images from URLs), no frames, no forms.
export const WIDGET_CSP =
  "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data:; font-src data:; form-action 'none'; base-uri 'none'";

// The iframe is sandboxed with allow-scripts only: no allow-same-origin, so
// it runs in an opaque origin and cannot touch the app's DOM, storage,
// cookies or Wails bindings; no allow-popups, allow-forms, or top
// navigation. Data arrives only by postMessage.
export const WIDGET_SANDBOX = "allow-scripts";

export function widgetDocument(html, theme) {
  return `<!doctype html><html data-theme="${theme === "dark" ? "dark" : "light"}"><head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="${WIDGET_CSP}">
<style>
:root{color-scheme:light dark;--text:#1d1d1f;--dim:#6e6e73;--accent:#007aff;--red:#d70015;--bg:transparent}
html[data-theme=dark]{--text:#f5f5f7;--dim:#98989d;--accent:#0a84ff;--red:#ff453a}
html,body{margin:0;background:var(--bg);color:var(--text);font:13px/1.4 -apple-system,BlinkMacSystemFont,"Inter",sans-serif}
body{padding:2px}
</style>
<script>
(function(){
  var post=function(m){parent.postMessage(m,"*")};
  window.jumpstart={open:function(p,t){post({type:"jumpstart:open",projectId:p,taskId:t})}};
  new ResizeObserver(function(){post({type:"jumpstart:height",height:document.documentElement.scrollHeight})}).observe(document.documentElement);
})();
</script>
</head><body>${html}</body></html>`;
}

// Runs a user-supplied HTML/JS widget. The widget receives
// {type:"jumpstart:data", data, theme} and may post
// {type:"jumpstart:open", projectId, taskId} (only tasks it was given) or
// {type:"jumpstart:height", height}.
export default function HtmlWidget({ widget, version }) {
  const data = useWidgetData(widget, version);
  const frame = useRef(null);
  const [height, setHeight] = useState(widget.size === "s" ? 140 : 220);
  const theme = document.documentElement.dataset.theme === "dark" ? "dark" : "light";
  const srcDoc = useMemo(() => widgetDocument(widget.html || "", theme), [widget.html, theme]);

  const send = () => {
    if (data && frame.current?.contentWindow) {
      frame.current.contentWindow.postMessage({ type: "jumpstart:data", data, theme }, "*");
    }
  };
  useEffect(send, [data, theme]);

  useEffect(() => {
    const onMessage = (e) => {
      if (!frame.current || e.source !== frame.current.contentWindow) return;
      const m = e.data || {};
      if (m.type === "jumpstart:height" && Number.isFinite(m.height)) {
        setHeight(Math.max(60, Math.min(800, Math.ceil(m.height))));
      } else if (m.type === "jumpstart:open") {
        const ok = data?.tasks?.some((t) => t.projectId === m.projectId && t.taskId === m.taskId);
        if (ok) openInApp(m.projectId, m.taskId);
      }
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [data]);

  return (
    <iframe
      ref={frame}
      className="dw-html"
      title={widget.title || "Custom widget"}
      sandbox={WIDGET_SANDBOX}
      referrerPolicy="no-referrer"
      srcDoc={srcDoc}
      style={{ height }}
      onLoad={send}
    />
  );
}
