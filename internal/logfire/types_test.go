package logfire

import (
	"testing"
)

func TestBuildTraceQuery(t *testing.T) {
	query := BuildTraceQuery("abc123def456", "")
	expected := "SELECT * FROM records WHERE trace_id = 'abc123def456' ORDER BY start_timestamp"
	if query != expected {
		t.Errorf("expected %s, got %s", expected, query)
	}
}

func TestBuildTraceQueryWithFields(t *testing.T) {
	query := BuildTraceQuery("abc123def456", "span_name,start_timestamp,duration")
	expected := "SELECT span_name,start_timestamp,duration FROM records WHERE trace_id = 'abc123def456' ORDER BY start_timestamp"
	if query != expected {
		t.Errorf("expected %s, got %s", expected, query)
	}
}

func TestEndpoint(t *testing.T) {
	url := Endpoint("https://logfire-us.pydantic.dev")
	expected := "https://logfire-us.pydantic.dev/v1/query"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}
