package cmd

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
