// Colored rounded-square project glyph, matching .project-icon.avatar in the app.
export default function ProjIcon(props) {
  const letter = props.g || (props.n || "?").charAt(0);
  return (
    <span
      class="proj-av"
      classList={{ lg: props.lg }}
      style={{ background: props.c || "var(--accent)", color: "#fff" }}
      aria-hidden="true"
    >
      {letter}
    </span>
  );
}
