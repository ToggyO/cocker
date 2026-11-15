package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ToggyO/cocker/internal/config/errors"
	"github.com/ToggyO/cocker/internal/domain"
	"github.com/ToggyO/cocker/internal/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize_DefaultValues(t *testing.T) {
	input := Input{}

	spec, err := Normalize(input)
	assert.NoError(t, err)

	expected := domain.ContainerSpec{
		RootFS:     "",
		Command:    nil,
		Hostname:   "",
		WorkingDir: slash,
		Env:        nil,
		Resources:  domain.ResourceSpec{},
		Namespaces: domain.DefaultNamespaceSpec(),
		IO:         domain.DefaultIOSpec(),
	}

	assert.Equal(t, expected, spec)
	assert.True(t, spec.IO.AttachStdin)
	assert.True(t, spec.IO.AttachStdout)
	assert.True(t, spec.IO.AttachStderr)
	require.Error(t, spec.Validate())
}

func TestNormalize_RootFS(t *testing.T) {
	workingDir := t.TempDir()
	t.Chdir(workingDir)

	testCases := []struct {
		name     string
		rootFS   string
		expected string
	}{
		{
			name:     "rootfs is empty",
			rootFS:   "",
			expected: "",
		},
		{
			name:     "absolute rootfs",
			rootFS:   filepath.Join(workingDir, "rootfs"),
			expected: filepath.Join(workingDir, "rootfs"),
		},
		{
			name:     "relative rootfs",
			rootFS:   "./rootfs",
			expected: filepath.Join(workingDir, "rootfs"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := Input{
				RootFS: ptr.To(tc.rootFS),
			}
			spec, err := Normalize(input)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, spec.RootFS)
		})
	}
}

func TestNormalize_WorkingDir(t *testing.T) {
	workingDir := t.TempDir()
	t.Chdir(workingDir)

	testCases := []struct {
		name       string
		workingDir string
		expected   string
	}{
		{name: "explicit empty working directory", workingDir: "", expected: "/"},
		{
			name:       "clean absolute working directory",
			workingDir: "/app/./cache/../work//",
			expected:   "/app/work",
		},
		{
			name:       "keep relative working directory relative",
			workingDir: "app/./cache/../work//",
			expected:   "app/work",
		},
		{
			name:       "preserve leading parent directory",
			workingDir: "../app/./work",
			expected:   "../app/work",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := Input{
				WorkingDir: ptr.To(tc.workingDir),
			}

			spec, err := Normalize(input)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, spec.WorkingDir)
		})
	}

}

func TestNormalize_ResourcesValidConfiguration(t *testing.T) {
	input := Input{
		Resources: &resourcesInput{
			Memory: ptr.To("256B"),
			CPUs:   ptr.To("500m"),
			Pids:   ptr.To(uint64(64)),
		},
	}

	spec, err := Normalize(input)
	assert.NoError(t, err)

	assert.Equal(t, ptr.To(uint64(256)), spec.Resources.MemoryBytes)
	require.NoError(t, err)
	require.NotNil(t, spec.Resources.CPUQuota)
	assert.Equal(t, uint64(50_000), spec.Resources.CPUQuota.Quota)
	assert.Equal(t, uint64(100_000), spec.Resources.CPUQuota.Period)
	assert.Equal(t, ptr.To(uint64(64)), spec.Resources.PidsLimit)
}

func TestNormalize_ResourcesInvalidConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		input         Input
		expectedError func(t *testing.T, err error)
	}{
		{
			name: "resources: invalid memory",
			input: Input{
				Resources: &resourcesInput{
					Memory: ptr.To("256UiB"),
				},
			},
			expectedError: func(t *testing.T, err error) {
				assertExpectedError(t, err, "resources.memory")
			},
		},
		{
			name: "resources: invalid cpus",
			input: Input{
				Resources: &resourcesInput{
					CPUs: ptr.To("500mic"),
				},
			},
			expectedError: func(t *testing.T, err error) {
				assertExpectedError(t, err, "resources.cpus")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Normalize(tc.input)
			assert.Error(t, err)
			tc.expectedError(t, err)
		})
	}
}

func assertExpectedError(t *testing.T, err error, fieldName string) {
	t.Helper()

	var fieldErr *configErrors.FieldError
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, fieldName, fieldErr.Field)

	var parseErr *configErrors.ParseError
	require.ErrorAs(t, fieldErr.Unwrap(), &parseErr)
}

func TestNormalize_ZeroPidsRequiresValidation(t *testing.T) {
	spec, err := Normalize(Input{
		RootFS:    ptr.To("/rootfs"),
		Command:   []string{"/bin/sh"},
		Resources: &resourcesInput{Pids: ptr.To(uint64(0))},
	})
	require.NoError(t, err)
	require.NotNil(t, spec.Resources.PidsLimit)
	assert.Zero(t, *spec.Resources.PidsLimit)
	require.Error(t, spec.Validate())
}

func TestNormalize_PreservesInputAndCopiesSlices(t *testing.T) {
	input, err := DecodeJSON(strings.NewReader(jsonConfig))
	require.NoError(t, err)
	before, err := DecodeJSON(strings.NewReader(jsonConfig))
	require.NoError(t, err)
	spec, err := Normalize(input)
	require.NoError(t, err)
	assert.Equal(t, before, input)
	assert.Equal(t, input.Command, spec.Command)
	assert.Equal(t, "cocker-dev", spec.Hostname)
	assert.Equal(t, domain.EnvVars{{Name: "APP_ENV", Value: "dev"}, {Name: "SOME_ENV", Value: "env val"}, {Name: "APP_ENV", Value: "prod"}}, spec.Env)
	require.ErrorContains(t, spec.Validate(), "env[2].name")
	spec.Command[0] = "changed"
	spec.Env[0].Value = "changed"
	*spec.Resources.PidsLimit = 99
	assert.Equal(t, before, input)
}

func TestNormalize_PartialResources(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input resourcesInput
		want  domain.ResourceSpec
	}{
		{"empty", resourcesInput{}, domain.ResourceSpec{}},
		{"memory", resourcesInput{Memory: ptr.To("1MiB")}, domain.ResourceSpec{MemoryBytes: ptr.To(uint64(1048576))}},
		{"cpu", resourcesInput{CPUs: ptr.To("500m")}, domain.ResourceSpec{CPUQuota: &domain.CPUQuota{Quota: 50000, Period: 100000}}},
		{"pids", resourcesInput{Pids: ptr.To(uint64(64))}, domain.ResourceSpec{PidsLimit: ptr.To(uint64(64))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := Normalize(Input{Resources: &tc.input})
			require.NoError(t, err)
			assert.Equal(t, tc.want, spec.Resources)
		})
	}
}

func TestNormalize_ResourceErrorCauses(t *testing.T) {
	for _, tc := range []struct {
		field string
		input resourcesInput
	}{
		{"resources.memory", resourcesInput{Memory: ptr.To("0")}},
		{"resources.cpus", resourcesInput{CPUs: ptr.To("0m")}},
	} {
		t.Run(tc.field, func(t *testing.T) {
			_, err := Normalize(Input{Resources: &tc.input})
			var target *configErrors.FieldError
			require.ErrorAs(t, err, &target)
			assert.Equal(t, tc.field, target.Field)
			assert.ErrorIs(t, err, configErrors.ParsingErrorZeroValue)
		})
	}
}

// Removing the current directory makes resolution of a relative rootfs fail.
func TestNormalize_RootFSError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.Remove(dir))
	_, err := Normalize(Input{RootFS: ptr.To("./rootfs")})
	var target *configErrors.FieldError
	require.ErrorAs(t, err, &target)
	assert.Equal(t, "rootfs", target.Field)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
