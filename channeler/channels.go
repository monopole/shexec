package channeler

// Channels holds a shell's input and output channels.
type Channels struct {
	// StdIn accepts command lines. A "command line" is opaque;
	// it might be complex, multi-line script, like a bash here-doc.
	// It's sent to the shell without any processing other than the
	// addition of a terminating NewLine if one isn't present.
	// Close StdIn to initiate graceful shutdown of the shell.
	StdIn chan<- string
	// Block on this after closing StdIn to assure that
	// everything finishes without an error.  An error here
	// is usually a timeout.  An error here has nothing to do
	// with the content of StdErr; the latter is merely another
	// output stream from the subprocess.
	Done <-chan error
	// StdOut provides lines from stdout with NewLine removed.
	StdOut <-chan string
	// StdErr provides lines from stderr with NewLine removed.
	StdErr <-chan string
}
