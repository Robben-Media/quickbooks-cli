package errfmt

import (
	"errors"
	"os"
	"testing"

	"github.com/99designs/keyring"
)

func TestFormat_NilError(t *testing.T) {
	result := Format(nil)
	if result != "" {
		t.Errorf("Format(nil) = %q, want empty string", result)
	}
}

func TestFormat_KeyNotFound(t *testing.T) {
	result := Format(keyring.ErrKeyNotFound)
	expected := "Secret not found in keyring. Run: quickbooks-cli auth set-credentials"

	if result != expected {
		t.Errorf("Format(ErrKeyNotFound) = %q, want %q", result, expected)
	}
}

func TestFormat_NotExist(t *testing.T) {
	err := os.ErrNotExist
	result := Format(err)

	if result != err.Error() {
		t.Errorf("Format(ErrNotExist) = %q, want %q", result, err.Error())
	}
}

func TestFormat_UserFacingError(t *testing.T) {
	err := NewUserFacingError("friendly message", errors.New("internal cause"))
	result := Format(err)

	if result != "friendly message" {
		t.Errorf("Format(UserFacingError) = %q, want 'friendly message'", result)
	}
}

func TestFormat_GenericError(t *testing.T) {
	err := errors.New("something failed")
	result := Format(err)

	if result != "something failed" {
		t.Errorf("Format(generic) = %q, want 'something failed'", result)
	}
}

func TestUserFacingError_Nil(t *testing.T) {
	var err *UserFacingError

	if err.Error() != "" {
		t.Errorf("nil UserFacingError.Error() = %q, want empty", err.Error())
	}

	if err.Unwrap() != nil {
		t.Error("nil UserFacingError.Unwrap() should return nil")
	}
}

func TestUserFacingError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := &UserFacingError{Message: "msg", Cause: cause}

	if !errors.Is(err, cause) {
		t.Error("UserFacingError should unwrap to cause")
	}
}
