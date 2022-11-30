package shexec

import (
	"fmt"
	"github.com/monopole/shexec/channeler"
	"log"
	"os"
)

// enableLogging can be set to true to see detailed logging.
var enableLogging = false

func abbrev(x string) string {
	if len(x) > channeler.AbbrevMaxLen {
		return x[0:channeler.AbbrevMaxLen-1] + "..."
	}
	return x
}

// VerboseLoggingEnable enables detailed logging.
func VerboseLoggingEnable() {
	enableLogging, channeler.VerboseLoggingEnabled = true, true
}

// VerboseLoggingDisable disables detailed logging.
func VerboseLoggingDisable() {
	enableLogging, channeler.VerboseLoggingEnabled = false, false
}

type logSink struct{}

func (l logSink) Write(p []byte) (n int, err error) {
	if enableLogging {
		return fmt.Fprint(os.Stderr, string(p))
	}
	return 0, nil
}

var logger = log.New(&logSink{}, "SHELL: ", log.Ldate|log.Ltime|log.Lshortfile)
