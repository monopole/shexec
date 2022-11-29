package scripter

import (
	"time"
)

// Shell manages a shell program, adding value by allowing the output
// from different commands to be handled differently.
//
// A Shell is in one of these states:
//
// off: no shell subprocess running.
//
//   - Shell freshly created, Start not yet called.
//   - Stop called and finished.
//   - An error encountered in any call meaning that the
//     subprocess had to be abandoned (must call Start again).
//   - Ok to Start, but not Run or Stop.
//
// idle: shell subprocess healthy and awaiting input.
//
//   - A call to Start finished without error.
//   - A call to Run finished without error.
//   - Ok to call Run or Stop, but not Start.
//
// All Shell calls block until they finish or their deadlines expire.
type Shell interface {
	// Start synchronously starts the shell.
	// It assures that the shell runs and that the sentinels work
	// before their first use in the Run method.
	// Errors:
	// * The shell was already started.
	// * Something's wrong in the Parameters, e.g. the shell program
	//   cannot be found.
	// * The sentinels failed to work in the time allotted.
	Start(time.Duration) error

	// Run sends the command in Commander to the shell, and
	// waits for it to complete.  It returns an error if
	// there was some infrastructure problem or if the
	// command timed out because no sentinels were detected
	// in the time given.
	// An error here means that the shell is dead, and in
	// need of fresh call to Start.
	// Errors:
	// * The shell hasn't been started.
	// * The command timed out.
	// * The shell exited, regardless of exit code.
	Run(time.Duration, Commander) error

	// Stop attempts to gracefully stop the shell.
	// It sends the given command to the shell (presumably something
	// like `quit` or `exit`), or just EOF if the command is empty.
	// Stop, unlike Run, treats the shell exiting with a zero status
	// as a success.
	// Errors:
	// * The shell wasn't started or is currently running.
	// * The shell's subprocess didn't finish in the time allotted.
	// * The shell exited with non-zero status.
	Stop(time.Duration, string) error
}
