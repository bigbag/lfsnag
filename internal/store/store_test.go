package store

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bigbag/lfsnag/internal/logfire"
)

func TestSaveAndQuery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.sqlite")
	rows := []map[string]any{
		{"span_name": "a", "duration": 1.5, "is_exception": false, "attributes": map[string]any{"k": "v"}},
		{"span_name": "a", "duration": 3.0, "is_exception": true, "attributes": map[string]any{"k": "w"}},
		{"span_name": "b", "duration": 0.1, "is_exception": false},
	}
	n, err := Save(path, rows)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("n=%d", n)
	}
	got, err := Query(path, "SELECT span_name, count(*) AS cnt FROM records GROUP BY span_name ORDER BY cnt DESC")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got=%v", got)
	}
	if got[0]["span_name"] != "a" {
		t.Fatalf("top=%v", got[0])
	}
	exc, err := Query(path, "SELECT span_name FROM records WHERE is_exception = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(exc) != 1 || exc[0]["span_name"] != "a" {
		t.Fatalf("exc=%v", exc)
	}
}

func TestRoundTripIntegerDuration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rt.sqlite")
	get := func(sql string) (json.RawMessage, error) {
		if strings.Contains(sql, "OFFSET 0") {
			return json.RawMessage(`{"columns":[
				{"name":"span_name","values":["root","child"]},
				{"name":"span_id","values":["aa","bb"]},
				{"name":"parent_span_id","values":[null,"aa"]},
				{"name":"duration","values":[5,3]}
			]}`), nil
		}
		return json.RawMessage(`{"columns":[
			{"name":"span_name","values":[]},
			{"name":"span_id","values":[]},
			{"name":"parent_span_id","values":[]},
			{"name":"duration","values":[]}
		]}`), nil
	}
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	err = logfire.FetchEach("tid", "", 2, get, func(rows []map[string]any) error {
		_, err := w.Write(rows)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Commit(); err != nil {
		t.Fatal(err)
	}
	rows, err := Query(path, "SELECT span_name, span_id, parent_span_id, duration FROM records")
	if err != nil {
		t.Fatal(err)
	}
	p := logfire.Peek(rows)
	if p.N != 2 || p.NSlow != 2 || len(p.Slow) != 2 {
		t.Fatalf("n=%d nslow=%d slow=%v", p.N, p.NSlow, p.Slow)
	}
	if *p.Slow[0].Duration != 5 {
		t.Fatalf("duration=%v", *p.Slow[0].Duration)
	}
	var root *logfire.PeekName
	for i := range p.Top {
		if p.Top[i].SpanName == "root" {
			root = &p.Top[i]
		}
	}
	if root == nil || root.MaxDur == nil || *root.MaxDur != 5 {
		t.Fatalf("root max_dur=%v", root)
	}
}

func TestSavePeekKeepsExceptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.sqlite")
	_, err := Save(path, []map[string]any{
		{"span_name": "ok", "span_id": "aa", "is_exception": false},
		{"span_name": "bad", "span_id": "bb", "is_exception": true, "exception_type": "ValueError", "exception_message": "boom"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Query(path, "SELECT * FROM records")
	if err != nil {
		t.Fatal(err)
	}
	p := logfire.Peek(rows)
	if len(p.Exceptions) != 1 || p.Exceptions[0].SpanID != "bb" {
		t.Fatalf("exceptions=%v rows=%v", p.Exceptions, rows)
	}
}

func TestSaveFailedReplacementKeepsOldRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.sqlite")
	if _, err := Save(path, []map[string]any{{"span_name": "old"}}); err != nil {
		t.Fatal(err)
	}
	_, err := Save(path, []map[string]any{{"span_name": "new", "ch": make(chan int)}})
	if err == nil {
		t.Fatal("expected save to fail")
	}
	got, err := Query(path, "SELECT span_name FROM records")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["span_name"] != "old" {
		t.Fatalf("old rows lost: %v", got)
	}
}

func TestWriterBatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.sqlite")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]map[string]any{{"span_name": "a", "span_id": "1"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]map[string]any{{"span_name": "b", "span_id": "2"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Commit(); err != nil {
		t.Fatal(err)
	}

	got, err := Query(path, "SELECT count(*) AS n FROM records")
	if err != nil {
		t.Fatal(err)
	}
	if got[0]["n"] != int64(2) {
		t.Fatalf("n=%v", got[0]["n"])
	}
}

func TestWriterRejectsLateColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "late.sqlite")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]map[string]any{{"span_name": "a"}}); err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]map[string]any{{"span_name": "b", "duration": 1.0}})
	if err == nil || !strings.Contains(err.Error(), "duration") {
		t.Fatalf("late column must error naming it: %v", err)
	}
	w.Abort()
}

func TestWriterPropagatesMarshalError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.sqlite")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]map[string]any{{"span_name": "x", "attributes": map[string]any{"ch": make(chan int)}}})
	if err == nil {
		t.Fatal("marshal failure must surface, not commit empty string")
	}
	w.Abort()
}

func TestAbortKeepsExistingOnFailedStream(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.sqlite")
	if _, err := Save(path, []map[string]any{{"span_name": "old", "span_id": "0"}}); err != nil {
		t.Fatal(err)
	}
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	get := func(sql string) (json.RawMessage, error) {
		calls++
		if calls == 1 {
			return json.Marshal([]map[string]any{{"span_name": "new", "span_id": "1", "duration": nil}})
		}
		return nil, fmt.Errorf("page 2 HTTP failed")
	}
	err = logfire.FetchEach("tid", "span_name", 1, get, func(rows []map[string]any) error {
		_, err := w.Write(rows)
		return err
	})
	if err == nil {
		t.Fatal("expected fetch error")
	}
	if err := w.Abort(); err != nil {
		t.Fatal(err)
	}
	got, err := Query(path, "SELECT span_name FROM records")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["span_name"] != "old" {
		t.Fatalf("partial commit leaked: %v", got)
	}
}

func TestWriterTypeStabilityAcrossPages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.sqlite")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]map[string]any{{"span_name": "log", "duration": nil}}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]map[string]any{{"span_name": "span", "duration": 42.5}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Commit(); err != nil {
		t.Fatal(err)
	}
	got, err := Query(path, "SELECT max(duration) AS m FROM records")
	if err != nil {
		t.Fatal(err)
	}
	if got[0]["m"] != 42.5 {
		t.Fatalf("max(duration)=%v (%T) — numeric duration lost", got[0]["m"], got[0]["m"])
	}
}
