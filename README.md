# lfsnag

[![Go Version](https://img.shields.io/github/go-mod/go-version/bigbag/lfsnag)](https://github.com/bigbag/lfsnag)
[![Build](https://img.shields.io/github/actions/workflow/status/bigbag/lfsnag/build.yaml?branch=master)](https://github.com/bigbag/lfsnag/actions/workflows/build.yaml)
[![Release](https://img.shields.io/github/v/release/bigbag/lfsnag)](https://github.com/bigbag/lfsnag/releases/latest)
[![license](https://img.shields.io/github/license/bigbag/lfsnag.svg)](https://github.com/bigbag/lfsnag/blob/master/LICENSE)

A CLI tool to investigate [Pydantic Logfire](https://logfire.pydantic.dev/) traces. One command streams a trace into local SQLite and prints a peek summary. Search the saved data offline with SQL.

## Features

- **Save + peek** - A traceId/URL fetch streams all records into SQLite, then prints a compact summary (roots, top spans, exceptions with ancestry paths, slow spans)
- **Local SQL** - Query saved traces offline with SQLite syntax. Local queries need no token and have no rate limit.
- **API SQL** - Run DataFusion SQL directly against Logfire
- **URL env auto-select** - A Logfire URL's org/project picks the matching environment profile
- **Compact/verbose modes** - Machine-friendly JSON. Show HTTP request/response details.
- **Flexible config** - CLI flags, environment variables, or config file

## Quick Start

```bash
# Build
make build

# Fetch a trace: streams to ~/.cache/lfsnag/<traceId>.sqlite, prints a peek summary
./bin/lfsnag <traceId>
```

## Installation

```bash
# Clone the repository
git clone https://github.com/bigbag/lfsnag.git
cd lfsnag

# Build
make build

# Or install to GOPATH/bin
make install
```

## CLI Flags

- `-c, --compact` - Compact JSON output
- `-v, --verbose` - Show HTTP request/response details
- `-e, --env` - Environment profile (auto-picked from a Logfire URL's org/project)
- `-f, --fields` - Columns to save on fetch (default: all)
- `--save FILE` - SQLite path for a fetch (default: `~/.cache/lfsnag/<traceId>.sqlite`)
- `--db FILE` - Query a local SQLite file (`--sql` or `--peek`)
- `--peek` - Peek summary. Works only with `--db`. A bare traceId already peeks.
- `--sql` - Raw SQL query (Logfire API, or `--db` local file)
- `--token` - Override read token

Mode rules:

- `--db` needs `--sql` or `--peek`. It rejects a traceId and `--save`.
- `--sql` rejects `--peek`, `--save`, and a traceId.
- A bare traceId always saves and peeks.

## Available Fields

- `start_timestamp` - The time when the span/log started (UTC)
- `end_timestamp` - When the span/log completed (UTC)
- `duration` - Elapsed time in seconds (NULL for logs)
- `trace_id` - Trace identifier (32 hex chars)
- `span_id` - Span identifier (16 hex chars)
- `parent_span_id` - Parent span reference (NULL for root spans)
- `kind` - `span`, `log`, `span_event`, or `pending_span`
- `span_name` - Short name shared by similar records
- `message` - Human-readable description
- `level` - Severity level
- `is_exception` - Whether the span holds an exception
- `exception_type` - Exception class name
- `exception_message` - Exception message
- `exception_stacktrace` - Formatted traceback
- `attributes` - Arbitrary structured data (JSON)
- `tags` - Grouping labels
- `otel_status_code` - Span status indicator
- `otel_status_message` - Span error status description
- `otel_events` - Span events (JSON)
- `otel_links` - Span links (JSON)
- `service_name` - Service/application name
- `service_version` - Service version
- `service_instance_id` - Service instance identifier
- `service_namespace` - Service namespace
- `deployment_environment` - Environment (production, staging, etc.)
- `process_pid` - Process ID
- `http_response_status_code` - HTTP status code
- `http_method` - HTTP method
- `http_route` - HTTP route pattern
- `url_full` - Complete URL
- `url_path` - URL path
- `url_query` - URL query string
- `log_body` - Body of OpenTelemetry log records
- `otel_resource_attributes` - Resource metadata (JSON)
- `otel_scope_name` - Instrumenting library name
- `otel_scope_version` - Instrumenting library version
- `otel_scope_attributes` - Scope metadata (JSON)
- `telemetry_sdk_name` - Telemetry SDK name
- `telemetry_sdk_language` - SDK language
- `telemetry_sdk_version` - SDK version

## Configuration

lfsnag resolves configuration in this priority order: **CLI flags > environment variables > config file**.

### Environment Profiles

Path: `~/.config/lfsnag/config.json`

```json
{
  "default": "prod",
  "environments": {
    "prod": {
      "token": "prod-read-token",
      "base_url": "https://logfire-us.pydantic.dev",
      "project": "my-org/prod-mra"
    },
    "stage": {
      "token": "stage-read-token",
      "base_url": "https://logfire-eu.pydantic.dev",
      "project": "my-org/stage-mra"
    }
  }
}
```

Select an environment with `-e`:

```bash
lfsnag -e prod abc123def456789012345678abcdef01
```

Without `-e`, a Logfire URL argument selects the profile whose `project` field matches. Otherwise lfsnag uses the `"default"` field. CLI flags and env vars still override profile values.


### Environment Variables

- `LOGFIRE_READ_TOKEN` - Logfire read token
- `LOGFIRE_BASE_URL` - API base URL (default: `https://logfire-us.pydantic.dev`)

## Examples

### Fetch a trace (save + peek)

A traceId or URL fetch streams all records into SQLite, then prints a small peek summary including the `db` path:

```bash
lfsnag abc123def456789012345678abcdef01
lfsnag 'https://logfire-us.pydantic.dev/org/proj?traceId=abc123def456789012345678abcdef01&spanId=...'
```

The URL's `org/project` auto-selects the matching environment profile. The flag `-e` overrides this. Default file: `~/.cache/lfsnag/<traceId>.sqlite`. The flag `--save` sets a different path:

```bash
lfsnag --save /tmp/t.sqlite -e stage abc123def456789012345678abcdef01
```

Cached files are derived data. Clear them with `rm -rf -- "${XDG_CACHE_HOME:-$HOME/.cache}/lfsnag"` when no lfsnag fetch/query is running.

### Query locally (offline, no token)

```bash
lfsnag --db /tmp/t.sqlite --peek
lfsnag --db /tmp/t.sqlite --sql "SELECT span_name, count(*) cnt FROM records GROUP BY span_name ORDER BY cnt DESC"
lfsnag --db /tmp/t.sqlite --sql "SELECT span_name FROM records WHERE is_exception = 1"
```

Logfire attribute keys contain dots. Quote the JSON path in SQLite:

```bash
lfsnag --db /tmp/t.sqlite --sql "SELECT sum(json_extract(attributes, '\$.\"gen_ai.aggregated_usage.input_tokens\"')) FROM records"
```

### Raw SQL against the API

```bash
lfsnag -e dev --sql "SELECT span_name, count(*) as cnt FROM records GROUP BY span_name ORDER BY cnt DESC LIMIT 10"
lfsnag -e dev --sql "SELECT span_name, duration FROM records WHERE trace_id = '019d05ee9be731d9f95c339fb7b9c6c1' AND is_exception = true"
```

### Piping with jq

```bash
lfsnag -c abc123def456789012345678abcdef01 | jq '.exceptions'
lfsnag -c --db /tmp/t.sqlite --sql "SELECT span_name FROM records WHERE is_exception = 1" | jq '.[].span_name'
```

### Verbose Mode

```bash
lfsnag -v abc123def456789012345678abcdef01
```


## Make Commands

```bash
make build         # Build binary to bin/lfsnag
make run           # Build and run
make run/quick     # Run without rebuild
make test          # Run tests
make test-race     # Run tests with race detection
make coverage      # Run tests with coverage report
make coverage-html # Generate HTML coverage report
make fmt           # Format code
make vet           # Run go vet
make lint          # Run fmt and vet
make tidy          # Tidy Go modules
make clean         # Remove build artifacts
make install       # Install to GOPATH/bin
make build-all     # Build for linux/darwin/windows amd64/arm64
```

## Testing

```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
make coverage

# Run tests with race detection
make test-race
```

## Claude Code Plugin

lfsnag includes a [Claude Code](https://claude.ai/code) plugin. Use `/lfsnag:trace` to investigate Logfire traces directly from Claude Code.

### Install from GitHub

```bash
/plugin marketplace add bigbag/lfsnag
/plugin install lfsnag@bigbag-lfsnag
```

### Install from local path

```bash
/plugin marketplace add /path/to/lfsnag
/plugin install lfsnag@lfsnag
```

Then type `/lfsnag:trace` to start investigating a trace.

### Local development

```bash
claude --plugin-dir .
```

Use `/reload-plugins` after making changes to the skill without restarting.

## References

- [Logfire Query API](https://logfire.pydantic.dev/docs/how-to-guides/query-api/)

## License

MIT License - see [LICENSE](LICENSE) file.
