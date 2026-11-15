package domain

import (
	"fmt"

	domainErrors "github.com/ToggyO/cocker/internal/domain/errors"
)

// EnvVar represents a process environment variable as a name-value pair.
type EnvVar struct {
	Name  string
	Value string
}

// EnvVars is an ordered list of process environment variables.
type EnvVars []EnvVar

// Validate appends violations for invalid or duplicate variable names to errors.
func (ev EnvVars) Validate(errors *domainErrors.ValidationErrors) {
	duplicatesMap := make(map[string]bool)

	for i, variable := range ev {
		if !isValidEnvName(variable.Name) {
			addError(errors, domainErrors.ValidationError{
				Field:   fmt.Sprintf("env[%d].name", i),
				Value:   variable.Name,
				Message: domainErrors.ValidationInvalidEnvName,
			})
		}

		if _, ok := duplicatesMap[variable.Name]; ok {
			addError(errors, domainErrors.ValidationError{
				Field:   fmt.Sprintf("env[%d].name", i),
				Value:   variable.Name,
				Message: domainErrors.ValidationNotUnique,
			})
			continue
		}

		duplicatesMap[variable.Name] = true
	}
}

func isValidEnvName(name string) bool {
	if name == "" {
		return false
	}

	for i, r := range name {
		if i == 0 && !isEnvStart(r) {
			return false
		}

		if !isEnvPart(r) {
			return false
		}
	}

	return true
}

func isEnvStart(r rune) bool {
	return r == '_' ||
		r >= 'A' && r <= 'Z' ||
		r >= 'a' && r <= 'z'
}

func isEnvPart(r rune) bool {
	return isEnvStart(r) ||
		r >= '0' && r <= '9'
}
