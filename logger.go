package shexec

import (
	"fmt"
	"log"
	"os"

	"github.com/monopole/shexec/channeler"
)

// verboseLoggingEnabled can be set to true to see detailed logging.
// nolint:gochecknoglobals
var verboseLoggingEnabled = true

func abbrev(x string) string {
	if len(x) > channeler.AbbrevMaxLen {
		return x[0:channeler.AbbrevMaxLen-1] + "..."
	}
	return x
}

type logSink struct{}

func (l logSink) Write(p []byte) (n int, err error) {
	if verboseLoggingEnabled {
		//nolint:wrapcheck
		return fmt.Fprint(os.Stderr, string(p))
	}
	return 0, nil
}

// nolint:gochecknoglobals
var lgr = log.New(&logSink{}, "SHELL: ", log.Ldate|log.Ltime|log.Lshortfile)
