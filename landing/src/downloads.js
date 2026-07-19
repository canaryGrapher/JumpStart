// Download links for the current JumpStart release.
// Asset names are produced by .github/workflows/release.yml.
export const RELEASE_TAG = "v1.1.0";

const base = `https://github.com/canaryGrapher/JumpStart/releases/download/${RELEASE_TAG}`;

export const DOWNLOADS = {
  macos: `${base}/devdeck-${RELEASE_TAG}-macos-universal.zip`,
  windows: `${base}/devdeck-${RELEASE_TAG}-windows-amd64.zip`,
  linux: `${base}/devdeck-${RELEASE_TAG}-linux-amd64.tar.gz`,
};

export const RELEASES_PAGE = "https://github.com/canaryGrapher/JumpStart/releases";
