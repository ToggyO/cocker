package config

import (
	"errors"
	"fmt"
)

const (
	unitsErrorMessageTemplate = "units must be in %q"
)

var (
	ParsingErrorOverflow      = errors.New("value overflow") // TODO: сообщение надо переформулировать
	ParsingErrorEmpty         = errors.New("value must not be empty")
	ParsingErrorInvalidNumber = errors.New("value must be a valid number and not start with a '+' or '-' or other non digit symbols")
	ParsingErrorZeroValue     = errors.New("value must not be zero")
)

// ParseError describes a configuration parsing failure and optionally wraps its cause.
type ParseError struct {
	message string
	err     error
}

// NewParseError returns a parsing error with the given message and optional cause.
// The cause may be nil when there is no underlying error.
func NewParseError(message string, err error) *ParseError {
	return &ParseError{
		message: message,
		err:     err,
	}
}

func (err ParseError) Error() string {
	return fmt.Sprintf(
		"an error occurred while parsing input value:\n'%s'",
		err.message,
	)
}

func (err ParseError) Unwrap() error {
	return err.err
}
