#!/usr/bin/env bash
# Create JumpStart product analytics dashboards in PostHog (idempotent by name).
#
# Required env:
#   POSTHOG_PERSONAL_API_KEY  — personal key with dashboard:write + insight:write
#   POSTHOG_PROJECT_ID        — numeric project id
#
# Optional:
#   POSTHOG_HOST              — management host (default https://us.posthog.com)
#
# Usage:
#   export POSTHOG_PERSONAL_API_KEY=phx_...
#   export POSTHOG_PROJECT_ID=12345
#   ./scripts/setup-posthog-dashboards.sh
#
# Re-run skips dashboards that already exist with the same name.
# To recreate, delete the dashboard in PostHog first.
set -euo pipefail

HOST="${POSTHOG_HOST:-https://us.posthog.com}"
HOST="${HOST%/}"
KEY="${POSTHOG_PERSONAL_API_KEY:-}"
PROJECT="${POSTHOG_PROJECT_ID:-}"

if [[ -z "$KEY" || -z "$PROJECT" ]]; then
  echo "Set POSTHOG_PERSONAL_API_KEY and POSTHOG_PROJECT_ID" >&2
  exit 1
fi

auth_hdr=(-H "Authorization: Bearer ${KEY}" -H "Content-Type: application/json")

api() {
  local method="$1" path="$2"
  shift 2
  curl -sS -X "$method" "${auth_hdr[@]}" "${HOST}${path}" "$@"
}

echo "Listing dashboards for project ${PROJECT}…"
existing="$(api GET "/api/projects/${PROJECT}/dashboards/?limit=100")"

has_dashboard() {
  local name="$1"
  echo "$existing" | NAME="$name" python3 -c '
import json, os, sys
name = os.environ["NAME"]
data = json.load(sys.stdin)
results = data.get("results", data if isinstance(data, list) else [])
print("yes" if any(d.get("name") == name for d in results) else "no")
'
}

create_trends() {
  local dash_id="$1" name="$2" event="$3"
  local body
  body="$(DASH_ID="$dash_id" NAME="$name" EVENT="$event" python3 - <<'PY'
import json, os
print(json.dumps({
  "name": os.environ["NAME"],
  "dashboards": [int(os.environ["DASH_ID"])],
  "saved": True,
  "query": {
    "kind": "InsightVizNode",
    "source": {
      "kind": "TrendsQuery",
      "series": [{
        "kind": "EventsNode",
        "event": os.environ["EVENT"],
        "name": os.environ["EVENT"],
      }],
      "dateRange": {"date_from": "-30d"},
      "interval": "day",
    },
  },
}))
PY
)"
  api POST "/api/projects/${PROJECT}/insights/" -d "$body" >/dev/null
  echo "  + insight: $name"
}

create_funnel() {
  local dash_id="$1"
  local body
  body="$(DASH_ID="$dash_id" python3 - <<'PY'
import json, os
print(json.dumps({
  "name": "Activation funnel",
  "dashboards": [int(os.environ["DASH_ID"])],
  "saved": True,
  "query": {
    "kind": "InsightVizNode",
    "source": {
      "kind": "FunnelsQuery",
      "series": [
        {"kind": "EventsNode", "event": "app_launched", "name": "app_launched"},
        {"kind": "EventsNode", "event": "project_created", "name": "project_created"},
        {"kind": "EventsNode", "event": "process_added", "name": "process_added"},
        {"kind": "EventsNode", "event": "process_started", "name": "process_started"},
      ],
      "dateRange": {"date_from": "-30d"},
    },
  },
}))
PY
)"
  api POST "/api/projects/${PROJECT}/insights/" -d "$body" >/dev/null
  echo "  + insight: Activation funnel"
}

create_insights_for() {
  local name="$1" id="$2"
  case "$name" in
    "JumpStart — Health")
      create_trends "$id" "App launches (DAU proxy)" "app_launched"
      create_trends "$id" "App closed" "app_closed"
      create_trends "$id" "App crashed" "app_crashed"
      ;;
    "JumpStart — Activation")
      create_funnel "$id"
      ;;
    "JumpStart — Feature adoption")
      create_trends "$id" "Panel opened" "panel_opened"
      create_trends "$id" "Git actions" "git_action_performed"
      create_trends "$id" "Docker actions" "docker_action_performed"
      ;;
    "JumpStart — Reliability")
      create_trends "$id" "Process started" "process_started"
      create_trends "$id" "Process crashed" "process_crashed"
      ;;
    "JumpStart — AI value")
      create_trends "$id" "AI chat messages" "ai_chat_message_sent"
      create_trends "$id" "AI suggestions accepted" "ai_suggestion_accepted"
      create_trends "$id" "Task enrich" "ai_task_enriched"
      ;;
    "JumpStart — Version fragmentation")
      create_trends "$id" "Launches by version" "app_launched"
      ;;
  esac
}

create_dashboard() {
  local name="$1" desc="$2"
  if [[ "$(has_dashboard "$name")" == "yes" ]]; then
    echo "Skip (exists): $name"
    return 0
  fi
  local body resp id
  body="$(NAME="$name" DESC="$desc" python3 -c 'import json,os; print(json.dumps({"name":os.environ["NAME"],"description":os.environ["DESC"],"pinned":True}))')"
  resp="$(api POST "/api/projects/${PROJECT}/dashboards/" -d "$body")"
  id="$(echo "$resp" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("id",""))')"
  if [[ -z "$id" ]]; then
    echo "Failed to create $name: $resp" >&2
    return 1
  fi
  echo "Created: $name → ${HOST}/dashboard/${id}"
  create_insights_for "$name" "$id"
}

create_dashboard "JumpStart — Health" "DAU/WAU/MAU proxies, sessions, crashes"
create_dashboard "JumpStart — Activation" "Onboarding funnel"
create_dashboard "JumpStart — Feature adoption" "Panel and feature reach"
create_dashboard "JumpStart — Reliability" "Process success and crashes"
create_dashboard "JumpStart — AI value" "Chat, enrich, acceptance"
create_dashboard "JumpStart — Version fragmentation" "app_version distribution via launches"

echo
echo "Done. Set a PostHog billing alert near 700k events/month in Project settings."
echo "Personal API keys are never stored in this repo or in GitHub Actions."
