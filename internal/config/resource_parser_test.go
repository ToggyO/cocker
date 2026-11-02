package config

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const expectedUnitsErrorMessagePrefix = "units must be in"

func TestMemoryBytesParser_ValidValues(t *testing.T) {
	testCases := []positiveTestCase{
		{
			name:           "no suffix",
			rawValue:       "1",
			expectedNumber: 1,
		},
		{
			name:           "no suffix for more complex value",
			rawValue:       "1048576",
			expectedNumber: 1048576,
		},
		{
			name:           "maximum bytes without suffix",
			rawValue:       "18446744073709551615",
			expectedNumber: math.MaxUint64,
		},
		{
			name:           "suffix 'B'",
			rawValue:       "512B",
			expectedNumber: 512,
		},
		{
			name:           "suffix 'KiB'",
			rawValue:       "64KiB",
			expectedNumber: 64 * 1024,
		},
		{
			name:           "suffix 'MiB'",
			rawValue:       "256MiB",
			expectedNumber: 256 * 1024 * 1024,
		},
		{
			name:           "suffix 'GiB'",
			rawValue:       "1GiB",
			expectedNumber: 1 * 1024 * 1024 * 1024,
		},
		{
			name:           "maximum safe GiB value",
			rawValue:       "17179869183GiB",
			expectedNumber: math.MaxUint64 - (math.MaxUint64 % (1024 * 1024 * 1024)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			num, err := ParseMemoryBytes(tc.rawValue)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedNumber, num)
		})
	}
}

func TestMemoryBytesParser_NotValidValues(t *testing.T) {
	testCases := []negativeTestCase{
		{
			name:     "empty",
			rawValue: "",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorEmpty)
			},
		},
		{
			name:     "bytes exceed uint64 range",
			rawValue: "18446744073709551616",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, strconv.ErrRange)
			},
		},
		{
			name:     "zero value",
			rawValue: "0B",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorZeroValue)
			},
		},
		{
			name:     "negative value",
			rawValue: "-1",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "value overflow",
			rawValue: "17179869184GiB",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorOverflow)
			},
		},
		{
			name:     "value with sign character",
			rawValue: "+1",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "value with invalid units",
			rawValue: "1UiB",
			expectedError: func(t *testing.T, err error) {
				var parseErr *ParseError
				require.ErrorAs(t, err, &parseErr)
				assert.True(t, strings.HasPrefix(parseErr.message, expectedUnitsErrorMessagePrefix))
			},
		},
		{
			name:     "invalid number",
			rawValue: "12u56B",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "unexpected space",
			rawValue: "1 11B",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "floating point value",
			rawValue: "1.1B",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMemoryBytes(tc.rawValue)
			assert.Error(t, err)
			tc.expectedError(t, err)
		})
	}
}

func TestCPUQuotaParser_ValidValues(t *testing.T) {
	testCases := []positiveTestCase{
		{
			name:           "1m",
			rawValue:       "1m",
			expectedNumber: 100,
		},
		{
			name:           "500m",
			rawValue:       "500m",
			expectedNumber: 50_000,
		},
		{
			name:           "1000m",
			rawValue:       "1000m",
			expectedNumber: 100_000,
		},
		{
			name:           "2000m",
			rawValue:       "2000m",
			expectedNumber: 200_000,
		},
		{
			name:           "184467440737096m",
			rawValue:       "184467440737096m",
			expectedNumber: 18446744073709600,
		},
		{
			name:           "maximum safe millicores",
			rawValue:       "184467440737095516m",
			expectedNumber: 18446744073709551600,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cpuQuota, err := ParseCPUQuota(tc.rawValue)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedNumber, cpuQuota.Quota)
			assert.Equal(t, uint64(100_000), cpuQuota.Period)
		})
	}

}

func TestCPUQuotaParser_NotValidValues(t *testing.T) {
	testCases := []negativeTestCase{
		{
			name:     "empty",
			rawValue: "",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorEmpty)
			},
		},
		{
			name:     "without units",
			rawValue: "10",
			expectedError: func(t *testing.T, err error) {
				var parseErr *ParseError
				require.ErrorAs(t, err, &parseErr)
				assert.True(t, strings.HasPrefix(parseErr.message, expectedUnitsErrorMessagePrefix))
			},
		},
		{
			name:     "zero number",
			rawValue: "0m",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorZeroValue)
			},
		},
		{
			name:     "negative value",
			rawValue: "-1m",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "value with sign character",
			rawValue: "+1m",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "single digit",
			rawValue: "1",
			expectedError: func(t *testing.T, err error) {
				var parseErr *ParseError
				require.ErrorAs(t, err, &parseErr)
				assert.True(t, strings.HasPrefix(parseErr.message, "invalid millicores value"))
			},
		},
		{
			name:     "floating point value",
			rawValue: "1.1m",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "unexpected space",
			rawValue: "1 11m",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "value overflow",
			rawValue: "184467440737095517m",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorOverflow)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCPUQuota(tc.rawValue)
			assert.Error(t, err)
			tc.expectedError(t, err)
		})
	}
}

func TestPidsLimitParser_ValidValues(t *testing.T) {
	testCases := []positiveTestCase{
		{
			name:           "1",
			rawValue:       "1",
			expectedNumber: 1,
		},
		{
			name:           "64",
			rawValue:       "64",
			expectedNumber: 64,
		},
		{
			name:           "1000",
			rawValue:       "1000",
			expectedNumber: 1000,
		},
		{
			name:           "maximum process limit",
			rawValue:       "18446744073709551615",
			expectedNumber: math.MaxUint64,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pidsLimit, err := ParsePidsLimit(tc.rawValue)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedNumber, pidsLimit)
		})
	}
}

func TestPidsLimitParser_NotValidValues(t *testing.T) {
	tesCases := []negativeTestCase{
		{
			name:     "empty",
			rawValue: "",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorEmpty)
			},
		},
		{
			name:     "zero value",
			rawValue: "0",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorZeroValue)
			},
		},
		{
			name:     "negative value",
			rawValue: "-1",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "floating point value",
			rawValue: "1.1",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "unexpected space",
			rawValue: "1 11",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorInvalidNumber)
			},
		},
		{
			name:     "value overflow",
			rawValue: "18446744073709551616",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ParsingErrorOverflow)
			},
		},
	}

	for _, tc := range tesCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePidsLimit(tc.rawValue)
			assert.Error(t, err)
			tc.expectedError(t, err)
		})
	}
}

type positiveTestCase struct {
	name           string
	rawValue       string
	expectedNumber uint64
}

type negativeTestCase struct {
	name          string
	rawValue      string
	expectedError func(t *testing.T, err error)
}
