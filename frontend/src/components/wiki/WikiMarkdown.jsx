import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { BrowserOpenURL } from "../../api";

// Resolve a markdown href into either an external URL or a wiki page slug.
// GitHub wiki links omit .md and use dashes for spaces: [Code Structure](Code-Structure).
export function resolveWikiHref(href) {
  const raw = String(href || "").trim();
  if (!raw || raw.startsWith("#")) {
    return { kind: "hash", href: raw };
  }
  if (/^(https?:|mailto:)/i.test(raw)) {
    return { kind: "external", href: raw };
  }
  // Strip optional ./ and .md; reject path escapes.
  let name = raw.replace(/^\.\//, "");
  if (name.includes("..") || name.includes("/") || name.includes("\\")) {
    return { kind: "external", href: raw };
  }
  if (/\.md$/i.test(name)) name = name.slice(0, -3);
  name = name.replace(/ /g, "-");
  if (!name) return { kind: "hash", href: "#" };
  return { kind: "page", name };
}

function WikiLink({ href, children, onNavigate, currentPage }) {
  const resolved = resolveWikiHref(href);

  if (resolved.kind === "page") {
    const active = currentPage && resolved.name === currentPage;
    return (
      <a
        href={`#wiki/${resolved.name}`}
        className={active ? "active" : undefined}
        aria-current={active ? "page" : undefined}
        onClick={(e) => {
          e.preventDefault();
          onNavigate?.(resolved.name);
        }}
      >
        {children}
      </a>
    );
  }

  if (resolved.kind === "external") {
    return (
      <a
        href={resolved.href}
        title={resolved.href}
        onClick={(e) => {
          e.preventDefault();
          BrowserOpenURL(resolved.href);
        }}
      >
        {children}
      </a>
    );
  }

  return <a href={resolved.href || "#"}>{children}</a>;
}

export default function WikiMarkdown({
  source,
  onNavigate,
  currentPage,
  className = "",
}) {
  const md = String(source || "");
  if (!md.trim()) return null;

  return (
    <div className={`wiki-md ${className}`.trim()}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ href, children }) => (
            <WikiLink
              href={href}
              onNavigate={onNavigate}
              currentPage={currentPage}
            >
              {children}
            </WikiLink>
          ),
          // Keep images inert — local wiki assets are not served by the webview.
          img: ({ alt, title }) => (
            <span className="wiki-md-img" title={title || alt || "image"}>
              {alt || "image"}
            </span>
          ),
        }}
      >
        {md}
      </ReactMarkdown>
    </div>
  );
}
