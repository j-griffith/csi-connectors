package logger

import (
	"log"
	"os"
)

var (
	// Debug provides a handle to a std logger to use for Debug level events
	Debug *log.Logger
	// Info provides a handle to a std logger to use for Info level events
	Info *log.Logger
	// Warning provides a handle to a std logger to use for Warning level events
	Warning *log.Logger
	// Error provides a handle to a std logger to use for Error level events
	Error *log.Logger
)

func init() {
	// TODO: expose a callable init function to consumer so they can customize as we go
	// for now we're just setting up what we hope are "sane" defaults
	Info = log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(os.Stderr,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}
