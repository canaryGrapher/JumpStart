import { useMemo } from "react";
import { BrowserOpenURL } from "../../api";

// Minimal markdown renderer for GitHub release bodies. It deliberately
// handles only what release notes actually use — headings, bullets, bold,
// code and links — and renders everything else as plain text, so no
// untrusted HTML from a release body can reach the DOM.
function NoteLink({ href, children }) {
  return (
    <a
      href={href}
      title={href}
      onClick={(e) => {
        e.preventDefault();
        BrowserOpenURL(href);
      }}
    >
      {children}
    </a>
  );
}

function trimBareUrl(url) {
  return url.replace(/[),.;:!?]+$/g, "");
}

function inline(text, keyPrefix) {
  const out = [];
  const pattern =
    /(\[([^\]]+)\]\((https?:\/\/[^\s)]+)\))|(`([^`]+)`)|(\*\*([^*]+)\*\*)|(https?:\/\/[^\s<>\[\]"'<>]+)/g;
  let last = 0;
  let m;
  let i = 0;
  while ((m = pattern.exec(text)) !== null) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const key = `${keyPrefix}-${i++}`;
    if (m[1]) {
      out.push(
        <NoteLink key={key} href={m[3]}>
          {m[2]}
        </NoteLink>
      );
    } else if (m[4]) {
      out.push(<code key={key}>{m[5]}</code>);
    } else if (m[6]) {
      out.push(<strong key={key}>{m[7]}</strong>);
    } else {
      const href = trimBareUrl(m[8]);
      out.push(
        <NoteLink key={key} href={href}>
          {href}
        </NoteLink>
      );
    }
    last = pattern.lastIndex;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

function render(md) {
  const blocks = [];
  let bullets = [];

  const flush = () => {
    if (bullets.length === 0) return;
    blocks.push(
      <ul key={`ul-${blocks.length}`}>
        {bullets.map((b, i) => (
          <li key={i}>{inline(b, `li-${blocks.length}-${i}`)}</li>
        ))}
      </ul>
    );
    bullets = [];
  };

  for (const raw of String(md).split(/\r?\n/)) {
    const line = raw.trimEnd();
    const bullet = line.match(/^\s*[-*+]\s+(.*)$/);
    if (bullet) {
      bullets.push(bullet[1]);
      continue;
    }
    flush();
    if (!line.trim()) continue;
    const heading = line.match(/^(#{1,6})\s+(.*)$/);
    if (heading) {
      blocks.push(
        <h4 key={`h-${blocks.length}`}>{inline(heading[2], `h-${blocks.length}`)}</h4>
      );
      continue;
    }
    blocks.push(<p key={`p-${blocks.length}`}>{inline(line, `p-${blocks.length}`)}</p>);
  }
  flush();
  return blocks;
}

export default function ReleaseNotes({ notes, className = "" }) {
  const body = useMemo(() => render(notes || ""), [notes]);
  if (!notes || !notes.trim()) {
    return <p className={`release-notes empty ${className}`}>No release notes provided.</p>;
  }
  return <div className={`release-notes ${className}`}>{body}</div>;
}
