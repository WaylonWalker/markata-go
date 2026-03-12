package cmd

import "testing"

func TestRunInitCommand_NoInputFails(t *testing.T) {
	originalNoInput := noInput
	defer func() { noInput = originalNoInput }()

	noInput = true
	err := runInitCommand(initCmd, nil)
	if err == nil {
		t.Fatal("expected error when --no-input is set")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}
