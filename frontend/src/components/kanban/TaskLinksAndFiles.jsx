import { useEffect, useRef, useState } from "react";
import {
  PickTaskAttachments,
  AddTaskAttachmentData,
  TaskAttachmentPreview,
  OpenTaskAttachment,
  PreviewTaskAttachment,
  RevealTaskAttachment,
  OpenTaskLink,
  AttachmentTextStatus,
  RerunAttachmentText,
} from "../../api";
import { uid } from "./columns";

const MAX_FILE_BYTES = 100 * 1024 * 1024;

const IMAGE_MIMES = new Set([
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
  "image/bmp",
  "image/svg+xml",
]);
export const isImageAttachment = (a) => IMAGE_MIMES.has((a.mime || "").toLowerCase());

const humanSize = (n) => {
  if (n >= 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  if (n >= 1024) return `${Math.round(n / 1024)} KB`;
  return `${n} B`;
};

// Turn what the user typed into an http(s)/mailto URL, or "" if it is not
// one we will open. Bare "example.com/x" gets https://. The backend checks
// again before opening, so this is only for fast feedback.
export const normalizeLinkUrl = (raw) => {
  const s = (raw || "").trim();
  if (!s) return "";
  if (/^(https?:\/\/\S+|mailto:\S+)$/i.test(s)) return s;
  // Any other scheme (javascript:, file:, data:, ftp:) is refused. A
  // host:port such as localhost:3000 is not a scheme.
  if (/^[a-z][a-z0-9+.-]*:(?!\d+([/?#]|$))/i.test(s)) return "";
  if (/\s/.test(s)) return "";
  return `https://${s}`;
};

// --- Links ---

export function TaskLinksField({ links = [], onChange, onError }) {
  const [url, setUrl] = useState("");
  const [title, setTitle] = useState("");

  const add = () => {
    const normalized = normalizeLinkUrl(url);
    if (!normalized) {
      onError && onError("Enter a web address (http, https) or a mailto: link.");
      return;
    }
    setUrl("");
    setTitle("");
    onChange([...links, { id: uid(), title: title.trim(), url: normalized }]);
  };

  const open = async (link) => {
    try {
      await OpenTaskLink(link.url);
    } catch (e) {
      onError && onError(String(e));
    }
  };

  return (
    <div className="field">
      <label>Links</label>
      {links.map((l) => (
        <div className="task-row kb-link-row" key={l.id}>
          <button
            type="button"
            className="kb-link"
            title={`Open in your browser: ${l.url}`}
            onClick={() => open(l)}
          >
            <span className="kb-link-title">{l.title || l.url}</span>
            {l.title && <span className="kb-link-url">{l.url}</span>}
          </button>
          <button
            type="button"
            className="link-btn"
            onClick={() => onChange(links.filter((x) => x.id !== l.id))}
          >
            Remove
          </button>
        </div>
      ))}
      <div className="row">
        <input
          value={url}
          placeholder="https://…"
          onChange={(e) => setUrl(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && add()}
        />
        <input
          value={title}
          placeholder="Label (optional)"
          onChange={(e) => setTitle(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && add()}
        />
        <button type="button" className="btn small" onClick={add}>
          Add
        </button>
      </div>
    </div>
  );
}

// --- Image viewer ---

function ImageViewer({ attachment, src, onOpen, onPreview, onClose }) {
  useEffect(() => {
    const onKey = (e) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="modal-overlay kb-viewer-overlay" onClick={onClose}>
      <div
        className="kb-viewer"
        role="dialog"
        aria-label={attachment.name}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="kb-viewer-bar">
          <span className="kb-viewer-name">{attachment.name}</span>
          <div className="spacer" />
          <button type="button" className="btn small" onClick={onPreview}>
            Preview
          </button>
          <button type="button" className="btn small" onClick={onOpen}>
            Open
          </button>
          <button type="button" className="btn small" onClick={onClose}>
            Close
          </button>
        </div>
        <div className="kb-viewer-stage">
          <img src={src} alt={attachment.name} />
        </div>
      </div>
    </div>
  );
}

// --- Attachments ---

const readAsDataUrl = (file) =>
  new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(r.result);
    r.onerror = () => reject(r.error || new Error("could not read file"));
    r.readAsDataURL(file);
  });

// Whether an image's text has been read for search, with a way to read it
// again. Files not yet saved on the task have no status and show nothing.
function OcrStatus({ projectId, taskId, attachment }) {
  const [info, setInfo] = useState(null);
  useEffect(() => {
    let live = true;
    let timer;
    const poll = () =>
      AttachmentTextStatus(projectId, taskId, attachment.id)
        .then((i) => {
          if (!live) return;
          setInfo(i);
          if (i.state === "pending") timer = setTimeout(poll, 1500);
        })
        .catch(() => live && setInfo(null));
    poll();
    return () => {
      live = false;
      clearTimeout(timer);
    };
  }, [projectId, taskId, attachment.id]);
  if (!info) return null;
  const label =
    info.state === "pending"
      ? "reading text…"
      : info.state === "ready"
        ? `searchable text (${info.chars} chars)`
        : info.error
          ? "text could not be read"
          : "no text found";
  return (
    <span className="kb-attachment-ocr" title={info.error || ""}>
      {" · "}
      {label}{" "}
      {info.state !== "pending" && (
        <button
          type="button"
          className="link-btn"
          onClick={(e) => {
            e.stopPropagation();
            RerunAttachmentText(projectId, taskId, attachment.id).then(setInfo).catch(() => {});
            setTimeout(() => AttachmentTextStatus(projectId, taskId, attachment.id).then(setInfo).catch(() => {}), 1500);
          }}
        >
          Read again
        </button>
      )}
    </span>
  );
}

export function TaskAttachmentsField({
  projectId,
  taskId,
  attachments = [],
  onAdd,
  onRemove,
  onError,
}) {
  const [busy, setBusy] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const [thumbs, setThumbs] = useState({}); // attachment id -> data URL | "" (failed)
  const [viewing, setViewing] = useState(null);
  const requested = useRef(new Set());

  // Load thumbnails for images as they appear. A failure (file too large to
  // inline, or missing) just leaves the generic file icon.
  useEffect(() => {
    for (const a of attachments) {
      if (!isImageAttachment(a) || requested.current.has(a.id)) continue;
      requested.current.add(a.id);
      TaskAttachmentPreview(projectId, taskId, a)
        .then((src) => setThumbs((t) => ({ ...t, [a.id]: src })))
        .catch(() => setThumbs((t) => ({ ...t, [a.id]: "" })));
    }
  }, [attachments, projectId, taskId]);

  const fail = (e) => onError && onError(String(e));

  const pick = async () => {
    setBusy(true);
    try {
      const added = await PickTaskAttachments(projectId, taskId);
      if (added && added.length) onAdd(added);
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  };

  const addFiles = async (files) => {
    const list = Array.from(files || []).filter((f) => f && f.size >= 0);
    if (!list.length) return;
    setBusy(true);
    const added = [];
    try {
      for (const f of list) {
        if (f.size > MAX_FILE_BYTES) {
          fail(`${f.name} is larger than ${MAX_FILE_BYTES / (1024 * 1024)} MB.`);
          continue;
        }
        const dataUrl = await readAsDataUrl(f);
        added.push(await AddTaskAttachmentData(projectId, taskId, f.name || "pasted-file", dataUrl));
      }
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
      if (added.length) onAdd(added);
    }
  };

  const onPaste = (e) => {
    const files = Array.from(e.clipboardData?.files || []);
    if (!files.length) return;
    e.preventDefault();
    // Screenshots arrive as an unnamed "image.png"; give them a useful name.
    const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, "-");
    addFiles(
      files.map((f) =>
        f.name && f.name !== "image.png"
          ? f
          : new File([f], `pasted-image-${stamp}.${(f.type.split("/")[1] || "png").replace("jpeg", "jpg")}`, {
              type: f.type,
            })
      )
    );
  };

  const run = (fn, a) => async (e) => {
    e && e.stopPropagation();
    try {
      await fn(projectId, taskId, a);
    } catch (err) {
      fail(err);
    }
  };

  return (
    <div
      className={`field kb-attachments ${dragOver ? "drag-over" : ""}`}
      tabIndex={-1}
      onPaste={onPaste}
      onDragOver={(e) => {
        if (e.dataTransfer?.types?.includes("Files")) {
          e.preventDefault();
          setDragOver(true);
        }
      }}
      onDragLeave={() => setDragOver(false)}
      onDrop={(e) => {
        if (!e.dataTransfer?.files?.length) return;
        e.preventDefault();
        setDragOver(false);
        addFiles(e.dataTransfer.files);
      }}
    >
      <label>Attachments</label>
      {attachments.map((a) => {
        const image = isImageAttachment(a);
        const thumb = thumbs[a.id];
        const primary = image && thumb ? () => setViewing(a) : run(OpenTaskAttachment, a);
        return (
          <div className="task-row kb-attachment" key={a.id}>
            <button
              type="button"
              className="kb-attachment-main"
              title={image && thumb ? "View here" : "Open with the default app"}
              onClick={primary}
            >
              <span className="kb-attachment-thumb">
                {image && thumb ? <img src={thumb} alt="" /> : <span aria-hidden>📄</span>}
              </span>
              <span className="kb-attachment-text">
                <span className="kb-attachment-name">{a.name}</span>
                <span className="kb-attachment-meta">
                  {humanSize(a.size)}
                  {a.mime ? ` · ${a.mime}` : ""}
                </span>
              </span>
            </button>
            {image && projectId && taskId && (
              <span className="kb-attachment-meta">
                <OcrStatus projectId={projectId} taskId={taskId} attachment={a} />
              </span>
            )}
            <button
              type="button"
              className="btn tiny"
              title="Quick look (Preview on a Mac)"
              onClick={run(PreviewTaskAttachment, a)}
            >
              Preview
            </button>
            <button
              type="button"
              className="btn tiny"
              title="Open with the default app"
              onClick={run(OpenTaskAttachment, a)}
            >
              Open
            </button>
            <button
              type="button"
              className="link-btn"
              title="Show in Finder"
              onClick={run(RevealTaskAttachment, a)}
            >
              Reveal
            </button>
            <button type="button" className="link-btn" onClick={() => onRemove(a)}>
              Remove
            </button>
          </div>
        );
      })}
      <div className="row">
        <button type="button" className="btn small" onClick={pick} disabled={busy}>
          {busy ? "Adding…" : "Add files…"}
        </button>
        <span className="kb-attachments-hint">
          or drop files here, or paste a screenshot (⌘V). Max 100 MB each.
        </span>
      </div>
      {viewing && thumbs[viewing.id] && (
        <ImageViewer
          attachment={viewing}
          src={thumbs[viewing.id]}
          onOpen={run(OpenTaskAttachment, viewing)}
          onPreview={run(PreviewTaskAttachment, viewing)}
          onClose={() => setViewing(null)}
        />
      )}
    </div>
  );
}
