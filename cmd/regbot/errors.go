package main

import "fmt"

// usageError marks configuration/validation failures that occur before any
// automation is attempted. It maps to exit code 1.
type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }

// usageErrorf builds a usageError from a format string.
func usageErrorf(format string, args ...any) error {
	return usageError{err: fmt.Errorf(format, args...)}
}
