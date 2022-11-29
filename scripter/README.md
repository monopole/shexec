[`Commander`]: ./commander.go
[`Executor`]: ./executor.go
[`Sentinel`]: ./sentinel.go

# scripter [![Go Report Card](https://goreportcard.com/badge/github.com/monopole/shexec/scripter)](https://goreportcard.com/report/github.com/monopole/shexec/scripter) [![Go Reference](https://pkg.go.dev/badge/github.com/monopole/shexec/scripter)](https://pkg.go.dev/github.com/monopole/shexec/scripter)

Package `scripter` lets one script a command
shell in Go as if a human were running it.

The package separates the problem of orchestrating shell
execution from the problem of generating a shell command and
parsing the shell's response to said command.

This package hides and solves the first problem (via [`Shell`]),
and makes the latter easy to do via Go
implementations of [`Commander`].

## Usage

Roughly:
```go
sh := NewShell(Parameters{
	ChParams: channeler.ChParams{Path: "/bin/sh"},
	SentinelOut: Sentinel{
		C: "echo " + unlikelyWord,
		V: unlikelyWord,
	},
})
assertNoErr(sh.Start(timeOut))
assertNoErr(sh.Run(timeOut, commander1))
// consult commander1 getters for whatever purpose,
// optionally use the results to define commander2.
assertNoErr(sh.Run(timeOut, commander2))
// consult commander2, etc.
assertNoErr(sh.Stop(timeOut, ""))
```

* [`example_test.go`](./example_test.go)
* [`shell_test.go`](./shell_test.go)

## Assumptions 

### Shell behavior

A _shell_ is any program that accepts newline terminated
commands, e.g. `bash`, and emits lines of output on
`stdOut` and `stdErr`.

The purpose of a shell, as opposed to a single-purpose
program that doesn't prompt for commands,
is to allow state that endures over multiple commands.

State contains things like authentication, authorization,
secrets obtained from vaults,
caches, database connections, etc.

A shell lets a user pay to build that state once,
then run many commands in the context of that state.

### Commands influence commands

There must be an opportunity to examine the
output of command _n_ before issuing command _n+1_.

The choice of command _n+1_ or its arguments
may be influenced by the output of command _n_.

### Command generation and parsing best live together

The code that _parses_ a command's output should live
close to the code that _generates_ the command.
The parser should have access to command
arguments and flags so that it knows what's
supposed to happen.

All a Go author need do is implement the [`Commander`]
interface, then pass instances of the implementation to
the `Run` method of a [`Shell`]. When a `Run` call
returns, the `Commander` instance can be consulted.
A commander can offer any number of methods yielding
validated data acquired from the shell; it can be 
viewed as a shell visitor.

A `Commander` can be tested in isolation
(without the need of a shell)
for its ability to compose a command and parse the output
expected from that command.

### Unreliable prompts, unreliable newlines, and command blocks

A human knows that a shell has completed command _n_
and is awaiting command _n+1_ because they
see a prompt following the output of command _n_.
Usually, but not always, the prompt
is on a new line.

But in a scripting context, prompts with newlines
are unreliable.

When running a shell as a subprocess,
e.g. as part of a pipe, the shell can see
that `stdIn` is not a `tty`, and won't
issue a prompt to avoid pipe contamination.

Sometimes command output can accidentally
contain data that matches the prompt,
making the prompt useless as an output delimiter.

Sometimes a shell will intentionally suppress
newline on command completion, e.g. `base64 -d`, `echo -n`.

Most importantly, sometimes a user wants to inject a
_command block_, multiple commands with embedded
newlines, as a single unit, not caring to know when
individual commands in the block finish.
Only the whole set matters.
This can happen when blindly executing command blocks
from some unknown source,
e.g. fenced code blocks embedded in markdown documentation.

For these reasons, a `Shell` cannot
depend on prompts and newlines to unambiguously
distinguish the data from commands
_n-1_, _n_ and _n+1_ on `stdOut` and `stdErr`.

So instead of relying on prompts or newlines,
[`Shell`] relies on a [`Sentinel`].

### Sentinels

#### stdOut

A `Shell` demands the existence of
a _sentinel command_ for `stdOut`.

Such a command
* does very little,
* does it quickly,
* has deterministic, newline terminated output on `stdOut`.

Example:

> ```
> $ echo "rumpelstiltskin"
> rumpelstiltskin
> ```

Commands that print a program's version, a help message,
and/or a copyright message are good candidates for
sentinel commands on the `stdOut` stream.

The unambiguously recognizable output of a sentinel command
called the _sentinel value_.

A [`Sentinel`] holds a `{command, value}` pair.

#### stdErr

Likewise, a `Shell` needs a sentinel
command for `stdErr`.

This command differs from the `stdOut` sentinel only
in that its output goes to `stdErr`.

Usually a shell will complain to `stdErr` if it sees
a command it doesn't recognize, meaning that
an unrecognized command is also a good sentinel
command for `stdErr`.

Example:
> ```
> $ rumpelstiltskin
> rumpelstiltskin: command not found
> ```

### Command results

The outcome of asking a shell to run a command is
one of the following:

* _crash_ - shell exits with non-zero status.
* _exit_ - shell exits with zero status.\
  If this happens unintentionally, it's treated as a crash.
* _timeout_ - shell fails to finish the command in a given time period.\
  The shell is assumed to be unusable, and should be killed.
* _ready_ - shell runs the command within the given time period and is ready to accept
  another command.\
  The command can be consulted for whatever
  results it parsed and saved.


