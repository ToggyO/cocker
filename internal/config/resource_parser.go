package config

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/ToggyO/cocker/internal/config/errors"
	"github.com/ToggyO/cocker/internal/domain"
)

// ParseMemoryBytes parses a positive integer memory limit into bytes.
// Supported suffixes are B, KiB, MiB, and GiB; an omitted suffix denotes bytes.
func ParseMemoryBytes(value string) (uint64, error) {
	if value == "" {
		return 0, configErrors.ParsingErrorEmpty
	}

	if !isDigit(value[0]) {
		return 0, configErrors.ParsingErrorInvalidNumber
	}

	length := len(value)

	suffixStart := length
	for i := 0; i < len(value); i++ {
		val := value[i]
		if isDigit(val) {
			if suffixStart < len(value) {
				// Reject digits after the suffix begins, for example "12Ki3B".
				return 0, configErrors.ParsingErrorInvalidNumber
			}
			continue
		}

		// Check value is valid number and non-floating point number
		if isFloatingPoint(val) {
			return 0, configErrors.ParsingErrorInvalidNumber
		}

		if suffixStart == len(value) {
			suffixStart = i
		}
	}

	numericPart := value[:suffixStart]
	suffix := value[suffixStart:]

	number, err := strconv.ParseUint(numericPart, 10, 64)
	if err != nil {
		return 0, configErrors.NewParseError("invalid memory amount", err)
	}

	if number == 0 {
		return 0, configErrors.ParsingErrorZeroValue
	}

	var multiplier uint64

	switch suffix {
	case "", unitsBytes:
		multiplier = 1
	case unitsKilobytes:
		multiplier = 1024
	case unitsMegabytes:
		multiplier = 1024 * 1024
	case unitsGigabytes:
		multiplier = 1024 * 1024 * 1024
	default:
		return 0, configErrors.NewParseError(fmt.Sprintf(configErrors.UnitsErrorMessageTemplate, unitsArr), nil)
	}

	//check for uint64 overflow of result value
	if number > math.MaxUint64/multiplier {
		return 0, configErrors.ParsingErrorOverflow
	}

	return number * multiplier, nil
}

// ParseCPUQuota parses a positive integer CPU limit with the m suffix (millicores).
// The returned quota and fixed period of 100000 are expressed in microseconds.
func ParseCPUQuota(value string) (domain.CPUQuota, error) {
	cpuQuota := domain.CPUQuota{}
	if value == "" {
		return cpuQuota, configErrors.ParsingErrorEmpty
	}

	if !isDigit(value[0]) {
		return cpuQuota, configErrors.ParsingErrorInvalidNumber
	}

	length := len(value)

	if length == 1 {
		return cpuQuota, configErrors.NewParseError(fmt.Sprintf("invalid millicores value '%s'", value), nil)
	}

	if value[length-1] != unitsMillicores {
		return cpuQuota, configErrors.NewParseError(fmt.Sprintf(configErrors.UnitsErrorMessageTemplate, []rune{unitsMillicores}), nil)
	}

	rawValue := value[:length-1]

	// Check value is valid number and non-floating point number
	if !isValidValue(rawValue, isDigit) {
		return cpuQuota, configErrors.ParsingErrorInvalidNumber
	}

	millicores, err := strconv.ParseUint(rawValue, 10, 64)
	if err != nil {
		return cpuQuota, configErrors.NewParseError("invalid quota amount", err)
	}

	if millicores < 1 {
		return cpuQuota, configErrors.ParsingErrorZeroValue
	}

	const quotaMultiplier = cpuPeriod / 1000 // 100

	if millicores > math.MaxUint64/quotaMultiplier {
		return cpuQuota, configErrors.ParsingErrorOverflow
	}

	cpuQuota.Quota = millicores * quotaMultiplier
	cpuQuota.Period = cpuPeriod
	return cpuQuota, nil
}

// ParsePidsLimit parses a positive decimal integer process limit into a uint64.
// Empty, signed, fractional, zero, and out-of-range values are rejected.
func ParsePidsLimit(value string) (uint64, error) {
	if value == "" {
		return 0, configErrors.ParsingErrorEmpty
	}

	if !isDigit(value[0]) {
		return 0, configErrors.ParsingErrorInvalidNumber
	}

	// Check value is valid number and non-floating point number
	if !isValidValue(value, isDigit) {
		return 0, configErrors.ParsingErrorInvalidNumber
	}

	number, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return 0, configErrors.ParsingErrorOverflow
		}
		return 0, configErrors.NewParseError("invalid PIDs amount", err)
	}

	if number == 0 {
		return 0, configErrors.ParsingErrorZeroValue
	}

	return number, nil
}

func isValidValue(value string, predicates ...func(ch uint8) bool) bool {
	for i := 0; i < len(value); i++ {
		for _, predicate := range predicates {
			if result := predicate(value[i]); !result {
				return false
			}
		}
	}
	return true
}

func isDigit(char uint8) bool {
	if char < '0' || char > '9' {
		return false
	}
	return true
}

func isFloatingPoint(char uint8) bool {
	if char == '.' || char == ',' {
		return true
	}
	return false
}
