// Small icon + title + copy row used across sections.
export default function IconFeat(props) {
  return (
    <div class="icon-feat">
      <span class={`ic ${props.color}`}>{props.icon}</span>
      <div>
        <strong>{props.title}</strong>
        <p>{props.children}</p>
      </div>
    </div>
  );
}
