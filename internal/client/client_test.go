package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bigbag/lfsnag/internal/logfire"
	"github.com/bigbag/lfsnag/internal/output"
)

func TestQueryTraceRequest(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAuth string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"span_id":"abc"}]`))
	}))
	defer server.Close()

	printer := output.NewPrinter(&bytes.Buffer{}, &bytes.Buffer{}, false, false)
	c := New("test-token", "org/project", server.URL, printer)

	result, err := c.QueryTrace("abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("QueryTrace failed: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/v1/org/project/query" {
		t.Errorf("expected /v1/org/project/query, got %s", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("expected Bearer test-token, got %s", gotAuth)
	}

	var req logfire.QueryRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatalf("unmarshal request body failed: %v", err)
	}
	if req.SQL == "" {
		t.Error("expected non-empty SQL in request body")
	}

	if string(result) != `[{"span_id":"abc"}]` {
		t.Errorf("expected result body, got %s", string(result))
	}
}

func TestQueryTraceErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	printer := output.NewPrinter(&bytes.Buffer{}, &bytes.Buffer{}, false, false)
	c := New("bad-token", "org/project", server.URL, printer)

	_, err := c.QueryTrace("abcdef1234567890abcdef1234567890")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestQueryTraceVerbose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	var outBuf bytes.Buffer
	printer := output.NewPrinter(&outBuf, &bytes.Buffer{}, false, true)
	c := New("test-token", "org/project", server.URL, printer)

	_, err := c.QueryTrace("abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("QueryTrace failed: %v", err)
	}

	output := outBuf.String()
	if len(output) == 0 {
		t.Error("expected verbose output")
	}
}
