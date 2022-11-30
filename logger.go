package shexec

import (
	"fmt"
	"github.com/monopole/shexec/channeler"
	"log"
	"os"
)

// enableLogging can be set to true to see detailed logging.
var enableLogging = false

// VerboseLoggingEnable enables detailed logging.
func VerboseLoggingEnable() {
	enableLogging = true
	channeler.VerboseLoggingEnabled = true
}

// VerboseLoggingDisable disables detailed logging.
func VerboseLoggingDisable() {
	enableLogging = false
	channeler.VerboseLoggingEnabled = false
}

type logSink struct{}

func (l logSink) Write(p []byte) (n int, err error) {
	if enableLogging {
		return fmt.Fprint(os.Stderr, string(p))
	}
	return 0, nil
}

var logger = log.New(&logSink{}, "SCRIPTER: ", log.Ldate|log.Ltime|log.Lshortfile)
