// Parse ANSI SGR escape sequences (colors/bold) into styled segments so
// terminal output like the Wails CLI renders with color instead of showing
// raw escape codes as garbage. Returns an array of { text, style } segments.

// Standard 8 colors (30-37 / 40-47) and bright variants (90-97 / 100-107).
const COLORS = [
  "#3b3f45", // black
  "#e06c75", // red
  "#98c379", // green
  "#d19a66", // yellow
  "#61afef", // blue
  "#c678dd", // magenta
  "#56b6c2", // cyan
  "#d4d6d9", // white
];
const BRIGHT = [
  "#5c6370",
  "#f87171",
  "#98c379",
  "#e5c07b",
  "#61afef",
  "#c678dd",
  "#56b6c2",
  "#ffffff",
];

// eslint-disable-next-line no-control-regex
const ANSI_RE = /\x1b\[([0-9;]*)m/g;

// applyCodes mutates a style state object from a list of SGR numeric codes.
function applyCodes(state, codes) {
  for (let i = 0; i < codes.length; i++) {
    const c = codes[i];
    if (c === 0) {
      state.color = null;
      state.bold = false;
    } else if (c === 1) {
      state.bold = true;
    } else if (c === 22) {
      state.bold = false;
    } else if (c === 39) {
      state.color = null;
    } else if (c >= 30 && c <= 37) {
      state.color = COLORS[c - 30];
    } else if (c >= 90 && c <= 97) {
      state.color = BRIGHT[c - 90];
    }
  }
}

function styleFor(state) {
  const style = {};
  if (state.color) style.color = state.color;
  if (state.bold) style.fontWeight = 600;
  return style;
}

// parseAnsi turns a string into styled segments, carrying color state across
// the whole string (SGR state persists until reset).
export function parseAnsi(text) {
  const segments = [];
  const state = { color: null, bold: false };
  let last = 0;
  let match;

  ANSI_RE.lastIndex = 0;
  while ((match = ANSI_RE.exec(text)) !== null) {
    if (match.index > last) {
      segments.push({ text: text.slice(last, match.index), style: styleFor(state) });
    }
    const codes = match[1] === "" ? [0] : match[1].split(";").map(Number);
    applyCodes(state, codes);
    last = ANSI_RE.lastIndex;
  }
  if (last < text.length) {
    segments.push({ text: text.slice(last), style: styleFor(state) });
  }
  return segments;
}
