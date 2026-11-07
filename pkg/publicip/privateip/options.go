package privateip

import (
	"errors"
)

type Options struct{}

var (
	ErrInvalidOption = errors.New("invalid option for private IP retrieval")
)

// NewOptions returns default options for the private IP provider
func NewOptions() *Options {
	return &Options{}
}

// Validate checks the options (for private IP, there might be no specific options to validate)
func (o *Options) Validate() error {
	// No specific options to validate for private IP retrieval
	return nil
}
