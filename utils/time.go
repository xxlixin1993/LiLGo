package utils

import "time"

const MicTimeFormat = "2006/01/02 15:04:05.000000"

// Get a formatted Microseconds time
func GetMicTimeFormat() string {
	return time.Now().Format(MicTimeFormat)
}
