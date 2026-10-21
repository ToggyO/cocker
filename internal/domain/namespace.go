package domain

import (
	"fmt"

	"github.com/ToggyO/cocker/internal/domain/errors"
)

type NamespaceSpec struct {
	PID     bool
	Mount   bool
	UTS     bool
	IPC     bool
	Network bool
	User    bool
}

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
