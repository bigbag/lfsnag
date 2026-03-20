package logfire

import "fmt"

func BuildTraceQuery(traceID, fields string) string {
	if fields == "" {
		fields = "*"
	}
	return fmt.Sprintf("SELECT %s FROM records WHERE trace_id = '%s' ORDER BY start_timestamp", fields, traceID)
}

func Endpoint(baseURL string) string {
	return fmt.Sprintf("%s/v1/query", baseURL)
}
