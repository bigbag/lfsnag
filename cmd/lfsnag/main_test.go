package main

import (
	"testing"
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

func TestParseTraceID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "raw trace id",
			input: "019d05d5c291ce65f49caad9bf2ebbdc",
			want:  "019d05d5c291ce65f49caad9bf2ebbdc",
		},
		{
			name:  "logfire url with traceId param",
			input: "https://logfire-us.pydantic.dev/jasper-calloway/dev-mra?q=trace_id%3D%27019d05d5c291ce65f49caad9bf2ebbdc%27&traceId=019d05d5c291ce65f49caad9bf2ebbdc",
			want:  "019d05d5c291ce65f49caad9bf2ebbdc",
		},
		{
			name:  "url without traceId param",
			input: "https://logfire-us.pydantic.dev/org/proj?q=something",
			want:  "https://logfire-us.pydantic.dev/org/proj?q=something",
		},
		{
			name:  "invalid input passthrough",
			input: "not-a-trace-id",
			want:  "not-a-trace-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTraceID(tt.input)
			if got != tt.want {
				t.Errorf("parseTraceID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
