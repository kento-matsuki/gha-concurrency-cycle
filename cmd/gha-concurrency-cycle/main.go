package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kentomk/gha-concurrency-cycle/internal/analyzer"
)

var version = "0.1.0-dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: gha-concurrency-cycle check [--format text|json] [--root PATH]")
		return 2
	}
	if args[0] == "version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if args[0] != "check" {
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}

	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "text", "output format: text or json")
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "check accepts only --format text|json and --root PATH")
		return 2
	}

	report, err := analyzer.Analyze(*root, version)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	} else {
		for _, diagnostic := range report.Diagnostics {
			fmt.Fprintf(stdout, "%s %s:%d -> %s:%d via %s:%d: effective concurrency group %q is held by the caller and requested by the called workflow; %s\n",
				diagnostic.RuleID,
				diagnostic.Caller.Path, diagnostic.Caller.Line,
				diagnostic.Callee.Path, diagnostic.Callee.Line,
				diagnostic.CallSite.Path, diagnostic.CallSite.Line,
				diagnostic.EffectiveGroup, diagnostic.Remediation,
			)
		}
	}
	if len(report.Diagnostics) > 0 {
		return 1
	}
	return 0
}
