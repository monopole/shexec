package internal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Program name and flags needed for tests.
const (
	FlagDisablePrompt = "disable-prompt"
	FlagExitOnErr     = "exit-on-error"
	FlagFailOnStartup = "fail-on-startup"
	FlagNumRowsInDb   = "num-rows-in-db"
	FlagRowToErrorOn  = "row-to-error-on"
)

// All individual commands.
const (
	CmdHelp    = "help"
	CmdQuit    = "quit"
	CmdEcho    = "echo"
	cmdVersion = "version"
	CmdSleep   = "sleep"
	cmdStatus  = "status"
	cmdSet     = "set"
	cmdPrint   = "print"
	CmdQuery   = "query"
)

// AllCommands can be used in help and validation.
var AllCommands = []string{
	CmdHelp,
	CmdSleep,
	cmdVersion,
	CmdQuit,
	CmdEcho,
	cmdStatus,
	cmdSet,
	cmdPrint,
	CmdQuery,
}

// Other constants.
//
//goland:noinspection SpellCheckingInspection
const (
	scanQueryWord    = "query"
	lookupQueryWord  = "bus"
	versionOfProgram = "v1.2.3"
)

// Shell parses stdin, executing anything that validates as a command.
// See AllCommands for a complete list of recognized commands.
// This code exists to drive tests of the cli package.
type Shell struct {
	promptCount   int
	disablePrompt bool
	exitOnError   bool
	stdOut        io.Writer
	stdErr        io.Writer
	scanner       *bufio.Scanner
	db            *SillyDb
	help          string
}

// NewShell returns a new instance.
func NewShell(db *SillyDb, disablePrompt, exitOnError bool, help string) *Shell {
	return &Shell{
		promptCount:   0,
		stdOut:        os.Stdout, // make into an arg for redirect?
		stdErr:        os.Stderr, // make into an arg for redirect?
		disablePrompt: disablePrompt,
		exitOnError:   exitOnError,
		scanner:       bufio.NewScanner(os.Stdin),
		db:            db,
		help:          help,
	}
}

// Run starts a loop to drain the shells input stream, executing commands.
//
//goland:noinspection GoUnhandledErrorResult
func (s *Shell) Run() error {
	s.maybeShowPrompt()
	for s.scanner.Scan() {
		done, err := s.handleCommand(normalizeCommand(s.scanner.Text()))
		if err != nil {
			fmt.Fprintln(s.stdErr, err.Error())
			if s.exitOnError {
				return err
			}
		}
		if done {
			return nil
		}
		s.maybeShowPrompt()
	}
	return nil
}

func (s *Shell) maybeShowPrompt() {
	if !s.disablePrompt {
		s.promptCount++
		fmt.Fprint(s.stdOut, s.makePrompt())
	}
}

func (s *Shell) makePrompt() string {
	return fmt.Sprintf("hey<%d>", s.promptCount)
}

func (s *Shell) handleCommand(cmd string) (done bool, err error) {
	if cmd == "" {
		// Ignore empty commands.
		return
	}
	if cmd == CmdQuit {
		// All done.
		return true, nil
	}
	if cmd == CmdHelp {
		fmt.Fprintf(s.stdOut, "Commands: %v\n", AllCommands)
		fmt.Fprintf(s.stdOut, s.help)
		return
	}
	if cmd == cmdVersion {
		fmt.Fprintln(s.stdOut, versionOfProgram)
		return
	}
	if strings.HasPrefix(cmd, CmdSleep+" ") {
		var d time.Duration
		d, err = time.ParseDuration(cmd[len(CmdSleep)+1:])
		if err != nil {
			return
		}
		// For use in tests. Simulate a long-running command.
		<-time.After(d)
		return
	}
	if cmd == cmdStatus {
		fmt.Fprintf(s.stdOut,
			"numRowsInDb = %d rowToErrorOn = %d\n",
			s.db.NumRowsInDb(), s.db.RowToErrorOn())
		return
	}
	if strings.HasPrefix(cmd, CmdEcho+" ") {
		fmt.Fprintln(s.stdOut, cmd[len(CmdEcho)+1:])
		return
	}
	if strings.HasPrefix(cmd, cmdSet+" ") {
		// Ignore set command, but don't error on it.  Emulates a real command.
		return
	}
	if strings.HasPrefix(cmd, cmdPrint+" ") {
		// Do a lookup.
		var id string
		id, err = parseLookupQuery(cmd)
		if err != nil {
			return
		}
		return false, s.db.DoLookupQuery(id)
	}
	if strings.Contains(cmd, CmdQuery) {
		// Try a query.
		var offset, limit int
		offset, limit, err = parseScanQuery(cmd)
		if err != nil {
			return
		}
		return false, s.db.DoScanQuery(offset, limit)
	}
	return false, fmt.Errorf("unrecognized command: %q", cmd)
}

func normalizeCommand(c string) string {
	// strip trailing semi-colon.
	if len(c) > 0 && c[len(c)-1] == ';' {
		c = c[:len(c)-1]
	}
	return c
}

// parseLookupQuery looks for "print bus AE000F"
// and returns the id or error
func parseLookupQuery(cmd string) (string, error) {
	i := strings.Index(cmd, lookupQueryWord)
	if i < 0 {
		return "", fmt.Errorf("unrecognized command %q", cmd)
	}
	return strings.TrimSpace(cmd[i+len(lookupQueryWord):]), nil
}

// parseScanQuery looks for "blah query blah offset 200 blah limit 10 whatever"
// and returns the offset and limit as integers, else error.
func parseScanQuery(cmd string) (offset, limit int, err error) {
	i := strings.Index(cmd, scanQueryWord)
	if i < 0 {
		return 0, 0, fmt.Errorf("unrecognized command %q", cmd)
	}
	cmd = cmd[i+len(scanQueryWord):]
	args := strings.Split(strings.TrimSpace(cmd), " ")
	offset, err = getIntArg(args, "offset")
	if err != nil {
		return 0, 0, err
	}
	limit, err = getIntArg(args, "limit")
	if err != nil {
		return 0, 0, err
	}
	return
}

func getIntArg(words []string, name string) (result int, err error) {
	i := findInSlice(words, name)
	if i < 0 || i >= len(words)-1 {
		return -1, nil
	}
	result, err = strconv.Atoi(words[i+1])
	if err != nil {
		return -1, fmt.Errorf("offset %q not a number\n", words[i+1])
	}
	return
}

func findInSlice(words []string, word string) int {
	for i := 0; i < len(words); i++ {
		if words[i] == word {
			return i
		}
	}
	return -1
}
