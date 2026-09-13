---
name: trace
description: Investigate Pydantic Logfire traces - one-command save plus peek, local SQLite search, API SQL, and debug production issues
user_invocable: true
---

# lfsnag: Logfire Trace Investigation

Use `lfsnag` (`./bin/lfsnag` or PATH). Token: `~/.config/lfsnag/config.json`, `LOGFIRE_READ_TOKEN`, or `--token`.

## The three commands

```bash
# 1. SAVE + PEEK: pass a traceId or a full Logfire URL.
#    Streams all records into SQLite (RAM-bounded), then prints a small summary
#    that includes the "db" path for step 3. URL org/project auto-picks the env profile.
lfsnag 'https://logfire-us.pydantic.dev/org/proj?traceId=<traceId>'
lfsnag -e stage <traceId>                      # -e overrides URL/auto selection
lfsnag --save /tmp/t.sqlite -e stage <traceId> # custom path instead of cache dir

# 2. API SQL (DataFusion) — cross-trace queries, no local file
lfsnag -e stage --sql "SELECT span_name, count() cnt FROM records WHERE trace_id = '<traceId>' GROUP BY span_name ORDER BY cnt DESC"

# 3. LOCAL SQL (SQLite) — search the saved file offline; no token, no rate limit
lfsnag --db ~/.cache/lfsnag/<traceId>.sqlite --peek
lfsnag --db ~/.cache/lfsnag/<traceId>.sqlite --sql "SELECT span_name FROM records WHERE is_exception = 1"
```

Rules the CLI enforces:

- `--peek` works only with `--db`. A bare traceId already peeks.
- `--db` needs `--sql` or `--peek`. It rejects a traceId and `--save`.
- `--sql` rejects `--peek`, `--save`, and a traceId.

Cache: fetches write `<UserCacheDir>/lfsnag/<traceId>.sqlite` (XDG_CACHE_HOME respected; `--save F` overrides). Re-fetch replaces the file atomically. Clear with `rm -rf -- "${XDG_CACHE_HOME:-$HOME/.cache}/lfsnag"` when no lfsnag fetch/query is running — deleting a DB mid-fetch silently loses that fetch.

## Reading the peek

```json
{"db": "…", "n": 107, "n_roots": 1, "n_exceptions": 4, "n_slow": 19,
 "roots": […], "exceptions": […], "slow": […], "top": […]}
```

- `n_*` are **exact totals**; arrays are capped samples (20).
- `exceptions[].message` truncated to 200 chars; get the full text from `--db --sql`.
- `path` on exceptions/slow = ancestry `root › … › span` (capped at 8 hops) — read the hot chain directly.
- `top` = most frequent `span_name` with `max_dur`.
- Slim `-f` fetches lack peek columns: output is `{"db","n","note"}` only — no `n_*` keys means unknown, not zero. Fetch without `-f` for a full peek.

## SQL tips

**API `--sql`** runs DataFusion (postgres-ish): `attributes->>'key'`. Prefer `span_name` over `message`. API `limit` is 10000; SQL `LIMIT` alone does not lift the cap.

```sql
SELECT count(*) n FROM records WHERE trace_id = '<traceId>'
SELECT span_name, count() cnt, max(duration) max_dur FROM records WHERE trace_id = '<traceId>' GROUP BY span_name ORDER BY cnt DESC
SELECT span_name, exception_type, exception_message FROM records WHERE trace_id = '<traceId>' AND is_exception
SELECT ... WHERE level >= 'error'          -- errors without exception
SELECT ... WHERE parent_span_id IS NULL    -- roots
```

**Local `--db --sql`** runs SQLite. JSON via `json_extract`; Logfire keys contain dots, so quote the path. Bools are `0/1`.

```sql
SELECT count(*) n FROM records
SELECT span_name, exception_type, exception_message FROM records WHERE is_exception = 1
SELECT span_name, duration FROM records WHERE duration > 2 ORDER BY duration DESC
SELECT span_name FROM records WHERE attributes LIKE '%gen_ai%'   -- cheap key probe
SELECT span_name, json_extract(attributes, '$."gen_ai.request.model"') model FROM records WHERE json_extract(attributes, '$."gen_ai.request.model"') IS NOT NULL
SELECT sum(json_extract(attributes, '$."gen_ai.aggregated_usage.input_tokens"')) FROM records
SELECT span_name, message, attributes FROM records WHERE span_id = '<spanId>'
```

`jq`: `lfsnag -c --db F --peek | jq '.exceptions'`

## Available fields

- `start_timestamp` - When the span/log was created (UTC)
- `end_timestamp` - When the span/log completed (UTC)
- `duration` - Elapsed time in seconds (NULL for logs)
- `trace_id` - Trace identifier (32 hex chars)
- `span_id` - Span identifier (16 hex chars)
- `parent_span_id` - Parent span reference (NULL for root)
- `kind` - `span`, `log`, `span_event`, or `pending_span`
- `span_name` - Short name shared by similar records
- `message` - Human-readable description
- `level` - Severity level
- `is_exception` - Whether an exception was recorded
- `exception_type` - Exception class name
- `exception_message` - Exception message
- `exception_stacktrace` - Formatted traceback
- `attributes` - Arbitrary structured data (JSON)
- `tags` - Grouping labels
- `otel_status_code` - Span status (UNSET, OK, ERROR)
- `otel_status_message` - Span error description
- `service_name` - Service/application name
- `deployment_environment` - Environment (production, staging)
- `http_response_status_code` - HTTP status code
- `http_method` - HTTP method
- `http_route` - HTTP route pattern
- `url_full` - Complete URL
- `url_path` - URL path
- `service_version` - Service version
- `service_instance_id` - Service instance identifier
- `service_namespace` - Service namespace
- `process_pid` - Process ID
- `url_query` - URL query string
- `log_body` - Body of OpenTelemetry log records
- `otel_events` - Span events (JSON)
- `otel_links` - Span links (JSON)
- `otel_resource_attributes` - Resource metadata (JSON)
- `otel_scope_name` - Instrumenting library name
- `otel_scope_version` - Instrumenting library version
- `otel_scope_attributes` - Scope metadata (JSON)
- `telemetry_sdk_name` - Telemetry SDK name
- `telemetry_sdk_language` - SDK language
- `telemetry_sdk_version` - SDK version

## CLI flags

- `<traceId | URL>` - Fetch: stream to SQLite (cache path or `--save`), print peek with `db`
- `--db FILE` - Query local SQLite (`--sql` or `--peek`; no token)
- `--sql` - Raw SQL (API without `--db`, local with `--db`)
- `--save FILE` - Override the SQLite path for a fetch
- `-f, --fields` - Columns to save (default all)
- `-e, --env` - Profile (auto-picked from URL org/project; `-e` wins)
- `-c, --compact` - One-line JSON
- `-v, --verbose` - HTTP debug
- `--token` - Override read token
