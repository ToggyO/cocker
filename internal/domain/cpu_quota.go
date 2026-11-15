package domain

import (
	"fmt"

	domainErrors "github.com/ToggyO/cocker/internal/domain/errors"
)

// CPUQuota specifies the allowed CPU time per scheduling period, in microseconds.
type CPUQuota struct {
	Quota  uint64
	Period uint64
}

// Validate appends violations for nonpositive quota or period values to errors.
func (q CPUQuota) Validate(errors *domainErrors.ValidationErrors) {
	if q.Quota <= 0 {
		addError(errors, domainErrors.ValidationError{
			Field:   "resources.cpuQuota.quota",
			Value:   fmt.Sprintf("%d", q.Quota),
			Message: domainErrors.ValidationMustBePositive,
		})
	}

	if q.Period <= 0 {
		addError(errors, domainErrors.ValidationError{
			Field:   "resources.cpuQuota.period",
			Value:   fmt.Sprintf("%d", q.Period),
			Message: domainErrors.ValidationMustBePositive,
		})
	}
}
