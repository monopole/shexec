package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"

	"github.com/monopole/shexec/conch/internal"
)

//go:embed README.md
var readMeMd string

type argSack struct {
	numRowsInDb   int
	rowToErrorOn  int
	disablePrompt bool
	exitOnError   bool
	failOnStartup bool
}

// main reads commands from stdin, pretending to be a database frontend CLI.
//
//goland:noinspection GoUnhandledErrorResult
func main() {
	var args argSack
	flag.IntVar(
		&args.rowToErrorOn,
		internal.FlagRowToErrorOn, 0,
		"Error if this row number is in the results.")
	flag.IntVar(
		&args.numRowsInDb,
		internal.FlagNumRowsInDb, 50,
		"Maximum number of rows in the fake db.")
	flag.BoolVar(
		&args.disablePrompt,
		internal.FlagDisablePrompt, false,
		"Disable the prompt.")
	flag.BoolVar(
		&args.exitOnError,
		internal.FlagExitOnErr, false,
		"Exit on error, else continue accepting commands.")
	flag.BoolVar(
		&args.failOnStartup,
		internal.FlagFailOnStartup, false,
		"Exit with error on startup, before processing any commands.")
	flag.Parse()
	if len(flag.Args()) > 0 {
		if flag.Args()[0] != internal.CmdHelp {
			fmt.Fprintln(os.Stderr, "unrecognized args: ", flag.Args())
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, readMeMd)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Commands: %v\n", internal.AllCommands)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if args.failOnStartup {
		fmt.Fprintln(os.Stderr, "Ordered to fail on startup.")
		os.Exit(1)
	}
	shell := internal.NewShell(
		internal.NewSillyDb(args.numRowsInDb, args.rowToErrorOn),
		args.disablePrompt,
		args.exitOnError,
		readMeMd,
	)
	if err := shell.Run(); err != nil {
		// Assume error was already printed.
		os.Exit(1)
	}
}
