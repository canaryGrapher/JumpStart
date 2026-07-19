// The real app icon, served from /public/icon.png. Replace that file to
// update the logo everywhere (nav, footer, window title, favicon).
export default function RocketLogo(props) {
  return (
    <img
      src="/icon.png"
      width={props.size || 26}
      height={props.size || 26}
      alt="JumpStart logo"
      style="border-radius:22%"
    />
  );
}
