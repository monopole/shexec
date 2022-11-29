package channeler

import (
	"fmt"
	"log"
	"os"
)

// VerboseLoggingEnabled can be set true to see detailed logging.
var VerboseLoggingEnabled = false

type logSink struct{}

func (l logSink) Write(p []byte) (n int, err error) {
	if //goland:noinspection GoBoolExpressions
	VerboseLoggingEnabled {
		return fmt.Fprint(os.Stderr, string(p))
	}
	return 0, nil
}

var logger = log.New(
	&logSink{}, "CHANNELER: ", log.Ldate|log.Ltime|log.Lshortfile)
