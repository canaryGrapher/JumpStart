import { OPTION_COLORS, isWritable, LABELS } from "./fieldTypes";

// One editor for one Projects v2 field. Writable types get a real
// control; read-only rollups GitHub computes render as a static row so
// the card still shows everything the board shows.
export default function FieldEditor({ field, value, onChange, disabled }) {
  const v = value || {};

  if (!isWritable(field)) {
    return (
      <div className="gh-field gh-field-readonly">
        <label>{field.name}</label>
        <div className="gh-readonly-value">
          {v.display || <span className="gh-muted">Not set</span>}
          <span className="gh-type-tag">{LABELS[field.dataType] || field.dataType}</span>
        </div>
      </div>
    );
  }

  const patch = (next) =>
    onChange({
      ...v,
      fieldId: field.id,
      name: field.name,
      dataType: field.dataType,
      ...next,
    });

  return (
    <div className="gh-field">
      <label>{field.name}</label>
      {field.dataType === "TEXT" || field.dataType === "TITLE" ? (
        <input
          value={v.text || ""}
          disabled={disabled}
          placeholder={`${field.name}…`}
          onChange={(e) => patch({ text: e.target.value })}
        />
      ) : null}

      {field.dataType === "NUMBER" ? (
        <input
          type="number"
          value={v.number ?? ""}
          disabled={disabled}
          onChange={(e) =>
            patch({ number: e.target.value === "" ? null : Number(e.target.value) })
          }
        />
      ) : null}

      {field.dataType === "DATE" ? (
        <input
          type="date"
          value={v.date || ""}
          disabled={disabled}
          onChange={(e) => patch({ date: e.target.value })}
        />
      ) : null}

      {field.dataType === "SINGLE_SELECT" ? (
        <div className="gh-options">
          {(field.options || []).map((o) => (
            <button
              key={o.id}
              type="button"
              disabled={disabled}
              title={o.description || o.name}
              className={`gh-option ${v.optionId === o.id ? "active" : ""}`}
              style={{ "--gh-dot": OPTION_COLORS[o.color] || OPTION_COLORS.GRAY }}
              onClick={() =>
                patch(
                  v.optionId === o.id
                    ? { optionId: "", optionKey: "" }
                    : { optionId: o.id, optionKey: o.name }
                )
              }
            >
              <span className="gh-dot" />
              {o.name}
            </button>
          ))}
        </div>
      ) : null}

      {field.dataType === "ITERATION" ? (
        <select
          value={v.iterationId || ""}
          disabled={disabled}
          onChange={(e) => patch({ iterationId: e.target.value })}
        >
          <option value="">No iteration</option>
          {(field.iterations || []).map((it) => (
            <option key={it.id} value={it.id}>
              {it.title}
              {it.completed ? " (completed)" : ""}
              {it.startDate ? ` · ${it.startDate}` : ""}
            </option>
          ))}
        </select>
      ) : null}
    </div>
  );
}
