// Host OS detection, used only to label OS-specific actions with the name the
// user's own desktop uses ("Finder", not "File manager"). The Wails webview
// reports the host in its user agent; nothing behavioural depends on this, so
// a wrong guess costs a word on a button and nothing else.
const ua = typeof navigator === "undefined" ? "" : navigator.userAgent || "";

export const isMac = /Macintosh|Mac OS X/i.test(ua);
export const isWindows = /Windows/i.test(ua);

// fileManagerName is what the platform calls its file browser.
export const fileManagerName = isMac
  ? "Finder"
  : isWindows
    ? "Explorer"
    : "Files";
