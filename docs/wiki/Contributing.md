# Contributing

## Before you start

1. Read [Architecture](Architecture) and [Code-Structure](Code-Structure).
2. Run `wails dev` and confirm the app boots.
3. For privacy-sensitive work, read [Analytics-and-Privacy](Analytics-and-Privacy) and `docs/privacy.md`.

## Contribution recipes

### Add a Wails API method

1. Implement `func (a *App) YourMethod(...)` in the right `*_api.go` (or `app.go`).
2. Put non-trivial logic in `internal/<pkg>`.
3. Instrument with `a.track` / `outcome` if user-facing.
4. Regenerate bindings (`wails dev`).
5. Export in `frontend/src/api.js` and call from UI.

### Add a language / framework detector

1. Extend `detectFramework` / helpers in `internal/detect/frameworks.go`.
2. Add fixtures + cases in `detect_test.go`.
3. Keep `skipDirs` and `maxScanDepth` in mind for monorepo performance.
4. Optionally mirror script discovery in `internal/scripts` if manifests expose scripts.

### Add a project panel / tab

1. Tab in `ProjectView.jsx` + `trackPanel('name')`.
2. Component + SCSS partial + `@use` in `styles.scss`.
3. Follow existing btn/tabs/glass patterns — no parallel CSS framework.

### Add an analytics event

1. Emit only from Go (or via `analytics.js` bridge for UI-only facts).
2. Bounded properties; run redaction tests.
3. Update privacy docs if user-visible collection description changes.

### Touch process lifecycle

1. Prefer extending `procman` over ad-hoc `exec` in bindings.
2. Preserve process-group kill behaviour and event names.
3. Mark intentional stops through App helpers so crashes are not miscounted.

### Git / tokens / release

1. Never persist tokens in JSON — use `internal/secrets`.
2. Wrap mutating git/docker calls with `gitOp` / `dockerOp`.

### Platform-specific behaviour

1. Use build-tagged files (`_darwin`, `_windows`, `_unix`, `_other`).
2. Keep a compiling stub on other OSes.

## Code style

- Match neighbouring files (comments explain non-obvious *why*).
- Do not drive-by refactor unrelated code.
- Go module path remains `devdeck` until an intentional rename migration.
- Do not commit generated `frontend/wailsjs` noise unless the repo already tracks it for your branch policy — prefer local generation via Wails.

## Tests to run

```sh
go test ./...
cd frontend && npm run build
```

Add unit tests next to packages you change when behaviour is non-trivial (detect, redact, merge, semver).

## PR checklist

- [ ] Feature works under `wails dev` on your OS
- [ ] No paths/secrets/prompts in analytics props
- [ ] API exported through `api.js`
- [ ] SCSS partial wired if UI added
- [ ] Privacy/README/release notes updated when user-facing behaviour changes
- [ ] CI-relevant: Linux WebKit tag and ldflags not broken by your change

## Issue reports from the app

In-app Contribute flow (`contribute_api.go`) can attach redacted diagnostics. Prefer that path when reproducing desktop-only bugs.
