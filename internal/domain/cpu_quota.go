package domain

import (
	"fmt"

	"github.com/ToggyO/cocker/internal/domain/errors"
)

type CPUQuota struct {
	Quota  uint64
	Period uint64
}

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
