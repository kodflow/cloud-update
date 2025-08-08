package console

import "fmt"

// Print outputs to stdout for CLI interactions (help, version, etc.)
// This is intentionally separate from the logger which is for service operations.
// nolint:fmt - Console output is legitimate for CLI tools
func Print(a ...interface{}) {
	fmt.Print(a...)
}

// Println outputs to stdout with newline for CLI interactions
// nolint:fmt - Console output is legitimate for CLI tools
func Println(a ...interface{}) {
	fmt.Println(a...)
}

// Printf outputs formatted text to stdout for CLI interactions
// nolint:fmt - Console output is legitimate for CLI tools
func Printf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}
