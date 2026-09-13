package logfire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	peekCap     = 20
	peekMsgMax  = 200
	peekPathMax = 8
)

type PeekResult struct {
	N           int         `json:"n"`
	Roots       []PeekSpan  `json:"roots"`
	NRoots      int         `json:"n_roots"`
	Top         []PeekName  `json:"top"`
	Exceptions  []PeekError `json:"exceptions"`
	NExceptions int         `json:"n_exceptions"`
	Slow        []PeekSpan  `json:"slow"`
	NSlow       int         `json:"n_slow"`
}

type PeekSpan struct {
	SpanName string   `json:"span_name"`
	SpanID   string   `json:"span_id"`
	Duration *float64 `json:"duration"`
	Path     string   `json:"path,omitempty"`
}

type PeekName struct {
	SpanName string   `json:"span_name"`
	Cnt      int      `json:"cnt"`
	MaxDur   *float64 `json:"max_dur"`
}

type PeekError struct {
	SpanName string `json:"span_name"`
	SpanID   string `json:"span_id"`
	Type     string `json:"type"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
}

func Normalize(raw json.RawMessage) (json.RawMessage, error) {
	probe, err := decode(raw)
	if err != nil {
		return nil, err
	}
	switch v := probe.(type) {
	case []any:
		return json.Marshal(v)
	case map[string]any:
		if cols, ok := v["columns"].([]any); ok {
			rows, err := columnarToRows(cols)
			if err != nil {
				return nil, err
			}
			return json.Marshal(rows)
		}
		if data, ok := v["data"].([]any); ok {
			return json.Marshal(data)
		}
	}
	return raw, nil
}

// decode reads numbers as json.Number. Integers above 2^53 keep their value.
func decode(raw json.RawMessage) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func columnarToRows(cols []any) ([]map[string]any, error) {
	type col struct {
		name   string
		values []any
	}
	parsed := make([]col, 0, len(cols))
	seen := map[string]bool{}
	n := -1
	for _, c := range cols {
		m, _ := c.(map[string]any)
		name, _ := m["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("columnar response: column without a name")
		}
		if seen[name] {
			return nil, fmt.Errorf("columnar response: duplicate column %q", name)
		}
		seen[name] = true
		vals, ok := m["values"].([]any)
		if !ok {
			return nil, fmt.Errorf("columnar response: column %q has no values array", name)
		}
		if n == -1 {
			n = len(vals)
		} else if len(vals) != n {
			return nil, fmt.Errorf("columnar response: ragged columns (column %q has %d values, expected %d)", name, len(vals), n)
		}
		parsed = append(parsed, col{name, vals})
	}
	if n < 0 {
		n = 0
	}
	rows := make([]map[string]any, n)
	for i := range n {
		row := make(map[string]any, len(parsed))
		for _, c := range parsed {
			row[c.name] = c.values[i]
		}
		rows[i] = row
	}
	return rows, nil
}

type spanRef struct {
	name   string
	parent string
}

func Peek(rows []map[string]any) PeekResult {
	p := PeekResult{
		N:          len(rows),
		Roots:      []PeekSpan{},
		Top:        []PeekName{},
		Exceptions: []PeekError{},
		Slow:       []PeekSpan{},
	}
	refs := make(map[string]spanRef, len(rows))
	for _, r := range rows {
		refs[str(r["span_id"])] = spanRef{name: str(r["span_name"]), parent: str(r["parent_span_id"])}
	}

	counts := map[string]*PeekName{}
	for _, r := range rows {
		name := str(r["span_name"])
		id := str(r["span_id"])
		dur := f64p(r["duration"])
		if r["parent_span_id"] == nil || str(r["parent_span_id"]) == "" {
			p.NRoots++
			if len(p.Roots) < peekCap {
				p.Roots = append(p.Roots, PeekSpan{SpanName: name, SpanID: id, Duration: dur})
			}
		}
		if truthy(r["is_exception"]) {
			p.NExceptions++
			if len(p.Exceptions) < peekCap {
				p.Exceptions = append(p.Exceptions, PeekError{
					SpanName: name, SpanID: id, Type: str(r["exception_type"]),
					Message: truncate(str(r["exception_message"]), peekMsgMax),
					Path:    path(refs, id),
				})
			}
		}
		if dur != nil && *dur > 2 {
			p.NSlow++
			p.Slow = append(p.Slow, PeekSpan{SpanName: name, SpanID: id, Duration: dur, Path: path(refs, id)})
		}

		agg := counts[name]
		if agg == nil {
			agg = &PeekName{SpanName: name}
			counts[name] = agg
		}
		agg.Cnt++
		if dur != nil && (agg.MaxDur == nil || *dur > *agg.MaxDur) {
			agg.MaxDur = dur
		}
	}
	for _, agg := range counts {
		p.Top = append(p.Top, *agg)
	}
	sort.Slice(p.Top, func(i, j int) bool {
		if p.Top[i].Cnt == p.Top[j].Cnt {
			return p.Top[i].SpanName < p.Top[j].SpanName
		}
		return p.Top[i].Cnt > p.Top[j].Cnt
	})
	sort.Slice(p.Slow, func(i, j int) bool {
		return *p.Slow[i].Duration > *p.Slow[j].Duration
	})

	if len(p.Top) > peekCap {
		p.Top = p.Top[:peekCap]
	}
	if len(p.Slow) > peekCap {
		p.Slow = p.Slow[:peekCap]
	}
	return p
}

func path(refs map[string]spanRef, id string) string {
	var parts []string
	seen := map[string]bool{}
	for cur := id; cur != ""; {
		if seen[cur] || len(parts) > 64 {
			break
		}
		seen[cur] = true
		ref, ok := refs[cur]
		if !ok {
			break
		}
		parts = append(parts, ref.name)
		cur = ref.parent
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	if len(parts) > peekPathMax {
		return "… › " + strings.Join(parts[len(parts)-peekPathMax:], " › ")
	}
	return strings.Join(parts, " › ")
}

func f64p(v any) *float64 {
	switch x := v.(type) {
	case float64:
		return &x
	case int64:
		f := float64(x)
		return &f

	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return nil
		}
		return &f
	}
	return nil
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case int64:
		return x != 0

	case float64:
		return x != 0
	case json.Number:
		n, err := x.Float64()
		return err == nil && n != 0
	}
	return false
}
