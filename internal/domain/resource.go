package domain

import (
	"fmt"

	domainErrors "github.com/ToggyO/cocker/internal/domain/errors"
)

// ResourceSpec holds optional resource limits; nil fields mean no explicit limit.
type ResourceSpec struct {
	MemoryBytes *uint64
	CPUQuota    *CPUQuota
	PidsLimit   *uint64
}

// Validate appends violations for nonpositive resource limits to errors.
func (r ResourceSpec) Validate(errors *domainErrors.ValidationErrors) {
	if r.MemoryBytes != nil && *r.MemoryBytes <= 0 {
		addError(errors, domainErrors.ValidationError{
			Field:   "resources.memoryBytes",
			Value:   fmt.Sprintf("%d", *r.MemoryBytes),
			Message: domainErrors.ValidationMustBePositive,
		})
	}

	if r.PidsLimit != nil && *r.PidsLimit <= 0 {
		addError(errors, domainErrors.ValidationError{
			Field:   "resources.pidsLimit",
			Value:   fmt.Sprintf("%d", *r.PidsLimit),
			Message: domainErrors.ValidationMustBePositive,
		})
	}

	if r.CPUQuota != nil {
		r.CPUQuota.Validate(errors)
	}
}
