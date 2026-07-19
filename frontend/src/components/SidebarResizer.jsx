/* Slim drag handle on the sidebar edge. Double-click resets the width. */
export default function SidebarResizer({ onResizeStart, onReset }) {
  return (
    <div
      className="sidebar-resizer"
      title="Drag to resize · double-click to reset"
      onPointerDown={onResizeStart}
      onDoubleClick={onReset}
    />
  );
}
