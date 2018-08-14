package configure

import "os"

var Pid = os.Getpid()

// Error code
const (
	KInitConfigError = iota + 1
	KInitLogError
)

// Error message
const (
	KUnknownTypeMsg = "unknown type"
)
