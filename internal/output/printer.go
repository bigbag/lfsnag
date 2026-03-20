package output

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Printer struct {
	out     io.Writer
	errOut  io.Writer
	compact bool
	verbose bool
}

func NewPrinter(out, errOut io.Writer, compact, verbose bool) *Printer {
	return &Printer{
		out:     out,
		errOut:  errOut,
		compact: compact,
		verbose: verbose,
	}
}

func (p *Printer) PrintJSON(v any) error {
	var data []byte
	var err error

	if p.compact {
		data, err = json.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}
	if err != nil {
		return err
	}

	fmt.Fprintln(p.out, string(data))
	return nil
}

func (p *Printer) PrintRawJSON(raw json.RawMessage) error {
	if p.compact {
		fmt.Fprintln(p.out, string(raw))
		return nil
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		fmt.Fprintln(p.out, string(raw))
		return nil
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(p.out, string(raw))
		return nil
	}

	fmt.Fprintln(p.out, string(data))
	return nil
}

func (p *Printer) PrintRequest(method, url string, headers map[string]string, body []byte) {
	if !p.verbose {
		return
	}

	fmt.Fprintf(p.out, "> %s %s\n", method, url)
	for k, v := range headers {
		fmt.Fprintf(p.out, "> %s: %s\n", k, v)
	}
	fmt.Fprintln(p.out, ">")

	if len(body) > 0 {
		var v any
		if err := json.Unmarshal(body, &v); err == nil {
			data, _ := json.MarshalIndent(v, "> ", "  ")
			fmt.Fprintf(p.out, "> %s\n", string(data))
		} else {
			fmt.Fprintf(p.out, "> %s\n", string(body))
		}
	}
	fmt.Fprintln(p.out)
}

func (p *Printer) PrintResponse(resp *http.Response) {
	if !p.verbose {
		return
	}

	fmt.Fprintf(p.out, "< %s\n", resp.Status)
	for k, v := range resp.Header {
		fmt.Fprintf(p.out, "< %s: %s\n", k, strings.Join(v, ", "))
	}
	fmt.Fprintln(p.out, "<")
	fmt.Fprintln(p.out)
}

func (p *Printer) PrintVerbose(format string, args ...any) {
	if !p.verbose {
		return
	}
	fmt.Fprintf(p.errOut, format+"\n", args...)
}

func (p *Printer) PrintError(err error) {
	fmt.Fprintf(p.errOut, "error: %v\n", err)
}
