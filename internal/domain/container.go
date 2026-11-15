package domain

import (
	"path/filepath"

	domainErrors "github.com/ToggyO/cocker/internal/domain/errors"
)

// ContainerSpec describes the desired process, filesystem, and isolation settings.
type ContainerSpec struct {
	RootFS     string
	Command    []string
	Hostname   string
	WorkingDir string
	Env        EnvVars
	Resources  ResourceSpec
	Namespaces NamespaceSpec
	IO         IOSpec
}

// DefaultNamespaceSpec enables PID, mount, UTS, IPC, and network isolation.
// User namespaces remain disabled for the root-based MVP.
func DefaultNamespaceSpec() NamespaceSpec {
	return NamespaceSpec{
		PID:     true,
		Mount:   true,
		UTS:     true,
		IPC:     true,
		Network: true,
		User:    false,
	}
}

// Validate returns all detected domain rule violations, or nil if none are found.
// It does not check filesystem existence or create runtime resources.
func (s ContainerSpec) Validate() error {
	validationErrorsPtr := domainErrors.NewValidationErrors()

	if s.RootFS == "" {
		addError(validationErrorsPtr, domainErrors.ValidationError{
			Field:   "rootfs",
			Value:   s.RootFS,
			Message: domainErrors.ValidationRequired,
		})
	}

	if len(s.Command) == 0 {
		addError(validationErrorsPtr, domainErrors.ValidationError{
			Field:   "command",
			Message: domainErrors.ValidationNotEmpty})
	}

	if s.WorkingDir != "" && !filepath.IsAbs(s.WorkingDir) {
		addError(validationErrorsPtr, domainErrors.ValidationError{
			Field:   "workingDir",
			Value:   s.WorkingDir,
			Message: domainErrors.ValidationMustBeAbsolute,
		})
	}

	s.Env.Validate(validationErrorsPtr)
	s.Resources.Validate(validationErrorsPtr)
	s.Namespaces.Validate(validationErrorsPtr)

	if len(validationErrorsPtr.Errors) > 0 {
		return validationErrorsPtr
	}

	return nil
}

func addError(ve *domainErrors.ValidationErrors, e domainErrors.ValidationError) {
	ve.Errors = append(ve.Errors, e)
}
