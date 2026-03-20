package main

import "testing"

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
