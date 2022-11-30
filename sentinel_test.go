package shexec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentinel_Validate(t *testing.T) {
	s := Sentinel{}
	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must specify Sentinel command")

	s.C = "whatever"
	err = s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short at len=0")

	// Fell off cliff.
	s.V = strings.Repeat("A", sentinelValueLenMin) + "h!"
	err = s.Validate()
	assert.NoError(t, err)
}
