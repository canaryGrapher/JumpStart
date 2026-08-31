// GitHub's Projects v2 API only hands back a fixed set of column-color
// *names* (see internal/github/create.go, StatusColumn.Color) — GRAY,
// BLUE, GREEN, YELLOW, ORANGE, RED, PINK, PURPLE. Those are not CSS
// colors: setting `--col-color: YELLOW` used to work by accident (CSS
// color keywords are case-insensitive, so it resolved to pure #FFFF00),
// which is why "In Progress" rendered as unreadable bright-yellow text
// on a barely-tinted yellow chip. This maps each name to one of the
// app's own theme tokens (falling back to a fixed hex for the two colors
// the theme doesn't define) so preset previews stay legible in both
// light and dark mode.
const GH_COLUMN_COLORS = {
  GRAY: "var(--dim)",
  BLUE: "var(--accent)",
  GREEN: "var(--green)",
  YELLOW: "var(--yellow)",
  ORANGE: "#c2660d",
  RED: "var(--red)",
  PINK: "#d6338f",
  PURPLE: "var(--purple)",
};

export function ghColumnColor(name) {
  return GH_COLUMN_COLORS[String(name || "").toUpperCase()] || "var(--dim)";
}
