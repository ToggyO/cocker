package config

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	configErrors "github.com/ToggyO/cocker/internal/config/errors"
	"github.com/ToggyO/cocker/internal/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var jsonConfig = `{
  "rootfs": "./rootfs",
  "command": ["/bin/echo", "hello world", "$HOME"],
  "hostname": "cocker-dev",
  "workingDir": "/app",
  "env": [
    {"name": "APP_ENV", "value": "dev"},
    {"name": "SOME_ENV", "value": "env val"},
    {"name": "APP_ENV", "value": "prod"}
  ],
  "resources": {
    "memory": "256MiB",
    "cpus": "500m",
    "pids": 64
  }
}

`

func TestDecodeJSON_PreservesOptionalValues(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		want      Input
	}{
		{"null fields", `{"rootfs":null,"command":null,"hostname":null,"workingDir":null,"env":null,"resources":null}`, Input{}},
		{"null limits", `{"resources":{"memory":null,"cpus":null,"pids":null}}`, Input{Resources: &resourcesInput{}}},
		{"explicit empty values", `{"rootfs":"","hostname":"","workingDir":"","command":[],"env":[]}`, Input{RootFS: ptr.To(""), Hostname: ptr.To(""), WorkingDir: ptr.To(""), Command: []string{}, Env: []envVar{}}},
		{"maximum pids", `{"resources":{"pids":18446744073709551615}}`, Input{Resources: &resourcesInput{Pids: ptr.To(^uint64(0))}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeJSON(strings.NewReader(tc.raw))
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDecodeJSON_RejectsAdditionalInvalidInput(t *testing.T) {
	for _, raw := range []string{`[]`, `123`, `"text"`, `{"command":"echo"}`, `{"resources":{"memory":256}}`} {
		t.Run(raw, func(t *testing.T) {
			_, err := DecodeJSON(strings.NewReader(raw))
			var target *json.UnmarshalTypeError
			require.ErrorAs(t, err, &target)
		})
	}
	for _, tc := range []struct{ raw, field string }{
		{`{"env":[{"name":"A","unexpected":true}]}`, "unexpected"},
		{`{"io":{}}`, "io"}, {`{"namespaces":{}}`, "namespaces"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			_, err := DecodeJSON(strings.NewReader(tc.raw))
			require.Error(t, err)
			assert.Contains(t, err.Error(), `unknown field "`+tc.field+`"`)
		})
	}
	for _, raw := range []string{`{"rootfs":`, `{} {`} {
		t.Run(raw, func(t *testing.T) {
			_, err := DecodeJSON(strings.NewReader(raw))
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		})
	}
	_, err := DecodeJSON(strings.NewReader(`{} garbage`))
	var syntaxErr *json.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	_, err = DecodeJSON(strings.NewReader(`{} null`))
	assert.ErrorIs(t, err, configErrors.DecodingErrorMultipleObjectNotAllowed)
}

var expectedValidInput = Input{
	RootFS:     ptr.To("./rootfs"),
	Command:    []string{"/bin/echo", "hello world", "$HOME"},
	Hostname:   ptr.To("cocker-dev"),
	WorkingDir: ptr.To("/app"),
	Env: []envVar{
		{Name: "APP_ENV", Value: "dev"},
		{Name: "SOME_ENV", Value: "env val"},
		{Name: "APP_ENV", Value: "prod"},
	},
	Resources: ptr.To(resourcesInput{
		Memory: ptr.To("256MiB"),
		CPUs:   ptr.To("500m"),
		Pids:   ptr.To(uint64(64)),
	}),
}

func TestDecodeJSON_ValidConfiguration(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected Input
	}{
		{
			name:     "valid single input object",
			input:    jsonConfig,
			expected: expectedValidInput,
		},
		{
			name: "single object with trailing whitespace",
			input: jsonConfig + `


`,
			expected: expectedValidInput,
		},
		{
			name:     "empty object",
			input:    "{}",
			expected: Input{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reader := strings.NewReader(testCase.input)

			input, err := DecodeJSON(reader)
			assert.NoError(t, err)

			assert.Equal(t, testCase.expected, input)
		})
	}
}

func TestDecodeJSON_InvalidConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedError func(t *testing.T, err error)
	}{
		{
			name:  "empty",
			input: "",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, configErrors.DecodingErrorObjectRequired)
			},
		},
		{
			name:  "spaces",
			input: "   ",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, configErrors.DecodingErrorObjectRequired)
			},
		},
		{
			name:  "null",
			input: "null",
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, configErrors.DecodingErrorObjectRequired)
			},
		},
		{
			name: "invalid field types",
			input: `{
  "rootfs": 1
}`,
			expectedError: func(t *testing.T, err error) {
				var typeErr *json.UnmarshalTypeError
				require.ErrorAs(t, err, &typeErr)
				assert.Equal(t, "rootfs", typeErr.Field)
			},
		},
		{
			name:  "unexpected top-level field",
			input: `{"unexpected": 1}`,
			expectedError: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), `unknown field "unexpected"`)
			},
		},
		{
			name:  "unexpected resource field",
			input: `{"resources": {"unexpected": true}}`,
			expectedError: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), `unknown field "unexpected"`)
			},
		},
		{
			name:  "syntax error",
			input: `{"rootfs": }`,
			expectedError: func(t *testing.T, err error) {
				var syntaxErr *json.SyntaxError
				require.ErrorAs(t, err, &syntaxErr)
			},
		},
		{
			name: "multiple input objects",
			input: jsonConfig + `
{
	"rootfs": "./rootfs"
}`,
			expectedError: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, configErrors.DecodingErrorMultipleObjectNotAllowed)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.input)
			_, err := DecodeJSON(reader)
			assert.Error(t, err)
			tc.expectedError(t, err)
		})
	}
}

func TestDecodeJSON_InvalidPidsTypes(t *testing.T) {
	for _, value := range []string{`"64"`, `-1`, `1.5`, `18446744073709551616`} {
		t.Run(value, func(t *testing.T) {
			_, err := DecodeJSON(strings.NewReader(`{"resources":{"pids":` + value + `}}`))
			var typeErr *json.UnmarshalTypeError
			require.ErrorAs(t, err, &typeErr)
		})
	}
}
