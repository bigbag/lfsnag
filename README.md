# lfsnag

[![Go Version](https://img.shields.io/github/go-mod/go-version/bigbag/lfsnag)](https://github.com/bigbag/lfsnag)
[![Build](https://img.shields.io/github/actions/workflow/status/bigbag/lfsnag/build.yaml?branch=master)](https://github.com/bigbag/lfsnag/actions/workflows/build.yaml)
[![Release](https://img.shields.io/github/v/release/bigbag/lfsnag)](https://github.com/bigbag/lfsnag/releases/latest)
[![license](https://img.shields.io/github/license/bigbag/lfsnag.svg)](https://github.com/bigbag/lfsnag/blob/master/LICENSE)

A CLI tool to fetch full trace details from [Pydantic Logfire](https://logfire.pydantic.dev/) by traceId.

## Features

- **Single-purpose** - Query traces by traceId, nothing more
- **Pretty output** - Formatted JSON by default
- **Compact mode** - Machine-friendly JSON for scripting
- **Verbose mode** - Show HTTP request/response details
- **Flexible config** - CLI flags, environment variables, or config file

## Quick Start

```bash
# Build
make build

# Query a trace
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
- `-e, --env` - Environment profile name
- `-f, --fields` - Comma-separated list of fields to select (default: all)
- `--sql` - Raw SQL query to execute against the Logfire API
- `--token` - Override read token

## Available Fields

- `start_timestamp` - When the span/log was created (UTC)
- `end_timestamp` - When the span/log completed (UTC)
- `duration` - Elapsed time in seconds (NULL for logs)
- `trace_id` - Trace identifier (32 hex chars)
- `span_id` - Span identifier (16 hex chars)
- `parent_span_id` - Parent span reference (NULL for root spans)
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

Configuration is resolved in priority order: **CLI flags > environment variables > config file**.

### Environment Profiles

Path: `~/.config/lfsnag/config.json`

```json
{
  "default": "prod",
  "environments": {
    "prod": {
      "token": "prod-read-token",
      "base_url": "https://logfire-us.pydantic.dev"
    },
    "stage": {
      "token": "stage-read-token",
      "base_url": "https://logfire-eu.pydantic.dev"
    }
  }
}
```

Select an environment with `-e`:

```bash
lfsnag -e prod abc123def456789012345678abcdef01
```

If `-e` is omitted, the `"default"` field is used. CLI flags and env vars still override profile values.

### Environment Variables

- `LOGFIRE_READ_TOKEN` - Logfire read token
- `LOGFIRE_BASE_URL` - API base URL (default: `https://logfire-us.pydantic.dev`)

## Examples

### Basic Query

```bash
lfsnag abc123def456789012345678abcdef01
```

### From Logfire URL

Paste a full Logfire URL — the `traceId` query parameter is extracted automatically:

```bash
lfsnag 'https://logfire-us.pydantic.dev/org/proj?traceId=abc123def456789012345678abcdef01&spanId=...'
```

### Compact Output

```bash
lfsnag -c abc123def456789012345678abcdef01
```

### Verbose Mode

```bash
lfsnag -v abc123def456789012345678abcdef01
```

### Select Specific Fields

```bash
lfsnag -f span_name,start_timestamp,duration abc123def456789012345678abcdef01
```

### With Token Flag

```bash
lfsnag --token "your-token" abc123def456789012345678abcdef01
```

### Raw SQL Query

```bash
lfsnag -e dev --sql "SELECT span_name, duration FROM records WHERE is_exception = true"
```

Filter by trace with custom fields:
```bash
lfsnag -e dev --sql "SELECT span_name, duration FROM records WHERE trace_id = '019d05ee9be731d9f95c339fb7b9c6c1' AND is_exception = true"
```

Aggregate spans:
```bash
lfsnag -e dev --sql "SELECT span_name, count(*) as cnt FROM records GROUP BY span_name ORDER BY cnt DESC LIMIT 10"
```

### Piping with jq

Extract span names:
```bash
lfsnag -c abc123def456789012345678abcdef01 | jq '.[].span_name'
```

Count spans in a trace:
```bash
lfsnag -c abc123def456789012345678abcdef01 | jq 'length'
```

Filter spans by attribute:
```bash
lfsnag -c abc123def456789012345678abcdef01 | jq '[.[] | select(.is_exception == true)]'
```

Save trace to file:
```bash
lfsnag abc123def456789012345678abcdef01 > trace.json
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

## References

- [Logfire Query API](https://logfire.pydantic.dev/docs/how-to-guides/query-api/)

## License

MIT License - see [LICENSE](LICENSE) file.
