---
name: trace
description: Investigate Pydantic Logfire traces - fetch by traceId or URL, run SQL queries, filter fields, and debug production issues
user_invocable: true
---

# lfsnag: Logfire Trace Investigation

You are investigating a Pydantic Logfire trace using the `lfsnag` CLI tool.

## Prerequisites

- `lfsnag` must be installed and on PATH (or available at `./bin/lfsnag`)
- A Logfire read token must be configured via `~/.config/lfsnag/config.json`, `LOGFIRE_READ_TOKEN` env var, or `--token` flag

## How to use

### Fetch a trace by ID

```bash
lfsnag <traceId>
```

traceId is a 32-character hex string (e.g., `019d05d5c291ce65f49caad9bf2ebbdc`).

### Fetch a trace from a Logfire URL

Paste the full URL directly — the traceId is extracted automatically:

```bash
lfsnag 'https://logfire-us.pydantic.dev/org/proj?traceId=019d05d5c291ce65f49caad9bf2ebbdc&spanId=...'
```

### Select specific fields

Use `-f` to return only the fields you need:

```bash
lfsnag -f span_name,duration,is_exception <traceId>
```

### Compact output for piping

Use `-c` for single-line JSON suitable for piping to `jq`:

```bash
lfsnag -c <traceId> | jq '[.[] | select(.is_exception == true)]'
```

### Use an environment profile

```bash
lfsnag -e prod <traceId>
lfsnag -e stage <traceId>
```

Profiles are defined in `~/.config/lfsnag/config.json`.

### Raw SQL queries

Use `--sql` to query the Logfire `records` table directly:

```bash
lfsnag --sql "SELECT span_name, duration FROM records WHERE trace_id = '<traceId>' AND is_exception = true"
```

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

## Common SQL patterns

Find exceptions in a trace:
```sql
SELECT span_name, exception_type, exception_message, exception_stacktrace
FROM records
WHERE trace_id = '<traceId>' AND is_exception = true
```

Find slow spans:
```sql
SELECT span_name, duration, message
FROM records
WHERE trace_id = '<traceId>' AND duration > 1.0
ORDER BY duration DESC
```

Trace timeline:
```sql
SELECT span_name, start_timestamp, end_timestamp, duration, kind
FROM records
WHERE trace_id = '<traceId>'
ORDER BY start_timestamp
```

Top spans by count:
```sql
SELECT span_name, count(*) as cnt
FROM records
GROUP BY span_name
ORDER BY cnt DESC
LIMIT 10
```

HTTP errors:
```sql
SELECT span_name, http_response_status_code, url_path, message
FROM records
WHERE trace_id = '<traceId>' AND http_response_status_code >= 400
```

## CLI flags reference

- `--compact`, `-c` - Compact single-line JSON output
- `--verbose`, `-v` - Show HTTP request/response details
- `--env`, `-e` - Environment profile name
- `--fields`, `-f` - Comma-separated fields to select
- `--sql` - Raw SQL query (mutually exclusive with traceId)
- `--token` - Override read token

## Investigation workflow

1. Start by fetching the full trace to understand its shape
2. If the trace is large, narrow down with `-f` to key fields: `span_name,duration,is_exception,message`
3. Look for exceptions: filter with `--sql` using `is_exception = true`
4. Check for slow spans: filter with `--sql` using `duration > N`
5. Examine specific spans in detail using their `span_id`
6. Use compact mode (`-c`) with `jq` for programmatic analysis
