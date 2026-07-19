export default function ImageList({ images }) {
  if (!images.length) {
    return <div className="sub">No images for this stack yet.</div>;
  }

  return (
    <div className="dk-table">
      <div className="dk-row dk-head dk-3col">
        <span>Repository</span>
        <span>Tag</span>
        <span>Size</span>
      </div>
      {images.map((img, i) => (
        <div className="dk-row dk-3col" key={img.id || i}>
          <span className="dk-name" title={img.id}>{img.repository || "—"}</span>
          <span className="dk-muted">{img.tag || "—"}</span>
          <span className="dk-muted">{img.size || "—"}</span>
        </div>
      ))}
    </div>
  );
}
