package logfire

import (
	"testing"
)

func TestBuildTraceQuery(t *testing.T) {
	query := BuildTraceQuery("abc123def456", "")
	expected := "SELECT * FROM records WHERE trace_id = 'abc123def456' ORDER BY start_timestamp, span_id"
	if query != expected {
		t.Errorf("expected %s, got %s", expected, query)
	}
}

func TestBuildTraceQueryWithFields(t *testing.T) {
	query := BuildTraceQuery("abc123def456", "span_name,start_timestamp,duration")
	expected := "SELECT span_name,start_timestamp,duration FROM records WHERE trace_id = 'abc123def456' ORDER BY start_timestamp, span_id"
	if query != expected {
		t.Errorf("expected %s, got %s", expected, query)
	}
}

func TestBuildPageQuery(t *testing.T) {
	got := BuildPageQuery("abc123def456", "span_id", 10000, 10000)
	want := "SELECT span_id FROM records WHERE trace_id = 'abc123def456' ORDER BY start_timestamp, span_id LIMIT 10000 OFFSET 10000"
	if got != want {
		t.Errorf("got %s", got)
	}
}

func TestEndpoint(t *testing.T) {
	url := Endpoint("https://logfire-us.pydantic.dev")
	expected := "https://logfire-us.pydantic.dev/v1/query"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}
