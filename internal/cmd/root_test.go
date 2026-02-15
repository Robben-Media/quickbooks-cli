package cmd

import (
	"errors"
	"testing"
)

func TestExecute_Help(t *testing.T) {
	err := Execute([]string{"--help"})
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}
}

func TestExecute_Version(t *testing.T) {
	err := Execute([]string{"version"})
	if err != nil {
		t.Fatalf("expected no error for version, got: %v", err)
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	err := Execute([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.Code)
	}
}

func TestVersionString(t *testing.T) {
	result := VersionString()
	if result == "" {
		t.Error("VersionString() should not be empty")
	}
}

func TestBoolString(t *testing.T) {
	if boolString(true) != "true" {
		t.Error("boolString(true) should be 'true'")
	}

	if boolString(false) != "false" {
		t.Error("boolString(false) should be 'false'")
	}
}

func TestEnvOr(t *testing.T) {
	result := envOr("NONEXISTENT_VAR_12345", "fallback")
	if result != "fallback" {
		t.Errorf("envOr should return fallback, got %q", result)
	}
}

func TestExitError_Error(t *testing.T) {
	err := &ExitError{Code: 1, Err: errors.New("test error")}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got %q", err.Error())
	}
}

func TestExitError_Nil(t *testing.T) {
	var err *ExitError

	if err.Error() != "" {
		t.Errorf("nil ExitError.Error() should be empty, got %q", err.Error())
	}

	if err.Unwrap() != nil {
		t.Error("nil ExitError.Unwrap() should return nil")
	}
}

func TestExitError_NoInnerError(t *testing.T) {
	err := &ExitError{Code: 42}
	expected := "exit code 42"

	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
