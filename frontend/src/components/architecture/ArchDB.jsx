import { useState } from "react";

export default function ArchDB({ entities = [], relations = [] }) {
  const [hover, setHover] = useState(null);
  if (!entities.length) {
    return <p className="arch-empty">No database entities in this section.</p>;
  }

  return (
    <div className="arch-db">
      <div className="arch-db-entities">
        {entities.map((e) => (
          <article
            key={e.name}
            className={`arch-entity ${hover === e.name ? "on" : ""}`}
            onMouseEnter={() => setHover(e.name)}
            onMouseLeave={() => setHover(null)}
            title={e.description || e.name}
          >
            <header>
              <h4>{e.name}</h4>
              {e.description ? <p className="dim">{e.description}</p> : null}
            </header>
            {e.columns?.length ? (
              <table>
                <thead>
                  <tr>
                    <th>Column</th>
                    <th>Type</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {e.columns.map((c) => (
                    <tr key={c.name} title={c.description || c.name}>
                      <td className="mono">{c.name}</td>
                      <td className="dim">{c.type || "—"}</td>
                      <td>{c.pk ? <span className="arch-pk">PK</span> : null}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : null}
          </article>
        ))}
      </div>

      {relations?.length > 0 && (
        <div className="arch-relations">
          <h4>Relations</h4>
          <ul>
            {relations.map((r, i) => (
              <li
                key={`${r.from}-${r.to}-${i}`}
                title={r.description || `${r.from} → ${r.to}`}
                onMouseEnter={() => setHover(r.from)}
                onMouseLeave={() => setHover(null)}
              >
                <span className="mono">{r.from}</span>
                <span className="arch-rel-type">{r.type || "→"}</span>
                <span className="mono">{r.to}</span>
                {r.description ? <span className="dim"> — {r.description}</span> : null}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
