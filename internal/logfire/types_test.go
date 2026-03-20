package logfire

import (
	"encoding/json"
	"testing"
)

func TestBuildTraceQuery(t *testing.T) {
	query := BuildTraceQuery("abc123def456")
	expected := "SELECT * FROM records WHERE trace_id = 'abc123def456' ORDER BY start_timestamp"
	if query != expected {
		t.Errorf("expected %s, got %s", expected, query)
	}
}

func TestEndpoint(t *testing.T) {
	url := Endpoint("https://logfire-us.pydantic.dev", "myorg/myproject")
	expected := "https://logfire-us.pydantic.dev/v1/myorg/myproject/query"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestQueryRequestJSON(t *testing.T) {
	req := QueryRequest{SQL: "SELECT * FROM records"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded QueryRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.SQL != req.SQL {
		t.Errorf("expected %s, got %s", req.SQL, decoded.SQL)
	}
}
