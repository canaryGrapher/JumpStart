export default function VolumeList({ volumes }) {
  if (!volumes.length) {
    return <div className="sub">No named volumes for this stack.</div>;
  }

  return (
    <div className="dk-table">
      <div className="dk-row dk-head dk-3col">
        <span>Name</span>
        <span>Driver</span>
        <span>Scope</span>
      </div>
      {volumes.map((v, i) => (
        <div className="dk-row dk-3col" key={v.name || i}>
          <span className="dk-name">{v.name}</span>
          <span className="dk-muted">{v.driver || "—"}</span>
          <span className="dk-muted">{v.scope || "—"}</span>
        </div>
      ))}
    </div>
  );
}
