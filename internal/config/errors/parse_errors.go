package configErrors

import (
	"errors"
)

var (
	ParsingErrorOverflow      = errors.New("value exceeds uint64 range")
	ParsingErrorEmpty         = errors.New("value must not be empty")
	ParsingErrorInvalidNumber = errors.New("value must contain only decimal digits")
	ParsingErrorZeroValue     = errors.New("value must be greater than zero")
)

// ParseError describes a configuration parsing failure and optionally wraps its cause.
type ParseError struct {
	message string
	err     error
}

// NewParseError returns a parsing error with the given message and optional cause.
// The cause may be nil when there is no underlying error.
func NewParseError(message string, cause error) *ParseError {
	return &ParseError{
		message: message,
		err:     cause,
	}
}

func (e ParseError) Error() string {
	if e.err == nil {
		return e.message
	}
	if e.message == "" {
		return e.err.Error()
	}
	return e.message + ": " + e.err.Error()
}

func (e ParseError) Unwrap() error {
	return e.err
}
