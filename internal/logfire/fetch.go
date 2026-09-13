package logfire

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// FetchEach pages through a trace and gives each decoded batch to consume.
// The loop holds one page at a time. The loop stops after a short page.
// A repeated full page is an error. It shows that the server ignores OFFSET.
// The guard reads span_id. A slim -f fetch without span_id skips the guard.
func FetchEach(traceID, fields string, page int, get func(sql string) (json.RawMessage, error), consume func([]map[string]any) error) error {
	if page <= 0 {
		page = DefaultLimit
	}
	firstID := ""
	seenFirst := false
	for offset := 0; ; offset += page {
		raw, err := get(BuildPageQuery(traceID, fields, page, offset))
		if err != nil {
			return err
		}
		norm, err := Normalize(raw)
		if err != nil {
			return err
		}
		var rows []map[string]any
		dec := json.NewDecoder(bytes.NewReader(norm))
		dec.UseNumber()
		if err := dec.Decode(&rows); err != nil {
			return err
		}
		if err := consume(rows); err != nil {
			return err
		}
		if len(rows) < page {
			return nil
		}
		if id, _ := rows[0]["span_id"].(string); id != "" {
			if seenFirst && offset > 0 && id == firstID {
				return fmt.Errorf("query API is repeating the same page (OFFSET ignored?) at offset %d", offset)
			}
			firstID = id
			seenFirst = true
		}
	}
}
