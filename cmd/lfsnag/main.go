package main

import (
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
	"--project": true, "-project": true,
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
		flagProject string
	)

	flag.BoolVar(&compact, "c", false, "Compact JSON output")
	flag.BoolVar(&compact, "compact", false, "Compact JSON output")
	flag.BoolVar(&verbose, "v", false, "Show HTTP request/response details")
	flag.BoolVar(&verbose, "verbose", false, "Show HTTP request/response details")
	flag.StringVar(&flagToken, "token", "", "Override read token")
	flag.StringVar(&flagProject, "project", "", "Override project")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lfsnag [options] <traceId>\n\n")
		fmt.Fprintf(os.Stderr, "Fetch full trace details from Pydantic Logfire by traceId.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  lfsnag 019d05ee9be731d9f95c339fb7b9c6c1\n")
		fmt.Fprintf(os.Stderr, "  lfsnag -c 019d05ee9be731d9f95c339fb7b9c6c1\n")
		fmt.Fprintf(os.Stderr, "  lfsnag -v --token mytoken --project org/proj 019d05ee9be731d9f95c339fb7b9c6c1\n")
	}

	os.Args = reorderArgs(os.Args)
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: traceId is required")
		flag.Usage()
		os.Exit(1)
	}

	traceID := flag.Arg(0)
	if !traceIDPattern.MatchString(traceID) {
		fmt.Fprintf(os.Stderr, "error: invalid traceId %q (must be 32 hex characters)\n", traceID)
		os.Exit(1)
	}

	cfg, err := config.Load(flagToken, flagProject)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "error: token is required (set LOGFIRE_READ_TOKEN, use --token, or add to ~/.config/lfsnag/config.json)")
		os.Exit(1)
	}
	if cfg.Project == "" {
		fmt.Fprintln(os.Stderr, "error: project is required (set LOGFIRE_PROJECT, use --project, or add to ~/.config/lfsnag/config.json)")
		os.Exit(1)
	}

	printer := output.NewPrinter(os.Stdout, os.Stderr, compact, verbose)
	c := client.New(cfg.Token, cfg.Project, cfg.BaseURL, printer)

	result, err := c.QueryTrace(traceID)
	if err != nil {
		printer.PrintError(err)
		os.Exit(1)
	}

	printer.PrintRawJSON(result)
}
