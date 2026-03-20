package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestPrinterPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, false)

	data := map[string]string{"key": "value"}
	err := p.PrintJSON(data)
	if err != nil {
		t.Fatalf("PrintJSON failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"key\"") {
		t.Errorf("expected key in output, got %s", output)
	}
	if !strings.Contains(output, "\"value\"") {
		t.Errorf("expected value in output, got %s", output)
	}
}

func TestPrinterPrintJSONCompact(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, true, false)

	data := map[string]string{"key": "value"}
	err := p.PrintJSON(data)
	if err != nil {
		t.Fatalf("PrintJSON failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	expected := `{"key":"value"}`
	if output != expected {
		t.Errorf("expected %s, got %s", expected, output)
	}
}

func TestPrinterPrintJSONPretty(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, false)

	data := map[string]string{"key": "value"}
	err := p.PrintJSON(data)
	if err != nil {
		t.Fatalf("PrintJSON failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\n") {
		t.Errorf("expected pretty output with newlines, got %s", output)
	}
}

func TestPrinterPrintRawJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, false)

	raw := json.RawMessage(`{"tools":[{"name":"test"}]}`)
	err := p.PrintRawJSON(raw)
	if err != nil {
		t.Fatalf("PrintRawJSON failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "tools") {
		t.Errorf("expected tools in output, got %s", output)
	}
}

func TestPrinterPrintRawJSONCompact(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, true, false)

	raw := json.RawMessage(`{"key":"value"}`)
	err := p.PrintRawJSON(raw)
	if err != nil {
		t.Fatalf("PrintRawJSON failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	expected := `{"key":"value"}`
	if output != expected {
		t.Errorf("expected %s, got %s", expected, output)
	}
}

func TestPrinterPrintRawJSONInvalid(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, false)

	raw := json.RawMessage(`not valid json`)
	err := p.PrintRawJSON(raw)
	if err != nil {
		t.Fatalf("PrintRawJSON should not fail on invalid JSON: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "not valid json" {
		t.Errorf("expected raw output for invalid JSON, got %s", output)
	}
}

func TestPrinterPrintVerbose(t *testing.T) {
	var errBuf bytes.Buffer
	p := NewPrinter(&bytes.Buffer{}, &errBuf, false, true)

	p.PrintVerbose("test message %s", "arg")

	output := errBuf.String()
	if !strings.Contains(output, "test message arg") {
		t.Errorf("expected verbose message, got %s", output)
	}
}

func TestPrinterPrintVerboseDisabled(t *testing.T) {
	var errBuf bytes.Buffer
	p := NewPrinter(&bytes.Buffer{}, &errBuf, false, false)

	p.PrintVerbose("test message")

	output := errBuf.String()
	if output != "" {
		t.Errorf("expected no output when verbose disabled, got %s", output)
	}
}

func TestPrinterPrintError(t *testing.T) {
	var errBuf bytes.Buffer
	p := NewPrinter(&bytes.Buffer{}, &errBuf, false, false)

	p.PrintError(errors.New("test error"))
	output := errBuf.String()
	if !strings.Contains(output, "error:") {
		t.Errorf("expected error prefix, got %s", output)
	}
	if !strings.Contains(output, "test error") {
		t.Errorf("expected error message, got %s", output)
	}
}

func TestPrinterPrintErrorToStderr(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	p := NewPrinter(&outBuf, &errBuf, false, false)

	p.PrintError(errors.New("test error"))

	if outBuf.String() != "" {
		t.Errorf("expected no output to stdout, got %s", outBuf.String())
	}
	if !strings.Contains(errBuf.String(), "error:") {
		t.Errorf("expected error to stderr, got %s", errBuf.String())
	}
}

func TestPrinterPrintRequestVerbose(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, true)

	headers := map[string]string{"Authorization": "Bearer token"}
	body := []byte(`{"method":"test"}`)

	p.PrintRequest("POST", "http://localhost/mcp", headers, body)

	output := buf.String()
	if !strings.Contains(output, "POST") {
		t.Errorf("expected POST in output, got %s", output)
	}
	if !strings.Contains(output, "Authorization") {
		t.Errorf("expected header in output, got %s", output)
	}
}

func TestPrinterPrintRequestNotVerbose(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, false)

	p.PrintRequest("POST", "http://localhost/mcp", nil, nil)

	output := buf.String()
	if output != "" {
		t.Errorf("expected no output when not verbose, got %s", output)
	}
}

func TestPrinterPrintResponseVerbose(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, true)

	resp := &http.Response{
		Status: "200 OK",
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}

	p.PrintResponse(resp)

	output := buf.String()
	if !strings.Contains(output, "200 OK") {
		t.Errorf("expected status in output, got %s", output)
	}
	if !strings.Contains(output, "Content-Type") {
		t.Errorf("expected header in output, got %s", output)
	}
}

func TestPrinterPrintResponseNotVerbose(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, false)

	resp := &http.Response{
		Status: "200 OK",
		Header: http.Header{},
	}

	p.PrintResponse(resp)

	if buf.String() != "" {
		t.Errorf("expected no output when not verbose, got %s", buf.String())
	}
}

func TestPrinterPrintRequestVerboseNonJSONBody(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, true)

	body := []byte("plain text body")
	p.PrintRequest("POST", "http://localhost/api", nil, body)

	output := buf.String()
	if !strings.Contains(output, "plain text body") {
		t.Errorf("expected raw body in output, got %s", output)
	}
}

func TestPrinterPrintRequestVerboseNoBody(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, &bytes.Buffer{}, false, true)

	p.PrintRequest("GET", "http://localhost/api", map[string]string{"Accept": "application/json"}, nil)

	output := buf.String()
	if !strings.Contains(output, "GET") {
		t.Errorf("expected method in output, got %s", output)
	}
	if !strings.Contains(output, "Accept") {
		t.Errorf("expected header in output, got %s", output)
	}
}
