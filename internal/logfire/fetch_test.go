package logfire

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestFetchEachStreamsBatches(t *testing.T) {
	page := 2
	calls := 0
	get := func(sql string) (json.RawMessage, error) {
		calls++
		switch calls {
		case 1:
			return json.Marshal([]map[string]any{{"span_id": "a"}, {"span_id": "b"}})
		case 2:
			return json.Marshal([]map[string]any{{"span_id": "c"}})
		}
		return nil, fmt.Errorf("extra call %d", calls)
	}
	var sizes []int
	total := 0
	err := FetchEach("tid", "span_id", page, get, func(rows []map[string]any) error {
		sizes = append(sizes, len(rows))
		total += len(rows)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || total != 3 {
		t.Fatalf("calls=%d total=%d", calls, total)
	}
	if len(sizes) != 2 || sizes[0] != 2 || sizes[1] != 1 {
		t.Fatalf("sizes=%v", sizes)
	}
}

func TestFetchEachErrorsOnRepeatedPage(t *testing.T) {
	page := 2
	calls := 0
	get := func(sql string) (json.RawMessage, error) {
		calls++
		// server that ignores OFFSET: same full page forever
		return json.Marshal([]map[string]any{{"span_id": "a"}, {"span_id": "b"}})
	}
	err := FetchEach("tid", "span_id", page, get, func(rows []map[string]any) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected repeated-page error")
	}
	if !strings.Contains(err.Error(), "repeating") {
		t.Fatalf("err=%v", err)
	}
	if calls > 3 {
		t.Fatalf("looped %d times before detecting", calls)
	}
}

func TestFetchEachAllowsLargeDistinctTraces(t *testing.T) {
	page := 2
	i := 0
	get := func(sql string) (json.RawMessage, error) {
		i++
		if i > 60000 { // final short page; 120k distinct rows total
			return json.Marshal([]map[string]any{{"span_id": fmt.Sprintf("f%d", i)}})
		}
		return json.Marshal([]map[string]any{{"span_id": fmt.Sprintf("a%d", i)}, {"span_id": fmt.Sprintf("b%d", i)}})
	}
	total := 0
	err := FetchEach("tid", "span_id", page, get, func(rows []map[string]any) error {
		total += len(rows)
		return nil
	})
	if err != nil {
		t.Fatalf("large distinct trace must fetch fully: %v", err)
	}
	if total < 120000 {
		t.Fatalf("total=%d", total)
	}
}
