package logfire

import "fmt"

type QueryRequest struct {
	SQL string `json:"sql"`
}

func BuildTraceQuery(traceID string) string {
	return fmt.Sprintf("SELECT * FROM records WHERE trace_id = '%s' ORDER BY start_timestamp", traceID)
}

func Endpoint(baseURL, project string) string {
	return fmt.Sprintf("%s/v1/%s/query", baseURL, project)
}
