package channeler

import (
	"fmt"
	"log"
	"os"
)

// VerboseLoggingEnabled can be set true to see detailed logging.
var VerboseLoggingEnabled = false

type logSink struct{}

const AbbrevMaxLen = 65

func abbrev(x string) string {
	if len(x) > AbbrevMaxLen {
		return x[0:AbbrevMaxLen-1] + "..."
	}
	return x
}

func (l logSink) Write(p []byte) (n int, err error) {
	if //goland:noinspection GoBoolExpressions
	VerboseLoggingEnabled {
		return fmt.Fprint(os.Stderr, string(p))
	}
	return 0, nil
}

var logger = log.New(&logSink{}, "CHNLR: ", log.Ldate|log.Ltime|log.Lshortfile)
