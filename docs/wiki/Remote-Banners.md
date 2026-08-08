# Remote Banners (in-app announcements)

JumpStart can show a dismissible announcement card (bottom-right) **without shipping a new app build**. Installed apps download a small JSON file from GitHub on launch.

This is how you “send a notification” to users.

## How it works

```
social/banner.json  (on branch main)
        │
        ▼  GET raw.githubusercontent.com
internal/banner.Fetch()  ← GetRemoteBanner()
        │
        ▼
useRemoteBanner.js  → first enabled, active, not-dismissed banner
        │
        ▼
AdOverlay.jsx  (promo / info / warning card + optional CTA)
```

**Fetch URL (fixed in code):**

```
https://raw.githubusercontent.com/canaryGrapher/JumpStart/main/social/banner.json
```

Defined as `banner.DefaultURL` in `internal/banner/banner.go`. There is **no** user-configurable URL.

## Publishing a new announcement

1. Edit [`social/banner.json`](https://github.com/canaryGrapher/JumpStart/blob/main/social/banner.json) in the repo.
2. **Prepend** your banner (or place it above older ones you still want shown). The UI shows the **first** entry that is enabled, in schedule, and not already dismissed.
3. Give it a **new unique `id`**. Reusing an id that users already dismissed will never show again on those machines.
4. Commit and **push to `main`**. Local-only or PR-branch edits are invisible to installed apps.
5. Ask users to **restart JumpStart** (or wait until next launch). The hook fetches once per session; it does not poll.

### Minimal example

```json
[
  {
    "id": "my-announcement-2026-08",
    "enabled": true,
    "title": "Short title",
    "message": "One or two sentences. Keep it scannable.",
    "linkUrl": "https://github.com/canaryGrapher/JumpStart/wiki",
    "linkText": "Learn more",
    "imageUrl": "",
    "style": "promo",
    "startsAt": "",
    "endsAt": ""
  }
]
```

The file may be either a **JSON array** of banners or a **single object**. Prefer an array so you can keep history and order.

## Field reference

| Field | Required | Notes |
|-------|----------|--------|
| `id` | Yes | Stable string; used for dismissals and analytics (`banner_id`). Empty ids are skipped. |
| `enabled` | Yes | `false` hides it without deleting the entry. |
| `title` | No | Bold heading in the card. |
| `message` | No | Body text. |
| `linkUrl` | No | Opens in the system browser via `BrowserOpenURL`. |
| `linkText` | No | Button label; defaults to “Learn more” if `linkUrl` is set. |
| `imageUrl` | No | Optional image above the text (remote URL). |
| `style` | No | `info` \| `promo` \| `warning` (unknown → `info`). |
| `startsAt` | No | RFC3339; hide until this time. Empty = already started. |
| `endsAt` | No | RFC3339; hide after this time. Empty = no end. |

### Scheduling example

```json
"startsAt": "2026-08-10T00:00:00Z",
"endsAt": "2026-08-31T23:59:59Z"
```

## Dismissal behaviour

- Closing the card stores `id` in frontend `localStorage` under `dismissedBannerIds` (last 50 kept).
- Dismissed ids never reappear for that install, even if you edit the message.
- To show something again to everyone who already dismissed an old banner: **use a new `id`**.
- Clearing localStorage (or that key) is the only local reset — not something users normally do.

## Ordering and “which banner do users see?”

`useRemoteBanner` picks the **first** list item where:

1. `enabled === true`
2. `id` is non-empty
3. `startsAt` / `endsAt` allow “now” (Go also filters this in `Fetch`)
4. `id` is not in `dismissedBannerIds`

So: put the current campaign first. Leave older entries enabled further down as a fallback for users who already dismissed the newer ones — or set `"enabled": false` on retired campaigns.

## Turning a campaign off

- Set `"enabled": false`, or
- Set `endsAt` to a past timestamp, or
- Remove the object from the array

Then push to `main`. No app release required.

## What is *not* a remote banner

| Mechanism | Purpose |
|-----------|---------|
| Remote banner (`AdOverlay`) | Marketing / docs / one-off announcements from `banner.json` |
| `UpdateBanner` | New GitHub Release detected by the updater |
| Toast in `App.jsx` | Local, ephemeral UI feedback for the current session |

Do not use `banner.json` for critical security alerts that must interrupt every user immediately — there is no push channel, only fetch-on-launch.

## Code map

| Piece | Path |
|-------|------|
| Config file | `social/banner.json` |
| Fetch + schedule filter | `internal/banner/banner.go` |
| Wails binding | `GetRemoteBanner` in `app.go` |
| Frontend fetch / dismiss | `frontend/src/hooks/useRemoteBanner.js` |
| UI card | `frontend/src/components/AdOverlay.jsx` |
| Styles | `frontend/src/styles/_banners.scss` (shared banner area) |
| Analytics | `banner_shown` / `banner_clicked` / `banner_dismissed` with `banner_id` |

## Checklist before pushing

- [ ] New unique `id`
- [ ] `enabled: true`
- [ ] Entry is first (or intentionally ordered) among active banners
- [ ] `linkUrl` is absolute `https://…`
- [ ] Valid JSON (array or single object)
- [ ] Committed and pushed to **`main`**
- [ ] Verified live file:  
      https://raw.githubusercontent.com/canaryGrapher/JumpStart/main/social/banner.json  
      (CDN may lag a minute; `git show origin/main:social/banner.json` is authoritative)
- [ ] Restarted a desktop build to confirm the card appears

## Why a change might not show

1. **Not on `main`** — PR or local-only edit (most common).
2. **Same `id` already dismissed** on that machine.
3. **Another banner earlier in the list** still active and not dismissed — users only see one at a time.
4. **App not restarted** since the fetch.
5. **Offline / fetch failure** — hook fails quietly; no card.
6. **`enabled: false`** or outside `startsAt`/`endsAt`.
7. **Stale CDN** — wait briefly or confirm via the GitHub API / `git show`.
