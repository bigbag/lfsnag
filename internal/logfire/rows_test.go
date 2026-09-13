package logfire

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestColumnarToRows(t *testing.T) {
	raw := []byte(`{"columns":[{"name":"span_name","values":["a","b"]},{"name":"duration","values":[1.5,null]}]}`)
	out, err := Normalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		t.Fatalf("expected row array, got %s: %v", out, err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows", len(rows))
	}
	if rows[0]["span_name"] != "a" {
		t.Errorf("row0 span_name=%v", rows[0]["span_name"])
	}
	if rows[1]["duration"] != nil {
		t.Errorf("row1 duration=%v", rows[1]["duration"])
	}
}

func TestNormalizeRejectsRaggedColumns(t *testing.T) {
	raw := []byte(`{"columns":[{"name":"a","values":[1,2]},{"name":"b","values":[1]}]}`)
	if _, err := Normalize(raw); err == nil {
		t.Fatal("ragged columns must error")
	}
}

func TestNormalizeRejectsDuplicateColumns(t *testing.T) {
	raw := []byte(`{"columns":[{"name":"x","values":[1]},{"name":"x","values":[2]}]}`)
	if _, err := Normalize(raw); err == nil {
		t.Fatal("duplicate column names must error")
	}
}

func TestNormalizePreservesBigIntegers(t *testing.T) {
	raw := []byte(`{"columns":[{"name":"n","values":[9007199254740993]}]}`)
	out, err := Normalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "9007199254740993") {
		t.Fatalf("precision lost: %s", out)
	}
}

func TestNormalizeV2Rows(t *testing.T) {
	raw := []byte(`{"schema":{"fields":[{"name":"span_name"}]},"data":[{"span_name":"x"}]}`)
	out, err := Normalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		t.Fatalf("expected row array, got %s: %v", out, err)
	}
	if len(rows) != 1 || rows[0]["span_name"] != "x" {
		t.Fatalf("got %s", out)
	}
}

func TestPeekAcceptsSQLiteInt64Duration(t *testing.T) {
	p := Peek([]map[string]any{
		{"span_name": "a", "span_id": "aa", "parent_span_id": nil, "duration": int64(5)},
		{"span_name": "b", "span_id": "bb", "parent_span_id": "aa", "duration": int64(2)},
	})
	if p.NSlow != 1 || len(p.Slow) != 1 || *p.Slow[0].Duration != 5 {
		t.Fatalf("slow=%v total=%d", p.Slow, p.NSlow)
	}
	if p.Top[0].MaxDur == nil || *p.Top[0].MaxDur != 5 {
		t.Fatalf("max_dur=%v", p.Top[0].MaxDur)
	}
}

func TestPeekSummarizesRows(t *testing.T) {
	rows := []map[string]any{
		{"span_name": "root", "span_id": "aa", "parent_span_id": nil, "duration": 10.0, "is_exception": false, "kind": "span"},
		{"span_name": "child", "span_id": "bb", "parent_span_id": "aa", "duration": 9.0, "is_exception": true, "exception_type": "ValueError", "exception_message": "boom", "kind": "span"},
		{"span_name": "child", "span_id": "cc", "parent_span_id": "aa", "duration": 1.0, "is_exception": false, "kind": "span"},
	}
	p := Peek(rows)
	if p.N != 3 {
		t.Fatalf("n=%d", p.N)
	}
	if len(p.Roots) != 1 || p.Roots[0].SpanID != "aa" {
		t.Fatalf("roots=%v", p.Roots)
	}
	if len(p.Exceptions) != 1 || p.Exceptions[0].Type != "ValueError" {
		t.Fatalf("exceptions=%v", p.Exceptions)
	}
	if len(p.Top) == 0 || p.Top[0].SpanName != "child" || p.Top[0].Cnt != 2 {
		t.Fatalf("top=%v", p.Top)
	}
}

func TestPeekTreatsIntegerException(t *testing.T) {
	p := Peek([]map[string]any{
		{"span_name": "x", "span_id": "aa", "is_exception": int64(1), "exception_type": "ValueError", "exception_message": "boom"},
		{"span_name": "y", "span_id": "bb", "is_exception": int64(0)},
	})
	if len(p.Exceptions) != 1 || p.Exceptions[0].SpanID != "aa" {
		t.Fatalf("exceptions=%v", p.Exceptions)
	}
}

func TestPeekBudgetOnErrorHeavyTrace(t *testing.T) {
	rows := make([]map[string]any, 0, 600)
	for i := range 500 {
		rows = append(rows, map[string]any{
			"span_name": "boom", "span_id": fmt.Sprintf("e%03d", i), "parent_span_id": "root",
			"duration": 0.01, "is_exception": true,
			"exception_type": "ValueError", "exception_message": strings.Repeat("x", 500),
		})
	}
	for i := range 100 {
		rows = append(rows, map[string]any{
			"span_name": "slow", "span_id": fmt.Sprintf("s%03d", i), "parent_span_id": "root",
			"duration": 5.0, "is_exception": false,
		})
	}
	p := Peek(rows)
	if p.N != 600 {
		t.Fatalf("n=%d", p.N)
	}
	if p.NExceptions != 500 || len(p.Exceptions) != 20 {
		t.Fatalf("exceptions total=%d sample=%d", p.NExceptions, len(p.Exceptions))
	}
	if m := p.Exceptions[0].Message; len([]rune(m)) > 201 {
		t.Fatalf("message not truncated: %d", len(m))
	}
	if p.NSlow != 100 || len(p.Slow) != 20 {
		t.Fatalf("slow total=%d sample=%d", p.NSlow, len(p.Slow))
	}
}

func TestPeekTruncateKeepsValidUTF8(t *testing.T) {
	msg := strings.Repeat("é", 300) // 600 bytes, 300 runes
	p := Peek([]map[string]any{
		{"span_name": "x", "span_id": "a", "is_exception": true, "exception_message": msg},
	})
	m := p.Exceptions[0].Message
	if !utf8.ValidString(m) {
		t.Fatalf("truncated message is not valid UTF-8")
	}
	if r := len([]rune(m)); r > 201 {
		t.Fatalf("rune length %d", r)
	}
}

func TestPeekPaths(t *testing.T) {
	rows := []map[string]any{
		{"span_name": "root", "span_id": "r", "parent_span_id": nil, "duration": 10.0},
		{"span_name": "mid", "span_id": "m", "parent_span_id": "r", "duration": 9.0},
		{"span_name": "leaf", "span_id": "l", "parent_span_id": "m", "duration": 8.0, "is_exception": true, "exception_type": "E"},
	}
	p := Peek(rows)
	if len(p.Exceptions) != 1 || p.Exceptions[0].Path != "root › mid › leaf" {
		t.Fatalf("exceptions=%v", p.Exceptions)
	}
	var midSpan *PeekSpan
	for i := range p.Slow {
		if p.Slow[i].SpanID == "m" {
			midSpan = &p.Slow[i]
		}
	}
	if midSpan == nil || midSpan.Path != "root › mid" {
		t.Fatalf("slow=%v", p.Slow)
	}
}

func TestPeekSlowSortedBeforeCap(t *testing.T) {
	rows := make([]map[string]any, 0, 25)
	for i := range 20 {
		rows = append(rows, map[string]any{"span_name": "meh", "span_id": fmt.Sprintf("m%02d", i), "duration": 3.0})
	}
	rows = append(rows, map[string]any{"span_name": "worst", "span_id": "w", "duration": 100.0})
	p := Peek(rows)
	if p.NSlow != 21 || len(p.Slow) != 20 {
		t.Fatalf("total=%d sample=%d", p.NSlow, len(p.Slow))
	}
	if p.Slow[0].SpanID != "w" || *p.Slow[0].Duration != 100.0 {
		t.Fatalf("slowest missing from sample: %+v", p.Slow[0])
	}
}
