package channeler

const newLineChar = '\n'

// assureTermination assures correct command line termination.
// The line will always end with newline, but before that there
// might also be something line a semicolon.
func assureTermination(line string, terminator byte) []byte {
	c := []byte(line)
	if len(c) == 0 {
		// TODO: treat as an error?
		return []byte{newLineChar}
	}
	if c[len(c)-1] == newLineChar {
		// Slice it off avoid confusion; will replace it momentarily.
		// This doesn't change the Cap() of the slice.
		c = c[:len(c)-1]
	}
	if terminator > 0 && c[len(c)-1] != terminator {
		c = append(c, terminator)
	}
	// Always, always end with a newLine.
	return append(c, newLineChar)
}
