package logfire

import "fmt"

const DefaultLimit = 10000

// PeekColumns lists the columns that Peek reads. It keeps attributes out of memory.
const PeekColumns = "span_name, span_id, parent_span_id, kind, duration, is_exception, exception_type, exception_message, level, otel_status_code"

func BuildTraceQuery(traceID, fields string) string {
	if fields == "" {
		fields = "*"
	}
	return fmt.Sprintf("SELECT %s FROM records WHERE trace_id = '%s' ORDER BY start_timestamp, span_id", fields, traceID)
}

func BuildPageQuery(traceID, fields string, limit, offset int) string {
	return fmt.Sprintf("%s LIMIT %d OFFSET %d", BuildTraceQuery(traceID, fields), limit, offset)
}

func Endpoint(baseURL string) string {
	return fmt.Sprintf("%s/v1/query", baseURL)
}
