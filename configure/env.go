package configure

import "os"

var Pid = os.Getpid()

// Error code
const (
	InitConfigError = iota + 1
	InitLogError
)

// Error message
const (
	UnknownTypeMsg = "unknown type"
)
