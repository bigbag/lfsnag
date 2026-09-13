package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

type Writer struct {
	db   *sql.DB
	tx   *sql.Tx
	stmt *sql.Stmt
	cols []string
	err  error
}

func NewWriter(path string) (*Writer, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Writer{db: db, tx: tx}, nil
}

func (w *Writer) Write(rows []map[string]any) (int, error) {
	if w.tx == nil {
		return 0, fmt.Errorf("writer is closed")
	}
	if w.err != nil {
		return 0, w.err
	}
	if len(rows) == 0 {
		if w.stmt == nil {
			if _, err := w.tx.Exec(`DROP TABLE IF EXISTS records`); err != nil {
				w.err = err
				return 0, err
			}
			if _, err := w.tx.Exec(`CREATE TABLE records ("span_name" TEXT)`); err != nil {
				w.err = err
				return 0, err
			}
		}
		return 0, nil
	}
	if w.stmt == nil {
		if err := w.createTable(rows); err != nil {
			w.err = err
			return 0, err
		}
	}
	for _, r := range rows {
		for k := range r {
			if !slices.Contains(w.cols, k) {
				w.err = fmt.Errorf("column %q appears after the first batch; schema is fixed by the first page", k)

				return 0, w.err
			}
		}
	}

	for _, r := range rows {
		args := make([]any, len(w.cols))
		for i, c := range w.cols {
			b, err := bind(r[c])
			if err != nil {
				w.err = err
				return 0, err
			}
			args[i] = b
		}
		if _, err := w.stmt.Exec(args...); err != nil {
			w.err = err
			return 0, err
		}
	}
	return len(rows), nil
}

func (w *Writer) createTable(rows []map[string]any) error {
	keys := map[string]struct{}{}
	for _, r := range rows {
		for k := range r {
			keys[k] = struct{}{}
		}
	}
	cols := make([]string, 0, len(keys))
	for k := range keys {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	if len(cols) == 0 {
		cols = []string{"span_name"}
	}
	defs := make([]string, len(cols))
	for i, c := range cols {
		defs[i] = quoteIdent(c)
	}
	if _, err := w.tx.Exec(`DROP TABLE IF EXISTS records`); err != nil {
		return err
	}
	if _, err := w.tx.Exec(`CREATE TABLE records (` + strings.Join(defs, ", ") + `)`); err != nil {
		return err
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",")
	q := `INSERT INTO records (` + identList(cols) + `) VALUES (` + placeholders + `)`
	stmt, err := w.tx.Prepare(q)
	if err != nil {
		return err
	}
	w.stmt = stmt
	w.cols = cols
	return nil
}

func (w *Writer) Commit() error {
	if w.tx == nil {
		return w.err
	}
	if w.stmt != nil {
		w.stmt.Close()
	}
	if w.err != nil {
		w.tx.Rollback()
		w.db.Close()
		w.tx = nil
		return w.err
	}
	err := w.tx.Commit()
	w.db.Close()
	w.tx = nil
	return err
}

func (w *Writer) Abort() error {
	if w.tx == nil {
		return nil
	}
	if w.stmt != nil {
		w.stmt.Close()
	}
	err := w.tx.Rollback()
	w.db.Close()
	w.tx = nil
	return err
}

func Save(path string, rows []map[string]any) (int, error) {
	w, err := NewWriter(path)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(rows)
	if err != nil {
		w.Abort()
		return 0, err
	}
	if err := w.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}

func Query(path, sqlQuery string) ([]map[string]any, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rs, err := db.Query(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	names, err := rs.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rs.Next() {
		raw := make([]any, len(names))
		ptrs := make([]any, len(names))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rs.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(names))
		for i, name := range names {
			row[name] = scanVal(raw[i])
		}
		out = append(out, row)
	}
	return out, rs.Err()
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func identList(cols []string) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = quoteIdent(c)
	}
	return strings.Join(parts, ", ")
}

func bind(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i, nil
		}
		f, err := x.Float64()
		if err != nil {
			return nil, fmt.Errorf("unusable number %q: %w", x.String(), err)
		}
		return f, nil
	case map[string]any, []any:
		b, err := json.Marshal(x)
		if err != nil {
			return nil, fmt.Errorf("value is not JSON-serializable: %w", err)
		}
		return string(b), nil
	default:
		return v, nil
	}
}

func scanVal(v any) any {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return v
	}
}

func QueryJSON(path, sqlQuery string) (json.RawMessage, error) {
	rows, err := Query(path, sqlQuery)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]any{}
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return b, nil
}
