package utils

import "fmt"

var Verbose bool

// SetVerbose sets the global verbosity level
func SetVerbose(v bool) {
	Verbose = v
}

// VerbosePrintf prints formatted output only if verbose mode is enabled
func VerbosePrintf(format string, args ...interface{}) {
	if Verbose {
		fmt.Printf(format, args...)
	}
}

// VerbosePrintln prints output only if verbose mode is enabled
func VerbosePrintln(args ...interface{}) {
	if Verbose {
		fmt.Println(args...)
	}
}

// VerbosePrint prints output only if verbose mode is enabled
func VerbosePrint(args ...interface{}) {
	if Verbose {
		fmt.Print(args...)
	}
}
