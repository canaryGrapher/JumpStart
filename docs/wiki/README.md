# Publishing this wiki to GitHub

The markdown in `docs/wiki/` is the source of truth for the JumpStart developer wiki.

## One-time bootstrap

GitHub does not create `JumpStart.wiki.git` until a first page exists:

1. Open https://github.com/canaryGrapher/JumpStart/wiki
2. Click **Create the first page** and save (placeholder text is fine)
3. Run the sync script below

## Sync

```sh
./scripts/publish-wiki.sh
```

This clones the wiki repo (or updates it), copies `docs/wiki/*.md`, commits, and pushes.
