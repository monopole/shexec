package shexec

import (
	"fmt"
	"log"
)

// Sentinel holds a {command, value} pair.
//
// A Sentinel is used to recognize the end of command output on a stream.
// Examples:
//
//	Command: echo pink elephants dance
//	Value: pink elephants dance
//
//	Command: version
//	Value: v1.2.3
//
//	Command: rumpelstiltskin
//	Value: rumpelstiltskin: command not found
type Sentinel struct {
	// C is a command that should do very little, do it quickly,
	// and have deterministic, newline terminated output.
	C string

	// V is the expected value from Command.
	// Sentinel value comparisons are only made when a
	// newline is encountered in the output stream,
	// and then only working backwards from that newline.
	// E.g. the value "foo" will match "foo\n" in the
	// output stream, but will not match "foo bar".
	V string
}

const (
	// sentinelValueLenMin is used in Sentinel validation.
	// A Sentinel fails validation if the length of the sentinel
	// value is less than this.  The longer the sentinel value,
	// the less the chance of confusing it with valid output.
	sentinelValueLenMin = 6
	// sentinelValueLenRecommendedMin triggers a nagging message.
	// TODO: find better way to detect a bad sentinel value
	sentinelValueLenRecommendedMin = 12
	// enableSentinelNagging turns on sentinel nagging.
	enableSentinelNagging = false
)

// Validate returns an error if there's a problem in the Sentinel.
// This validation is critical; if a sentinel value is empty,
// the infrastructure will hang.
func (s *Sentinel) Validate() error {
	if s.C == "" {
		return fmt.Errorf("must specify Sentinel command")
	}
	if len(s.V) < sentinelValueLenMin {
		return fmt.Errorf(
			"sentinel value %q too short at len=%d; must be >= %d chars long",
			s.V, len(s.V), sentinelValueLenMin)
	}
	if //goland:noinspection GoBoolExpressions
	enableSentinelNagging && len(s.V) < sentinelValueLenRecommendedMin {
		log.Printf(
			"sentinel value %q very short at len == %d; recommend len >= %d",
			s.V, len(s.V), sentinelValueLenRecommendedMin)
	}
	return nil
}
