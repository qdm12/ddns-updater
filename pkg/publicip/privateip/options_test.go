package privateip

import (
	"testing"
)

func TestNewOptions(t *testing.T) {
	t.Parallel()

	opts := NewOptions()
	if opts == nil {
		t.Fatalf("NewOptions() returned nil, expected non-nil *Options")
	}
}

func TestOptions_Validate(t *testing.T) {
	t.Parallel()

	opts := NewOptions()
	err := opts.Validate()
	if err != nil {
		t.Errorf("Options.Validate() returned error: %v, expected nil", err)
	}
}

func TestErrInvalidOption(t *testing.T) {
	t.Parallel()

	expectedMessage := "invalid option for private IP retrieval"
	if ErrInvalidOption.Error() != expectedMessage {
		t.Errorf("ErrInvalidOption message = %q, want %q", ErrInvalidOption.Error(), expectedMessage)
	}
}
