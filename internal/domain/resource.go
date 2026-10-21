package domain

import (
	"fmt"

	"github.com/ToggyO/cocker/internal/domain/errors"
)

type ResourceSpec struct {
	MemoryBytes *uint64
	CPUQuota    *CPUQuota
	PidsLimit   *uint64
}

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
