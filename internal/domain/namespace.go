package domain

import (
	"fmt"

	domainErrors "github.com/ToggyO/cocker/internal/domain/errors"
)

// NamespaceSpec identifies the Linux namespaces requested for a container.
type NamespaceSpec struct {
	PID     bool
	Mount   bool
	UTS     bool
	IPC     bool
	Network bool
	User    bool
}

// Validate appends violations for disabled PID or mount isolation or enabled user isolation.
func (ns NamespaceSpec) Validate(errors *domainErrors.ValidationErrors) {
	if !ns.PID {
		addError(errors, domainErrors.ValidationError{
			Field:   "namespaces.pid",
			Value:   fmt.Sprintf("%t", ns.PID),
			Message: domainErrors.ValidationMustBeTrue,
		})
	}

	if !ns.Mount {
		addError(errors, domainErrors.ValidationError{
			Field:   "namespaces.mount",
			Value:   fmt.Sprintf("%t", ns.Mount),
			Message: domainErrors.ValidationMustBeTrue,
		})
	}

	if ns.User {
		addError(errors, domainErrors.ValidationError{
			Field:   "namespaces.user",
			Value:   fmt.Sprintf("%t", ns.User),
			Message: domainErrors.ValidationMustBeFalse,
		})
	}
}
