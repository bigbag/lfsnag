package client

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bigbag/lfsnag/internal/output"
)

func TestQueryTraceRequest(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAuth string
	var gotSQL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotSQL = r.URL.Query().Get("sql")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"span_id":"abc"}]`))
	}))
	defer server.Close()

	printer := output.NewPrinter(&bytes.Buffer{}, &bytes.Buffer{}, false, false)
	c := New("test-token", server.URL, printer)

	result, err := c.QueryTrace("abcdef1234567890abcdef1234567890", "")
	if err != nil {
		t.Fatalf("QueryTrace failed: %v", err)
	}

	if gotMethod != "GET" {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/v1/query" {
		t.Errorf("expected /v1/query, got %s", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("expected Bearer test-token, got %s", gotAuth)
	}
	if gotSQL == "" {
		t.Error("expected non-empty sql query param")
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
	c := New("bad-token", server.URL, printer)

	_, err := c.QueryTrace("abcdef1234567890abcdef1234567890", "")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestQueryRawSQL(t *testing.T) {
	var gotSQL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSQL = r.URL.Query().Get("sql")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"span_name":"test","duration":0.5}]`))
	}))
	defer server.Close()

	printer := output.NewPrinter(&bytes.Buffer{}, &bytes.Buffer{}, false, false)
	c := New("test-token", server.URL, printer)

	sql := "SELECT span_name, duration FROM records WHERE is_exception = true"
	result, err := c.Query(sql)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if gotSQL != sql {
		t.Errorf("expected sql %q, got %q", sql, gotSQL)
	}

	if string(result) != `[{"span_name":"test","duration":0.5}]` {
		t.Errorf("unexpected result: %s", string(result))
	}
}

func TestQueryRawSQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"syntax error in SQL"}`))
	}))
	defer server.Close()

	printer := output.NewPrinter(&bytes.Buffer{}, &bytes.Buffer{}, false, false)
	c := New("test-token", server.URL, printer)

	_, err := c.Query("INVALID SQL")
	if err == nil {
		t.Fatal("expected error for 400 response")
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
	c := New("test-token", server.URL, printer)

	_, err := c.QueryTrace("abcdef1234567890abcdef1234567890", "")
	if err != nil {
		t.Fatalf("QueryTrace failed: %v", err)
	}

	output := outBuf.String()
	if len(output) == 0 {
		t.Error("expected verbose output")
	}
}
