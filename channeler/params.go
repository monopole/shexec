package channeler

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Params captures all parameters to channeler.Start.
// It's a mix of subprocess parameters, like Path and Args,
// and orchestration parameters like buffer sizes and timeouts.
type Params struct {
	// Path is either the absolute path to the executable, or a $PATH
	// relative command name.  This is the shell being run.
	Path string

	// Args has the arguments, flags and flag arguments for the
	// shell invocation.
	Args []string

	// WorkingDir is the working directory of the shell process.
	WorkingDir string

	// CommandTerminator, if not 0, is appended to the end of every command.
	// This is a convenience for shells like mysql that want such things.
	// Example: ';'
	CommandTerminator byte

	// BuffSizeIn is how many commands can be added before
	// StdIn will block.
	BuffSizeIn int

	// ChTimeoutIn is how long to wait for a command from stdIn
	// before we fail, suspecting a deadlock or some other irreconcilable
	// problem in the client.  If you have a command sequence A, B, and you
	// know command A will take 2 hours to run, then this value should
	// exceed 2 hours, to avoid a timeout waiting to receive command B.
	// An expired timeout results in an error on the Done channel.
	// This can be long, and maybe it should be removed completely
	// (effectively treated as infinite).
	// Increasing BuffSizeIn doesn't help here.
	ChTimeoutIn time.Duration

	// BuffSizeOut is how many lines of output can be accepted
	// from the shell's stdout before back pressure is applied,
	// forcing the shell to wait before its output is consumed.
	BuffSizeOut int

	// BuffSizeErr is like BuffSizeOut, except for stderr.
	BuffSizeErr int

	// InfraConsumerTimeout is how long to wait for data from stdOut
	// or stdErr to be consumed by the infrastructure (the parser).
	// This is effectively an exit hatch if the parser or
	// some other aspect of the infrastructure (not the subprocess)
	// is taking too long.
	// To avoid timeouts, increase buffer BuffSizeOut and/or
	// BuffSizeErr and/or consume output faster.
	// This is an infrastructure parameter, not really meant
	// for use by a client.
	InfraConsumerTimeout time.Duration
}

const (
	defaultBuffSizeIn  = 100
	defaultBuffSizeOut = 10000
	defaultBuffSizeErr = 100

	// make this value interesting so that it's easy to spot.
	// It's not clear that there's any reason for this to be long.
	defaultInfraConsumerTimeout = 7777 * time.Millisecond

	// This, however, can be long, and maybe it should be removed completely
	// effectively treated as infinite.
	defaultChTimeoutIn = 12 * time.Hour
)

func (p *Params) Validate() error {
	p.setDefaults()
	if err := p.validateWorkDir(); err != nil {
		return err
	}
	return p.validatePath()
}

func (p *Params) setDefaults() {
	if p.BuffSizeIn < 1 {
		p.BuffSizeIn = defaultBuffSizeIn
	}
	if p.BuffSizeOut < 1 {
		p.BuffSizeOut = defaultBuffSizeOut
	}
	if p.BuffSizeErr < 1 {
		p.BuffSizeErr = defaultBuffSizeErr
	}
	if p.InfraConsumerTimeout == 0 {
		p.InfraConsumerTimeout = defaultInfraConsumerTimeout
	}
	if p.ChTimeoutIn == 0 {
		p.ChTimeoutIn = defaultChTimeoutIn
	}
}

func (p *Params) validateWorkDir() (err error) {
	p.WorkingDir, err = filepath.Abs(p.WorkingDir)
	if err != nil {
		return paramErrCaused(err, "bad working dir path")
	}
	var info os.FileInfo
	info, err = os.Stat(p.WorkingDir)
	if err != nil {
		return paramErrCaused(err, "bad working dir stat")
	}
	if !info.IsDir() {
		return paramErr("%q is not a directory that exists", p.WorkingDir)
	}
	return nil
}

func (p *Params) validatePath() (err error) {
	if p.Path == "" {
		return paramErr("must specify Path to the executable to run")
	}
	return errIfNoCommand(p.Path)
}

func errIfNoCommand(name string) error {
	cmd := exec.Command("/bin/sh", "-c", "command -v "+name)
	if err := cmd.Run(); err != nil {
		return paramErrCaused(err, "path %q not available", name)
	}
	return nil
}
