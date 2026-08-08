# Landing Site

Directory: `landing/`.

## Purpose

Marketing site for JumpStart (downloads, FAQ, privacy, contribute). **Not** part of the Wails binary.

## Stack

- Solid.js + Vite + GSAP
- Analytics: GA4, Microsoft Clarity, `@vercel/analytics` (env-gated)
- Hash routing via `router.js` / `pages/*`

## Key files

| Path | Role |
|------|------|
| `src/Home.jsx` | Landing composition |
| `src/components/Hero.jsx`, `Nav.jsx`, `Faq.jsx`, … | Sections |
| `src/pages/Downloads.jsx`, `Privacy.jsx` | Routes |
| `src/analytics.js`, `engagement.js` | Site analytics / scroll / CTAs |
| `src/downloads.js`, `releases.js` | GitHub release asset helpers |

## Configuration

Copy `landing/.env.example` → `.env`:

```sh
VITE_GA_ID=G-XXXXXXXXXX
VITE_CLARITY_ID=xxxxxxxxxx
```

Empty IDs disable those collectors.

## Relationship to the desktop app

- Separate product analytics (never joined with PostHog desktop data)
- Privacy page should stay consistent with `docs/privacy.md` / in-app copy
- Download links should track the GitHub Releases assets produced by CI

## Developing

```sh
cd landing
npm install
npm run dev
```
