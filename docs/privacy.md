# Privacy: what JumpStart collects

JumpStart reports anonymous product usage so we can see which features are
used and where they fail. This document is the complete, honest description
of what that means. If anything here turns out to be inaccurate, that is a
bug — please file it.

## The short version

- **Nothing identifies you.** No account, no email, no machine ID, no
  hostname, no IP-derived profile.
- **Nothing describes your code.** No file paths, project names, process
  names, commands, repository URLs, branch names, commit messages,
  environment variables, script contents, log lines, or AI chat text.
- **You can turn it off** in Settings → Privacy. Turning it off stops
  collection immediately and deletes anything still buffered on disk.
  You can also lower the detail level or disable individual categories
  while leaving analytics on.
- **It is on by default.** We think that is a fair trade for a free tool,
  and the switches are one click away.

## Where to change it

Settings → Privacy → *Share anonymous usage data*.

Under that switch you can choose a **detail level** (Full, Balanced, or
Minimal) and turn individual categories on or off (lifecycle, onboarding,
processes, Git & Docker, kanban, AI, updates, UI panels). Everything is on
by default (Full). Turning the master switch off stops collection
immediately and deletes anything still buffered on disk.

The choice is stored in `~/.jumpstart/settings.json`:

```json
{
  "analytics": {
    "enabled": true,
    "decidedAt": 1767225600000,
    "detailLevel": "full",
    "categories": {
      "lifecycle": true,
      "onboarding": true,
      "processes": true,
      "git_docker": true,
      "kanban": true,
      "ai": true,
      "updates": true,
      "ui_panels": true
    }
  }
}
```

## What is actually sent

Each batch request to Google Analytics 4 includes a **`client_id`**: a random
UUID generated on first launch, stored in `~/.jumpstart/install_id`, and
derived from nothing about you or your machine. GA4 uses this to group one
installation's events over time. It is sent at the request level, not as an
event parameter.

Every event carries the same set of global parameters:

| Parameter | Example | Why |
|---|---|---|
| `app` | `desktop` | Separates the desktop app from any future surfaces. |
| `platform` | `darwin` | Same as `os`; kept for GA4 exploration filters. |
| `app_version` | `1.4.2` | Regression and adoption tracking. |
| `update_channel` | `stable` | Beta users are the early warning for regressions. |
| `os` / `os_version` / `arch` | `darwin` / `15.3` / `arm64` | Decides which platforms to keep supporting. |
| `locale` | `en-US` | Localisation decisions. |
| `install_age_days` | `12` | Separates new users from long-time ones. |
| `session_id` | a random UUID, new every launch | Groups one run. Not persistent. |
| `is_first_session` | `true` | Onboarding funnel. |
| `is_key_event` | `true` | Marks conversion candidates (e.g. `process_started`, `project_created`). Set only on qualifying events. |

Events themselves carry counts, durations, booleans, and values from a
fixed list. A few representative examples:

- `app_launched` — `cold_start_ms`, `project_count`, `process_count`
- `process_started` — `runtime` (one of `node`, `go`, `python`, …),
  `trigger`, `succeeded`, `duration_ms`, `failure_reason`
- `git_action_performed` — `action` (`commit`, `push`, `pull`, …),
  `succeeded`, `failure_reason`. For commits, `message_length` — a number,
  never the message.
- `ai_chat_message_sent` — `model_family` (`llama`, `qwen`, …),
  `message_length`, `latency_ms`, `used_code_context`. Never the message,
  the reply, or the files the retriever looked at.
- `panel_opened` — which panel, once per session.

## What is deliberately never sent

This list is enforced in code, not just in policy. See
`internal/analytics/redact.go` and its tests.

- Absolute or relative filesystem paths. A project's location becomes
  `path_depth: 4` and nothing else.
- Project, process, and script names.
- Shell commands. A command becomes a `runtime` value from a fixed list.
- Git remotes, branch names, and commit messages. A commit reports
  `message_length` only.
- Environment variable names and values.
- Chat messages, AI prompts, model responses, and generated task text.
- Model tags. `qwen2.5-coder:7b` becomes `model_family: qwen` and
  `param_size: 7b`, so a private fine-tune's name cannot leak.
- Access tokens and anything resembling one.
- Raw error strings. Every failure maps onto a fixed set of reasons such as
  `port_in_use`, `command_not_found`, `permission_denied`.

Because a mistake at one call site should not become a privacy incident,
every property passes through a sanitiser before it leaves the process. Any
string containing whitespace, a path separator, an `@`, an `=`, a URL
scheme, or a known credential prefix — or longer than 32 characters — is
replaced with `redacted`.

## How project counts work without project names

Some questions need to distinguish projects without identifying them: "do
people run processes in more than one project?" To answer that, a project's
local ID is hashed with HMAC-SHA256 keyed by your install ID, and the first
12 hex characters become `project_ref`.

The key never leaves your machine, so the same project on two machines
hashes differently, and the value cannot be reversed into anything.

## Where it goes, and when

Events go to [Google Analytics 4](https://analytics.google.com) via the Measurement Protocol. They are batched and sent
at most every 15 seconds.

If the machine is offline or the send fails, events are appended to
`~/.jumpstart/analytics_queue.ndjson` (capped at 5,000 events, oldest
dropped) and retried on the next launch. Turning analytics off deletes that
file.

## Builds with no analytics at all

The GA4 credentials are injected at release build time. A build made without them —
including any build you make yourself from source — sends nothing, and
Settings → Privacy says so.

## The desktop app and the website are separate

The landing site at jumpstart.workvar.com uses Google Analytics, Microsoft Clarity,
and Vercel Analytics to understand how visitors find and use the site. Those
run on the website only. The desktop app ships none of them, and the two
data sets are never joined.

## Everything else the app does with the network

Analytics aside, JumpStart only reaches the network for actions you start:
git fetch/pull/push, publishing a release, checking for updates, fetching
the announcement banner, and talking to your own local Ollama instance. AI
runs locally; nothing you type into chat is sent anywhere but your own
machine.
