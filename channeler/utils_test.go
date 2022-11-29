package channeler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Make sure the end of the command is as expected.
func TestAssureTermination(t *testing.T) {
	var empty byte // intentionally left uninitialized
	const (
		semiColonCh = ';'
		hello       = "hello"
		semiColon   = string(semiColonCh)
		newLine     = string(NewLineCh)
	)
	testCases := map[string]struct {
		line     string
		term     byte
		expected string
	}{
		// FWIW: empty lines are empty lines (no semicolon).
		"t0": {
			line:     "",
			term:     semiColonCh,
			expected: newLine,
		},
		"t1": {
			line:     hello,
			term:     semiColonCh,
			expected: hello + semiColon + newLine,
		},
		"t2": {
			line:     hello + semiColon,
			term:     semiColonCh,
			expected: hello + semiColon + newLine,
		},
		"t3": {
			line:     hello + newLine,
			term:     semiColonCh,
			expected: hello + semiColon + newLine,
		},
		"t4": {
			line:     hello + semiColon + newLine,
			term:     semiColonCh,
			expected: hello + semiColon + newLine,
		},
		"t5": {
			line:     hello,
			term:     empty,
			expected: hello + newLine,
		},
		"t6": {
			line:     hello + semiColon,
			term:     empty,
			expected: hello + semiColon + newLine,
		},
		"t7": {
			line:     hello + newLine,
			term:     empty,
			expected: hello + newLine,
		},
		"t8": {
			line:     hello + semiColon + newLine,
			term:     empty,
			expected: hello + semiColon + newLine,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			assert.Equal(
				t, tc.expected, string(assureTermination(tc.line, tc.term)))
		})
	}
}
