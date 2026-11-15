package domainErrors

import (
	"fmt"
	"strings"
)

// ValidationError describes a violated rule for a domain field and its value.
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}

func (e ValidationErrors) Error() string {
	var builder strings.Builder

	builder.WriteString("validation errors:\n")

	for i, validationError := range e.Errors {
		if i > 0 {
			builder.WriteString(";\n")
		}

		_, _ = fmt.Fprintf(&builder, "%s", validationError.Field)

		if validationError.Value != "" {
			_, _ = fmt.Fprintf(&builder, " %q", validationError.Value)
		}

		_, _ = fmt.Fprintf(&builder, ": %s", validationError.Message)
	}

	return builder.String()
}
