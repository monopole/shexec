package scripter_test

import (
	"testing"

	. "github.com/monopole/shexec/scripter"
	"github.com/stretchr/testify/assert"
)

func TestParameters_Validate(t *testing.T) {
	p := Parameters{}
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(
		t, err.Error(), "must specify Path to the executable to run")

	p.Path = "/whatever"
	err = p.Validate()
	assert.Error(t, err)
	assert.Contains(
		t, err.Error(), "path \"/whatever\" not available; exit status 127")

	p.Path = "/bin/sh"
	err = p.Validate()
	assert.Error(t, err)
	assert.Contains(
		t, err.Error(), "problem in SentinelOut; must specify Sentinel command")

	p.SentinelOut = Sentinel{
		C: "echo " + unlikelyWord,
	}
	err = p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "problem in SentinelOut;")
	assert.Contains(t, err.Error(), " must be >=")

	p.SentinelOut.V = unlikelyWord
	err = p.Validate()
	assert.NoError(t, err)
}
