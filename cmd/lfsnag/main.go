package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bigbag/lfsnag/internal/client"
	"github.com/bigbag/lfsnag/internal/config"
	"github.com/bigbag/lfsnag/internal/logfire"
	"github.com/bigbag/lfsnag/internal/output"
	"github.com/bigbag/lfsnag/internal/store"
)

var traceIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

var flagsWithValues = map[string]bool{
	"--token": true, "-token": true,
	"--env": true, "-env": true, "-e": true,
	"--fields": true, "-fields": true, "-f": true,
	"--sql": true, "-sql": true,
	"--save": true, "-save": true,
	"--db": true, "-db": true,
}

func parseTraceRef(arg string) (id, project string) {
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		u, err := url.Parse(arg)
		if err == nil {
			if id = u.Query().Get("traceId"); id != "" {
				project = projectFromPath(u.Path)
				return id, project
			}
		}
	}
	return arg, ""
}

func projectFromPath(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" && !strings.HasPrefix(parts[0], "-") {
		return parts[0] + "/" + parts[1]
	}
	return ""
}

func reorderArgs(args []string) []string {
	if len(args) <= 1 {
		return args
	}

	var flags []string
	var positional []string

	i := 1
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "=") {
				flags = append(flags, arg)
				i++
			} else if flagsWithValues[arg] && i+1 < len(args) {
				flags = append(flags, arg, args[i+1])
				i += 2
			} else {
				flags = append(flags, arg)
				i++
			}
		} else {
			positional = append(positional, arg)
			i++
		}
	}

	result := []string{args[0]}
	result = append(result, flags...)
	result = append(result, positional...)
	return result
}

func checkMode(peek bool, sql, save, db string, nargs int) error {
	if sql != "" && peek {
		return fmt.Errorf("--sql cannot be combined with --peek")
	}
	if sql != "" && save != "" {
		return fmt.Errorf("--sql cannot be combined with --save")
	}
	if db != "" {
		if nargs > 0 {
			return fmt.Errorf("--db cannot be combined with a traceId")
		}
		if save != "" {
			return fmt.Errorf("--db cannot be combined with --save")
		}
		if sql == "" && !peek {
			return fmt.Errorf("--db requires --sql or --peek")
		}
		return nil
	}

	if peek {
		return fmt.Errorf("--peek requires --db (a bare traceId already peeks)")
	}
	if sql != "" && nargs > 0 {

		return fmt.Errorf("--sql and traceId are mutually exclusive")
	}
	if sql == "" && nargs < 1 {
		return fmt.Errorf("traceId or --sql is required")
	}
	return nil
}

func main() {
	var (
		compact    bool
		verbose    bool
		flagPeek   bool
		flagToken  string
		flagEnv    string
		flagFields string
		flagSQL    string
		flagSave   string
		flagDB     string
	)

	flag.BoolVar(&compact, "c", false, "Compact JSON output")
	flag.BoolVar(&compact, "compact", false, "Compact JSON output")
	flag.BoolVar(&verbose, "v", false, "Show HTTP request/response details")
	flag.BoolVar(&verbose, "verbose", false, "Show HTTP request/response details")
	flag.BoolVar(&flagPeek, "peek", false, "Peek summary (only with --db)")
	flag.StringVar(&flagToken, "token", "", "Override read token")
	flag.StringVar(&flagEnv, "e", "", "Environment profile name")
	flag.StringVar(&flagEnv, "env", "", "Environment profile name")
	flag.StringVar(&flagFields, "f", "", "Comma-separated list of fields to save (default: all)")
	flag.StringVar(&flagFields, "fields", "", "Comma-separated list of fields to save (default: all)")
	flag.StringVar(&flagSQL, "sql", "", "SQL query (API, or --db local file)")
	flag.StringVar(&flagSave, "save", "", "SQLite path (default: cache dir)")
	flag.StringVar(&flagDB, "db", "", "Query a local SQLite file")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lfsnag [options] <traceId | logfire-url>\n")
		fmt.Fprintf(os.Stderr, "       lfsnag [options] --sql \"<query>\"\n")
		fmt.Fprintf(os.Stderr, "       lfsnag --db <file> (--sql \"<query>\" | --peek)\n\n")
		fmt.Fprintf(os.Stderr, "A traceId fetch streams all records into a SQLite file, then prints a peek summary.\n")
		fmt.Fprintf(os.Stderr, "The URL's org/project auto-selects the matching environment profile.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  lfsnag 'https://logfire-us.pydantic.dev/org/proj?traceId=019d05ee9be731d9f95c339fb7b9c6c1'\n")
		fmt.Fprintf(os.Stderr, "  lfsnag --db ~/.cache/lfsnag/019d05ee9be731d9f95c339fb7b9c6c1.sqlite --sql \"SELECT span_name FROM records WHERE is_exception = 1\"\n")
		fmt.Fprintf(os.Stderr, "  lfsnag --db ~/.cache/lfsnag/019d05ee9be731d9f95c339fb7b9c6c1.sqlite --peek\n")
		fmt.Fprintf(os.Stderr, "  lfsnag -e stage --sql \"SELECT span_name, duration FROM records WHERE is_exception\"\n")
	}

	os.Args = reorderArgs(os.Args)
	flag.Parse()

	if err := checkMode(flagPeek, flagSQL, flagSave, flagDB, flag.NArg()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	printer := output.NewPrinter(os.Stdout, os.Stderr, compact, verbose)

	if flagDB != "" {
		if err := runLocal(printer, flagDB, flagSQL, flagPeek); err != nil {
			printer.PrintError(err)
			os.Exit(1)
		}
		return
	}

	if flagSQL != "" {
		cfg, err := config.Load(flagToken, flagEnv, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: loading config: %v\n", err)
			os.Exit(1)
		}
		if cfg.Token == "" {
			fmt.Fprintln(os.Stderr, "error: token is required (set LOGFIRE_READ_TOKEN, use --token, use --env, or add to ~/.config/lfsnag/config.json)")
			os.Exit(1)
		}
		c := client.New(cfg.Token, cfg.BaseURL, printer)
		result, err := c.Query(flagSQL)
		if err != nil {
			printer.PrintError(err)
			os.Exit(1)
		}
		norm, err := logfire.Normalize(result)
		if err != nil {
			printer.PrintError(err)
			os.Exit(1)
		}
		printer.PrintRawJSON(norm)
		return
	}

	traceID, project := parseTraceRef(flag.Arg(0))
	if !traceIDPattern.MatchString(traceID) {
		fmt.Fprintf(os.Stderr, "error: invalid traceId %q (must be 32 hex characters)\n", flag.Arg(0))
		os.Exit(1)
	}

	cfg, err := config.Load(flagToken, flagEnv, project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: loading config: %v\n", err)
		os.Exit(1)
	}
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "error: token is required (set LOGFIRE_READ_TOKEN, use --token, use --env, or add to ~/.config/lfsnag/config.json)")
		os.Exit(1)
	}

	dbPath := flagSave
	if dbPath == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cache dir: %v\n", err)
			os.Exit(1)
		}
		dbPath = filepath.Join(base, "lfsnag", traceID+".sqlite")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: creating db dir: %v\n", err)
		os.Exit(1)
	}

	c := client.New(cfg.Token, cfg.BaseURL, printer)
	w, err := store.NewWriter(dbPath)
	if err != nil {
		printer.PrintError(err)
		os.Exit(1)
	}
	err = logfire.FetchEach(traceID, flagFields, 2000, c.Query, func(rows []map[string]any) error {
		_, err := w.Write(rows)
		return err
	})

	if err != nil {
		w.Abort()
		printer.PrintError(err)
		os.Exit(1)
	}
	if err := w.Commit(); err != nil {
		printer.PrintError(err)
		os.Exit(1)
	}

	p, note := peekDB(printer, dbPath)
	printer.PrintJSON(peekPayload(dbPath, p, note))

}

func peekDB(printer *output.Printer, dbPath string) (logfire.PeekResult, string) {
	rows, err := store.Query(dbPath, "SELECT "+logfire.PeekColumns+" FROM records")
	if err != nil {
		// A slim -f fetch can lack these columns. Print the record count only.
		cnt, cerr := store.Query(dbPath, "SELECT count(*) AS n FROM records")
		if cerr != nil {
			printer.PrintError(cerr)
			os.Exit(1)
		}
		n, _ := cnt[0]["n"].(int64)
		return logfire.PeekResult{N: int(n)}, "peek metrics unavailable: file lacks required columns (fetch without -f for a full peek)"
	}
	return logfire.Peek(rows), ""
}

func peekPayload(db string, p logfire.PeekResult, note string) any {
	if note != "" {
		return struct {
			DB   string `json:"db,omitempty"`
			N    int    `json:"n"`
			Note string `json:"note"`
		}{db, p.N, note}
	}
	type peekOut struct {
		DB   string `json:"db,omitempty"`
		Note string `json:"note,omitempty"`
		logfire.PeekResult
	}
	return peekOut{db, note, p}
}

func runLocal(printer *output.Printer, path, sqlQuery string, peek bool) error {
	if peek {
		p, note := peekDB(printer, path)
		return printer.PrintJSON(peekPayload("", p, note))
	}
	raw, err := store.QueryJSON(path, sqlQuery)
	if err != nil {
		return err
	}
	return printer.PrintRawJSON(raw)
}
