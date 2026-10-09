# JumpStart end-to-end tests

Drives the **real Go backend** (`wails dev`) through a browser, with `HOME` pointed at a
throwaway directory, so the user's real `~/.jumpstart` is never touched.

```bash
cd e2e && npm install
./scripts/stack.sh start        # seeds /tmp/js-e2e-home, starts Vite + wails dev (UI :34115, MCP :18788)
npx playwright test             # 83 tests: due dates, filters, quarters, links, attachments, MCP, AI
./scripts/stack.sh stop
```

- `OS launchers (open, qlmanage, xdg-open)` are replaced by stubs that log their arguments to
  `/tmp/js-e2e-launch.log`, so Preview/Open/Reveal and link opening are asserted without opening apps.
- The AI tests use a mock Ollama server and assert exactly what the app sends to the model.
- **Run one suite at a time.** Every test reseeds the same data directory; two concurrent runs corrupt each other.
- `E2E_COPY_REAL=1 ./scripts/stack.sh start` then `npx playwright test -c playwright.real.config.ts`
  is a read-only smoke test of a *copy* of the real data (screenshots in `artifacts/real-data/`).
- Wails' browser dev bridge throws `Cannot read properties of null (reading 'nodes')` from
  `/wails/ipc.js`; the helpers ignore that one error and fail on any other page error.
