# JumpStart Analytics Plan

Status: proposal, not implemented. Scope: remote product analytics across all users, zero recurring cost.

---

## 1. Vendor decision

**Recommendation: PostHog Cloud, free tier, ingested from the Go backend.**

| | PostHog Cloud (free) | Aptabase Cloud (free) | Self-hosted (PostHog/Umami) |
|---|---|---|---|
| Cost | $0 up to 1M events/mo | $0 up to 1M events/mo | $0 software, ~$10-20/mo VPS |
| Custom events + properties | Yes, unlimited | Yes, but props capped and shallow | Yes |
| Funnels, retention, cohorts | Yes | Basic charts only | Yes |
| Anonymous-user support | Yes, `distinct_id` you control | Yes, by design | Yes |
| Go SDK | `posthog-go`, official | HTTP only | `posthog-go` |
| Ops burden | None | None | You own uptime, backups, upgrades |

Reasoning: the questions worth asking about JumpStart are funnel and retention questions ("do people who import a config ever start a process?", "does the AI chat bring anyone back on day 7?"). Aptabase is beautifully suited to desktop apps but deliberately does not model users, so it cannot answer those. Self-hosting is not free once a VPS is involved.

Trade-off to accept: PostHog free tier resets monthly and analytics stop when the cap is hit. Section 5 keeps volume far under the cap.

### Why the Go side, not the React side

Wails ships a webview, not a browser. Frontend ingestion means: CSP exemptions, a JS bundle you must keep updated at release cadence, no delivery when the user is offline, and no visibility into anything that happens in `Startup`/`Shutdown`. The Go process already sees every user action because every action crosses the Wails binding boundary into `app.go`. Instrument there.

---

## 2. Consent and privacy

This is a desktop app on other people's machines. Non-negotiables:

- **Opt-in, not opt-out.** First launch shows a one-time dialog. Default off. Nothing is sent before the user chooses.
- **Persist the choice** in `~/.jumpstart/config.json` as `analytics.enabled` plus `analytics.decidedAt`. A togglable switch lives in Preferences.
- **Anonymous install ID.** A random UUIDv4 generated once, stored in `~/.jumpstart/install_id`. Never a machine ID, MAC address, hostname, or email.
- **Never send:** absolute filesystem paths, project names, process names, repo URLs, branch names, commit messages, env var names or values, script bodies, chat text, prompts, model responses, log lines, git tokens.
- **Send derived shapes instead.** Not `/Users/yash/code/acme-api` but `path_depth: 4`. Not `feat: add billing` but `message_length: 18`. Not the project name but a per-install HMAC of it (`project_ref`) so you can count distinct projects without learning any.

Add a `docs/privacy.md` and link it from the consent dialog. This is what makes the opt-in honest rather than decorative.

---

## 3. Global properties

Attached to every event by the emitter, never by callers.

| Property | Example | Why |
|---|---|---|
| `app_version` | `0.4.2` | Regression and adoption tracking. From `version.go`. |
| `update_channel` | `stable` \| `beta` | Beta cohort is your early-warning system. |
| `os` | `darwin` \| `windows` \| `linux` | |
| `os_version` | `15.3` | Drop-support decisions. |
| `arch` | `arm64` | |
| `locale` | `en-US` | |
| `install_age_days` | `12` | Splits new users from veterans in every chart. |
| `session_id` | UUID, regenerated per launch | Groups a run without persistent tracking. |
| `is_first_session` | `true` | Onboarding funnel. |

`distinct_id` = the install ID.

---

## 4. Event taxonomy

Naming: `object_verb_past_tense`, snake_case. Properties snake_case. This ordering makes PostHog's alphabetical event list group by feature area for free.

### 4.1 Lifecycle

| Event | Properties | Question it answers |
|---|---|---|
| `app_launched` | `cold_start_ms`, `project_count`, `favorite_count`, `theme` | DAU/WAU/MAU. How big do real installs get? |
| `app_closed` | `session_duration_s`, `events_this_session` | Session length distribution. |
| `app_crashed` | `panic_type`, `goroutine` | Stability by version. Fires from a `recover()` in `main.go`. |
| `consent_decided` | `granted`, `seconds_to_decide` | Opt-in rate. Send once, at decision time. |

### 4.2 Onboarding funnel

The single most important sequence. Define it in PostHog as a funnel in this order:

`app_launched` (first) → `project_created` → `process_added` → `process_started` → `process_started` (second distinct day)

| Event | Properties |
|---|---|
| `project_created` | `source` (`manual` \| `import` \| `detect`), `process_count`, `path_depth` |
| `process_added` | `detected` (bool), `runtime` (`node` \| `go` \| `python` \| `docker` \| `other`), `has_env_file` |
| `config_imported` | `source` (`file` \| `paste` \| `picker`), `project_count`, `succeeded`, `failure_reason` |
| `processes_detected` | `count`, `runtimes[]`, `accepted_count` |

`processes_detected` with `accepted_count: 0` is the highest-signal failure event in the app: auto-detection ran and the user rejected everything it found.

### 4.3 Core loop — process management

| Event | Properties |
|---|---|
| `process_started` | `runtime`, `trigger` (`single` \| `start_all`), `startup_ms`, `succeeded`, `failure_reason` |
| `process_stopped` | `trigger` (`single` \| `stop_all` \| `app_shutdown`), `uptime_s`, `exit_code` |
| `process_crashed` | `runtime`, `exit_code`, `uptime_s` |
| `deps_installed` | `manager` (`npm` \| `pnpm` \| `yarn` \| `go` \| `pip`), `duration_ms`, `succeeded` |
| `logs_opened` | `line_count`, `source` (`process` \| `script` \| `test`) |
| `ports_viewed` | `port_count`, `conflict_count` |

`failure_reason` must be a bounded enum (`command_not_found`, `port_in_use`, `permission_denied`, `cwd_missing`, `nonzero_exit`, `other`) — never a raw error string, which would leak paths and explode cardinality.

### 4.4 Feature adoption

Every panel in the app gets one `*_opened` event. Cheap, and it tells you what to delete.

| Event | Properties |
|---|---|
| `panel_opened` | `panel` (`git` \| `docker` \| `kanban` \| `scripts` \| `tests` \| `deps` \| `ports` \| `env` \| `roadmap` \| `chat` \| `preferences`) |
| `git_action_performed` | `action` (`commit` \| `push` \| `pull` \| `fetch` \| `branch_create` \| `checkout` \| `diff` \| `stash` \| `init` \| `remote_add`), `succeeded`, `failure_reason` |
| `git_token_saved` | `provider` (`github` \| `gitlab`) |
| `docker_action_performed` | `action` (`compose_up` \| `compose_down` \| `container_start` \| `container_stop` \| `container_remove`), `container_count`, `succeeded` |
| `script_run` | `duration_ms`, `succeeded`, `stopped_early` |
| `tests_run` | `framework` (`jest` \| `vitest` \| `go_test` \| `pytest` \| `custom`), `duration_ms`, `succeeded`, `detected` |
| `release_created` | `provider`, `has_assets`, `prerelease`, `succeeded` |
| `env_file_edited` | `var_count`, `added`, `removed` |
| `theme_changed` | `to` (`light` \| `dark` \| `system`) |

Use one `git_action_performed` with an `action` property rather than fourteen separate events. Fourteen events means fourteen charts to maintain and no way to ask "does this user use git at all".

### 4.5 Kanban and planning

| Event | Properties |
|---|---|
| `task_created` | `column`, `has_sprint`, `ai_enriched` |
| `task_moved` | `from_column`, `to_column`, `age_hours` |
| `sprint_created` | `duration_days`, `task_count` |
| `roadmap_opened` | `item_count` |

`task_moved` with `age_hours` gives you cycle time, which is the only number that proves the kanban is used for real work rather than tried once.

### 4.6 AI features

Your most expensive surface to maintain, so instrument it hardest. **Zero prompt or response content.**

| Event | Properties |
|---|---|
| `ollama_detected` | `reachable`, `model_count` |
| `ai_model_selected` | `model_family` (`llama` \| `qwen` \| `mistral` \| `gemma` \| `other`), `param_size` (`7b`) |
| `ai_task_enriched` | `kind`, `latency_ms`, `succeeded`, `accepted` |
| `ai_chat_message_sent` | `session_message_count`, `latency_ms`, `used_code_context`, `context_chunk_count`, `succeeded` |
| `ai_description_generated` | `latency_ms`, `succeeded`, `accepted` |
| `code_context_built` | `file_count`, `chunk_count`, `duration_ms`, `succeeded` |
| `code_context_searched` | `hit_count`, `top_score_bucket` (`high` \| `medium` \| `low` \| `none`) |

`accepted` is the payoff property. Generation counts tell you the feature runs; acceptance rate tells you it works. Track it by wiring the "insert"/"keep" button, not the generation call.

### 4.7 Updates

| Event | Properties |
|---|---|
| `update_checked` | `channel`, `update_available`, `current_version`, `latest_version` |
| `update_installed` | `from_version`, `to_version`, `channel`, `duration_ms`, `succeeded` |
| `update_dismissed` | `latest_version`, `dismiss_count` |
| `banner_shown` / `banner_clicked` / `banner_dismissed` | `banner_id` |

Version fragmentation is the thing to watch: if `app_version` distribution has a long tail weeks after a release, your updater is failing silently and every other metric is polluted.

---

## 5. Volume budget

Free tier is 1M events/month. Sanity check at 500 MAU:

| Event class | Per user/day | Monthly at 500 users |
|---|---|---|
| Lifecycle (launch, close) | 3 | 45k |
| Process start/stop | 12 | 180k |
| Panel opens | 8 | 120k |
| Git/docker/scripts/tests | 6 | 90k |
| AI | 4 | 60k |
| **Total** | **~33** | **~495k** |

Roughly half the cap at 500 users, so ~1,000 MAU before it's a problem. Guardrails:

1. **Never instrument polling.** `GetUsage`, `GetStatus`, `GetLogs`, and the ports refresh are called on timers. Instrumenting any of them is an instant cap breach.
2. **Debounce `panel_opened`** — one per panel per session, not per click. A `map[string]bool` on the session covers it.
3. **Sample nothing else.** Sampling breaks funnel math and you lose more than you save.
4. Set a PostHog billing alert at 700k.

---

## 6. Implementation shape

New package `internal/analytics`, kept small per the repo's file-size convention:

```
internal/analytics/
  client.go     // PostHog client init, consent gate, graceful no-op when disabled
  identity.go   // install ID, session ID, project_ref HMAC
  props.go      // global property assembly (version, os, channel, install age)
  emit.go       // Track(name string, props map[string]any) — the only exported call site API
  queue.go      // disk-backed buffer at ~/.jumpstart/analytics_queue.ndjson, flush on launch
  redact.go     // failure_reason enum mapping, path→depth, name→ref
  emit_test.go  // asserts no raw path/name/token can escape
```

Design rules:

- `Track` never returns an error and never blocks. Fire onto a buffered channel; drop on full.
- With consent off, `Track` returns immediately. No client, no goroutine, no file.
- **Offline buffering matters more here than on the web.** Devs work on planes. Append to the ndjson queue when the send fails, flush at next launch, cap the file at 5k events and drop oldest.
- Flush on `Shutdown` with a 2s timeout so `app_closed` actually lands.
- One `Track` call per Wails binding, at the top of the method, using the `defer`-with-named-result pattern so `succeeded` and duration come from the real outcome rather than the attempt.

`redact_test.go` is the load-bearing test: a table of realistic paths, repo URLs, and commit messages asserting none survive into an emitted payload.

---

## 7. Dashboards to build once data lands

1. **Health** — DAU/WAU/MAU, sessions per user, `app_crashed` rate by `app_version`.
2. **Activation funnel** — the section 4.2 sequence, broken down by `os`.
3. **Feature adoption matrix** — % of MAU firing each `panel_opened` variant. Anything under 5% after a month is a deletion candidate.
4. **Reliability** — `process_started` success rate by `runtime`, top `failure_reason` values. This is your bug backlog, ranked by real frequency.
5. **AI value** — `accepted` rate for enrich and description, chat messages per session, `used_code_context` correlation with acceptance.
6. **Version fragmentation** — `app_version` distribution over time.

---

## 8. Rollout

1. Consent dialog, Preferences toggle, `docs/privacy.md`. Ship alone, one release ahead of any tracking, so consent is collected before there is anything to send.
2. `internal/analytics` package with `Track` as a no-op, plus redaction tests.
3. Lifecycle + activation events only (4.1, 4.2). Verify shapes in PostHog live view against your own install.
4. Core loop and feature adoption (4.3, 4.4, 4.5).
5. AI and updates (4.6, 4.7).
6. Dashboards, then a monthly review.

Ship 1 and 2 before writing a single `Track` call site. Retrofitting consent onto shipped tracking is the one mistake here that cannot be undone.

---

## Sources

- [PostHog pricing 2026](https://posthog.com/pricing)
- [Aptabase](https://aptabase.com/)
- [aptabase/aptabase on GitHub](https://github.com/aptabase/aptabase)
