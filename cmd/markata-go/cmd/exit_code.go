package cmd

import (
	"errors"
	"strings"
)

const exitCodeUsage = 2

// ExitCodeError carries a process exit code without forcing command handlers to
// call os.Exit directly.
type ExitCodeError struct {
	code int
	err  error
}

func (e *ExitCodeError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *ExitCodeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *ExitCodeError) ExitCode() int {
	if e == nil || e.code == 0 {
		return 1
	}
	return e.code
}

func newExitCodeError(code int, err error) error {
	if code == 0 && err == nil {
		return nil
	}
	return &ExitCodeError{code: code, err: err}
}

func newUsageError(err error) error {
	if err == nil {
		return nil
	}
	return newExitCodeError(exitCodeUsage, err)
}

func ExitCodeForError(err error) int {
	if err == nil {
		return 0
	}

	exitCode := 1
	var exitErr interface{ ExitCode() int }
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	if isUsageError(err) {
		return exitCodeUsage
	}
	return exitCode
}

func isUsageError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		return false
	}

	usagePrefixes := []string{
		"accepts ",
		"unknown flag:",
		"unknown shorthand flag:",
		"unknown command ",
		"required flag(s) ",
		"flag needs an argument:",
		"requires at least ",
		"requires at most ",
		"requires between ",
		"invalid argument ",
	}

	for _, prefix := range usagePrefixes {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}

	return false
}
