package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bigbag/lfsnag/internal/client"
	"github.com/bigbag/lfsnag/internal/config"
	"github.com/bigbag/lfsnag/internal/output"
)

var traceIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

var flagsWithValues = map[string]bool{
	"--token": true, "-token": true,
	"--env": true, "-env": true, "-e": true,
	"--fields": true, "-fields": true, "-f": true,
	"--sql": true, "-sql": true,
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

func main() {
	var (
		compact    bool
		verbose    bool
		flagToken  string
		flagEnv    string
		flagFields string
		flagSQL    string
	)

	flag.BoolVar(&compact, "c", false, "Compact JSON output")
	flag.BoolVar(&compact, "compact", false, "Compact JSON output")
	flag.BoolVar(&verbose, "v", false, "Show HTTP request/response details")
	flag.BoolVar(&verbose, "verbose", false, "Show HTTP request/response details")
	flag.StringVar(&flagToken, "token", "", "Override read token")
	flag.StringVar(&flagEnv, "e", "", "Environment profile name")
	flag.StringVar(&flagEnv, "env", "", "Environment profile name")
	flag.StringVar(&flagFields, "f", "", "Comma-separated list of fields to select (default: all)")
	flag.StringVar(&flagFields, "fields", "", "Comma-separated list of fields to select (default: all)")
	flag.StringVar(&flagSQL, "sql", "", "Raw SQL query to execute against the Logfire API")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lfsnag [options] <traceId>\n")
		fmt.Fprintf(os.Stderr, "       lfsnag [options] --sql \"<query>\"\n\n")
		fmt.Fprintf(os.Stderr, "Fetch trace details from Pydantic Logfire by traceId or raw SQL.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  lfsnag 019d05ee9be731d9f95c339fb7b9c6c1\n")
		fmt.Fprintf(os.Stderr, "  lfsnag -e prod 019d05ee9be731d9f95c339fb7b9c6c1\n")
		fmt.Fprintf(os.Stderr, "  lfsnag -c 019d05ee9be731d9f95c339fb7b9c6c1\n")
		fmt.Fprintf(os.Stderr, "  lfsnag -v --token mytoken 019d05ee9be731d9f95c339fb7b9c6c1\n")
		fmt.Fprintf(os.Stderr, "  lfsnag -e dev --sql \"SELECT span_name, duration FROM records WHERE is_exception\"\n")
	}

	os.Args = reorderArgs(os.Args)
	flag.Parse()

	if flagSQL != "" && flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "error: --sql and traceId are mutually exclusive")
		flag.Usage()
		os.Exit(1)
	}

	if flagSQL == "" && flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: traceId or --sql is required")
		flag.Usage()
		os.Exit(1)
	}

	var traceID string
	if flagSQL == "" {
		traceID = flag.Arg(0)
		if !traceIDPattern.MatchString(traceID) {
			fmt.Fprintf(os.Stderr, "error: invalid traceId %q (must be 32 hex characters)\n", traceID)
			os.Exit(1)
		}
	}

	cfg, err := config.Load(flagToken, flagEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "error: token is required (set LOGFIRE_READ_TOKEN, use --token, use --env, or add to ~/.config/lfsnag/config.json)")
		os.Exit(1)
	}

	printer := output.NewPrinter(os.Stdout, os.Stderr, compact, verbose)
	c := client.New(cfg.Token, cfg.BaseURL, printer)

	var result json.RawMessage
	if flagSQL != "" {
		result, err = c.Query(flagSQL)
	} else {
		result, err = c.QueryTrace(traceID, flagFields)
	}
	if err != nil {
		printer.PrintError(err)
		os.Exit(1)
	}

	printer.PrintRawJSON(result)
}
