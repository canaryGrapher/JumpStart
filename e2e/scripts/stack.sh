#!/bin/bash
# Start/stop the e2e stack: Vite dev server + `wails dev` (real Go backend),
# with HOME pointed at a throwaway directory and OS launchers (open,
# qlmanage, xdg-open) replaced by stubs that only log their arguments.
#
#   scripts/stack.sh start | stop | status
set -u
HERE="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(cd "$HERE/.." && pwd)"
E2E_HOME="${E2E_HOME:-/tmp/js-e2e-home}"
SHIM_DIR="${SHIM_DIR:-/tmp/js-e2e-bin}"
LOG_DIR=/tmp
export PATH="$PATH:$HOME/go/bin"
# Pin Go's caches to the real ones before HOME is redirected, otherwise the
# toolchain re-downloads every module into the throwaway HOME.
GO_ENV="GOPATH=$(go env GOPATH) GOMODCACHE=$(go env GOMODCACHE) GOCACHE=$(go env GOCACHE) GOFLAGS=-modcacherw"

stop() {
  pkill -f "wails dev" 2>/dev/null
  pkill -f "vite --port 5173" 2>/dev/null
  pkill -f "devdeck" 2>/dev/null
  # Only processes started from this stack: the dev binary lives under the repo build dir.
  pkill -f "$REPO/build/bin/.*dev" 2>/dev/null
  sleep 1
}

case "${1:-}" in
  stop) stop; echo stopped ;;
  status)
    curl -s -o /dev/null -w "ui     %{http_code}\n" http://localhost:34115/
    curl -s -o /dev/null -w "mcp    %{http_code} (401 = up, auth required)\n" http://127.0.0.1:18788/mcp ;;
  start)
    stop
    chmod -R u+w "$E2E_HOME" 2>/dev/null; rm -rf "$E2E_HOME" "$SHIM_DIR"; mkdir -p "$SHIM_DIR"
    if [ -n "${E2E_COPY_REAL:-}" ]; then
      # Read-only smoke test of real data: work on a COPY, with MCP switched off
      # so it cannot collide with the real app's server.
      mkdir -p "$E2E_HOME/.jumpstart"
      cp "$HOME/.jumpstart/config.json" "$E2E_HOME/.jumpstart/config.json"
      echo '{"enabled": false, "port": 18788, "token": "unused-copy-token-000000"}' > "$E2E_HOME/.jumpstart/mcp.json"
    else
      node "$HERE/scripts/seed.mjs" "$E2E_HOME" >/dev/null
    fi
    for cmd in open qlmanage xdg-open explorer; do
      printf '#!/bin/sh\necho "%s $*" >> "%s/js-e2e-launch.log"\n' "$cmd" "$LOG_DIR" > "$SHIM_DIR/$cmd"
      chmod +x "$SHIM_DIR/$cmd"
    done
    : > "$LOG_DIR/js-e2e-launch.log"
    # Fully detach children (own stdin/stdout/stderr) so callers piping this
    # script's output are not kept waiting by the long-running servers.
    (cd "$REPO/frontend" && exec nohup npx vite --port 5173 --strictPort) < /dev/null > "$LOG_DIR/js-vite.log" 2>&1 &
    for i in $(seq 1 30); do curl -s -o /dev/null http://localhost:5173/ && break; sleep 1; done
    (cd "$REPO" && exec env $GO_ENV HOME="$E2E_HOME" PATH="$SHIM_DIR:$PATH" nohup wails dev -s -m -nosyncgomod -nogorebuild -noreload \
        -frontenddevserverurl http://localhost:5173 -loglevel Info) < /dev/null > "$LOG_DIR/js-wails-dev.log" 2>&1 &
    for i in $(seq 1 90); do
      curl -s -o /dev/null http://localhost:34115/ && { [ -n "${E2E_COPY_REAL:-}" ] || curl -s -o /dev/null http://127.0.0.1:18788/mcp; } && { echo "stack up after ${i}s"; exit 0; }
      sleep 1
    done
    echo "stack failed to start; see $LOG_DIR/js-wails-dev.log" >&2; exit 1 ;;
  *) echo "usage: $0 start|stop|status" >&2; exit 2 ;;
esac
