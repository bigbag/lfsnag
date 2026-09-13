package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/bigbag/lfsnag/internal/output"
	"github.com/bigbag/lfsnag/internal/store"
)

func TestReorderArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{

		{
			name: "empty args",
			args: []string{},
			want: []string{},
		},
		{
			name: "program name only",
			args: []string{"prog"},
			want: []string{"prog"},
		},
		{
			name: "positional arg only",
			args: []string{"prog", "traceid"},
			want: []string{"prog", "traceid"},
		},
		{
			name: "flags before positional",
			args: []string{"prog", "-c", "traceid"},
			want: []string{"prog", "-c", "traceid"},
		},
		{
			name: "positional before flag",
			args: []string{"prog", "traceid", "-c"},
			want: []string{"prog", "-c", "traceid"},
		},
		{
			name: "flag with equals value",
			args: []string{"prog", "traceid", "--token=abc"},
			want: []string{"prog", "--token=abc", "traceid"},
		},
		{
			name: "flag with next-arg value",
			args: []string{"prog", "traceid", "--token", "abc"},
			want: []string{"prog", "--token", "abc", "traceid"},
		},
		{
			name: "mixed flags and positional",
			args: []string{"prog", "traceid", "-c", "--token", "abc"},
			want: []string{"prog", "-c", "--token", "abc", "traceid"},
		},
		{
			name: "short flag alias with value",
			args: []string{"prog", "traceid", "-e", "prod"},
			want: []string{"prog", "-e", "prod", "traceid"},
		},
		{
			name: "save and db flags with values",
			args: []string{"prog", "traceid", "--save", "t.sqlite", "--db", "t.sqlite"},
			want: []string{"prog", "--save", "t.sqlite", "--db", "t.sqlite", "traceid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reorderArgs(tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("reorderArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("reorderArgs(%v) = %v, want %v", tt.args, got, tt.want)
					break
				}
			}
		})
	}
}

func TestParseTraceRef(t *testing.T) {
	cases := []struct {
		name, in, wantID, wantProject string
	}{
		{"raw id", "019d05d5c291ce65f49caad9bf2ebbdc", "019d05d5c291ce65f49caad9bf2ebbdc", ""},
		{"url", "https://logfire-us.pydantic.dev/jasper-calloway/stage-mra?q=x&traceId=019d05d5c291ce65f49caad9bf2ebbdc", "019d05d5c291ce65f49caad9bf2ebbdc", "jasper-calloway/stage-mra"},
		{"url without traceId", "https://logfire-us.pydantic.dev/org/proj", "https://logfire-us.pydantic.dev/org/proj", ""},
		{"not a trace id", "not-a-trace-id", "not-a-trace-id", ""},
	}
	for _, tc := range cases {
		id, project := parseTraceRef(tc.in)
		if id != tc.wantID || project != tc.wantProject {
			t.Errorf("%s: got (%s, %s)", tc.name, id, project)
		}
	}
}

func TestCheckMode(t *testing.T) {
	if err := checkMode(true, "SELECT 1", "", "", 1); err == nil {
		t.Fatal("peek+sql")
	}
	if err := checkMode(false, "SELECT 1", "t.sqlite", "", 0); err == nil {
		t.Fatal("save+sql")
	}
	if err := checkMode(false, "", "", "t.sqlite", 0); err == nil {
		t.Fatal("db without sql/peek")
	}
	if err := checkMode(true, "", "", "t.sqlite", 0); err != nil {
		t.Fatal(err)
	}
	if err := checkMode(false, "SELECT 1", "", "t.sqlite", 0); err != nil {
		t.Fatal(err)
	}
}

func TestCheckModeFetchRules(t *testing.T) {
	if err := checkMode(true, "", "", "", 1); err == nil {
		t.Fatal("remote peek rejected: bare fetch always peeks")
	}
	if err := checkMode(false, "", "t.sqlite", "", 1); err != nil {
		t.Fatal(err)
	}
	if err := checkMode(false, "", "", "", 1); err != nil {
		t.Fatal(err)
	}
}

func TestCheckModeDBRejectsIgnoredInputs(t *testing.T) {
	if err := checkMode(false, "", "out.sqlite", "f.sqlite", 1); err == nil {
		t.Fatal("--db must reject positional traceId")
	}
	if err := checkMode(true, "", "out.sqlite", "f.sqlite", 0); err == nil {
		t.Fatal("--db must reject --save")
	}
	if err := checkMode(true, "", "", "f.sqlite", 0); err != nil {
		t.Fatal(err)
	}
}

func TestPeekDBSlimNote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slim.sqlite")
	if _, err := store.Save(path, []map[string]any{
		{"span_name": "a", "duration": 5.0},
		{"span_name": "b", "duration": 1.0},
	}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	printer := output.NewPrinter(&buf, &bytes.Buffer{}, true, false)
	p, note := peekDB(printer, path)
	if note == "" {
		t.Fatal("expected slim-projection note")
	}
	if p.N != 2 {
		t.Fatalf("n=%d", p.N)
	}
	if err := printer.PrintJSON(peekPayload("", p, note)); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"n_roots", "n_exceptions", "n_slow", "roots", "slow", "top", "exceptions"} {
		if _, ok := out[k]; ok {
			t.Fatalf("slim payload must omit %q, got %s", k, buf.String())
		}
	}
	if out["n"] != float64(2) || out["note"] == "" {
		t.Fatalf("payload=%s", buf.String())
	}
}
