package channeler

import (
	"fmt"
	"log"
	"os"
)

// VerboseLoggingEnabled can be set true to see detailed logging.
// nolint:gochecknoglobals
var VerboseLoggingEnabled = false

type logSink struct{}

const AbbrevMaxLen = 70

func abbrev(x string) string {
	if len(x) > AbbrevMaxLen {
		return x[0:AbbrevMaxLen-1] + "..."
	}
	return x
}

func (l logSink) Write(p []byte) (n int, err error) {
	if VerboseLoggingEnabled {
		//nolint:wrapcheck
		return fmt.Fprint(os.Stderr, string(p))
	}
	return 0, nil
}

// nolint:gochecknoglobals
var logger = log.New(&logSink{}, "CHNLR: ", log.Ldate|log.Ltime|log.Lshortfile)

const errCategory = "channeler"

func paramErr(format string, a ...any) error {
	//nolint:goerr113
	return fmt.Errorf("%s: %s", errCategory, fmt.Sprintf(format, a...))
}

func paramErrCaused(err error, format string, a ...any) error {
	return fmt.Errorf(
		"%s: %s; %w", errCategory, fmt.Sprintf(format, a...), err)
}
