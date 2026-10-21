package domain

import (
	"strings"
	"testing"

	"github.com/ToggyO/cocker/internal/domain/errors"
	"github.com/ToggyO/cocker/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const Val string = "val"

func TestContainerSpecValidate_Valid(t *testing.T) {
	containerSpec := createValidContainerSpec()
	assert.NoError(t, containerSpec.Validate(), "ContainerSpec.Validate() returned error. Expected no error")
}

func TestContainerSpecValidate_CollectsAllErrors(t *testing.T) {
	containerSpec := createInvalidContainerSpec()

	err := containerSpec.Validate()

	assert.Error(t, err)

	var validationErrors *domainErrors.ValidationErrors
	require.ErrorAs(t, err, &validationErrors) // TODO: сообщения

	assert.Len(t, validationErrors.Errors, 9)
	actualFieldNames := collectErrorFields(validationErrors)

	assert.Contains(t, actualFieldNames, "rootfs") // TODO: сообщения
	assert.Contains(t, actualFieldNames, "command")
	assert.Contains(t, actualFieldNames, "workingDir")
	assert.Contains(t, actualFieldNames, "namespaces.user")
	assert.Contains(t, actualFieldNames, "resources.memoryBytes")
	assert.Contains(t, actualFieldNames, "resources.pidsLimit")
	assert.Contains(t, actualFieldNames, "resources.cpuQuota.quota")
	assert.Contains(t, actualFieldNames, "resources.cpuQuota.period")
	assert.Contains(t, actualFieldNames, "namespaces.mount")
}

func TestContainerSpecValidate_WorkingDir(t *testing.T) {
	testCases := []struct {
		name       string
		workingDir string
		shouldFail bool
	}{
		{
			name:       "workingDir empty",
			workingDir: "",
			shouldFail: false,
		},
		{
			name:       "workingDir is absolute",
			workingDir: "/workingDir",
			shouldFail: false,
		},
		{
			name:       "workingDir is relative",
			workingDir: "workingDir",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			containerSpec := createContainerSpec(
				"/",
				[]string{"cmd"},
				"",
				tc.workingDir,
				nil,
				nil)

			err := containerSpec.Validate()

			if tc.shouldFail {
				assert.Error(t, err)
				var validationErrors *domainErrors.ValidationErrors
				require.ErrorAs(t, err, &validationErrors) // TODO: сообщения
				actualFieldNames := collectErrorFields(validationErrors)

				assert.Contains(t, actualFieldNames, "workingDir")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceSpecValidate(t *testing.T) {
	testCases := []struct {
		name         string
		spec         ResourceSpec
		targetFields map[string]struct{}
		shouldFail   bool
	}{
		{
			name:         "MemoryBytes and PidsLimit both nil is valid",
			spec:         createResourcesSpec(nil, nil, nil),
			targetFields: map[string]struct{}{"resources.memoryBytes": {}, "resources.pidsLimit": {}},
			shouldFail:   false,
		},
		{
			name:         "MemoryBytes and PidsLimit both 0 is invalid",
			spec:         createResourcesSpec(testutil.Ptr(uint64(0)), testutil.Ptr(uint64(0)), nil),
			targetFields: map[string]struct{}{"resources.memoryBytes": {}, "resources.pidsLimit": {}},
			shouldFail:   true,
		},
		{
			name:         "CPUQuota.Quota and CPUQuota.Period both 0 is invalid",
			spec:         createResourcesSpec(testutil.Ptr(uint64(1)), testutil.Ptr(uint64(1)), createCPUQuota(testutil.Ptr(uint64(0)), testutil.Ptr(uint64(0)))),
			targetFields: map[string]struct{}{"resources.cpuQuota.quota": {}, "resources.cpuQuota.period": {}},
			shouldFail:   true,
		},
		{
			name:         "All fields are valid",
			spec:         createResourcesSpec(testutil.Ptr(uint64(1)), testutil.Ptr(uint64(1)), createCPUQuota(testutil.Ptr(uint64(1)), testutil.Ptr(uint64(1)))),
			targetFields: map[string]struct{}{"resources.memoryBytes": {}, "resources.pidsLimit": {}, "resources.cpuQuota.quota": {}, "resources.cpuQuota.period": {}},
			shouldFail:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validationErrorsPtr := domainErrors.NewValidationErrors()
			tc.spec.Validate(validationErrorsPtr)

			actualFieldNames := collectErrorFields(validationErrorsPtr)
			if tc.shouldFail {
				assert.Equal(t, tc.targetFields, actualFieldNames)
			} else {
				assert.Empty(t, actualFieldNames)
			}
		})
	}
}

func TestDefaultNamespaceSpec(t *testing.T) {
	defaultNamespaceSpec := DefaultNamespaceSpec()
	assert.True(t, defaultNamespaceSpec.PID, "DefaultNamespaceSpec() return 'PID' as false. Expected true")
	assert.True(t, defaultNamespaceSpec.Mount, "DefaultNamespaceSpec() return 'Mount' as false. Expected true")
	assert.True(t, defaultNamespaceSpec.IPC, "DefaultNamespaceSpec() return 'IPC' as false. Expected true")
	assert.True(t, defaultNamespaceSpec.UTS, "DefaultNamespaceSpec() return 'UTS' as false. Expected true")
	assert.True(t, defaultNamespaceSpec.Network, "DefaultNamespaceSpec() return 'Network' as false. Expected true")
	assert.False(t, defaultNamespaceSpec.User, "DefaultNamespaceSpec() return 'User' as true. Expected false")
}

func TestNamespaceSpecValidate(t *testing.T) {
	spec := DefaultNamespaceSpec()
	spec.PID = false
	spec.Mount = false
	spec.User = true

	validationErrorsPtr := domainErrors.NewValidationErrors()
	spec.Validate(validationErrorsPtr)

	actualFieldNames := collectErrorFields(validationErrorsPtr)
	expectedFields := map[string]struct{}{
		"namespaces.pid":   {},
		"namespaces.mount": {},
		"namespaces.user":  {},
	}

	assert.Equal(t, expectedFields, actualFieldNames)
}

func TestEnvVarsValidate_ValidNames(t *testing.T) {
	envVars := EnvVars{
		{"PATH", Val},
		{"APP_ENV", Val},
		{"_LOG_LEVEL", Val},
		{"PORT2", Val},
		{"_2FA", Val},
	}

	validationErrorsPtr := domainErrors.NewValidationErrors()
	envVars.Validate(validationErrorsPtr)

	assert.Empty(t, validationErrorsPtr.Errors)
}

func TestEnvVarsValidate_InvalidNames(t *testing.T) {
	envVars := EnvVars{
		{"", Val},
		{"2APP", Val},
		{"APP-ENV", Val},
		{"APP.ENV", Val},
		{"APP ENV", Val},
		{"APP=ENV", Val},
		{"ПЕРЕМЕННАЯ", Val},
	}

	validationErrorsPtr := domainErrors.NewValidationErrors()
	envVars.Validate(validationErrorsPtr)

	actualValues := collectErrorValues(validationErrorsPtr)
	expectedValues := make(map[string]struct{}, len(envVars))
	for _, env := range envVars {
		expectedValues[env.Name] = struct{}{}
	}
	assert.Equal(t, expectedValues, actualValues)
}

func TestEnvVarsValidate_DuplicateNames(t *testing.T) {
	envVars := EnvVars{
		{"APP_ENV", Val},
		{"DAO_ENV", Val},
		{"APP_ENV", Val},
	}

	validationErrorsPtr := domainErrors.NewValidationErrors()
	envVars.Validate(validationErrorsPtr)

	assert.Equal(t, 1, len(validationErrorsPtr.Errors))
	assert.Equal(t, "env[2].name", validationErrorsPtr.Errors[0].Field)
	assert.Equal(t, "APP_ENV", validationErrorsPtr.Errors[0].Value)
}

func TestValidationErrorsError(t *testing.T) {
	validationErrors := domainErrors.NewValidationErrors()

	validationErrors.Errors = append(
		validationErrors.Errors,
		domainErrors.ValidationError{
			Field:   "rootfs",
			Message: domainErrors.ValidationRequired,
		},
		domainErrors.ValidationError{
			Field:   "env[0].name",
			Value:   "APP-ENV",
			Message: domainErrors.ValidationInvalidEnvName,
		},
	)

	message := validationErrors.Error()

	expectedParts := []string{
		"validation errors:",
		"rootfs: value is required",
		`env[0].name "APP-ENV": ` + domainErrors.ValidationInvalidEnvName,
	}

	for _, part := range expectedParts {
		if !strings.Contains(message, part) {
			t.Errorf(
				"error message does not contain %q: %q",
				part,
				message,
			)
		}
	}
}

func createValidContainerSpec() ContainerSpec {
	return createContainerSpec("/", []string{"echo", "hello"}, "host", "/opt/workdir", nil, nil)
}

func createInvalidContainerSpec() ContainerSpec {
	namespaces := DefaultNamespaceSpec()
	namespaces.Mount = false
	namespaces.User = true

	resources := createResourcesSpec(testutil.Ptr(uint64(0)), testutil.Ptr(uint64(0)), createCPUQuota(testutil.Ptr(uint64(0)), testutil.Ptr(uint64(0))))
	return createContainerSpec(
		"",
		[]string{},
		"",
		"./opt/workdir",
		&resources,
		&namespaces)
}

func createContainerSpec(
	rootFs string,
	command []string,
	hostname string,
	workingDir string,
	resources *ResourceSpec,
	namespaces *NamespaceSpec,
) ContainerSpec {
	spec := ContainerSpec{
		RootFS:     rootFs,
		Command:    command,
		Hostname:   hostname,
		WorkingDir: workingDir,
		Env: []EnvVar{
			{"APP_ENV", "production"},
		},
		IO: IOSpec{},
	}

	if resources != nil {
		spec.Resources = *resources
	} else {
		spec.Resources = createResourcesSpec(nil, nil, nil)
	}

	if namespaces != nil {
		spec.Namespaces = *namespaces
	} else {
		spec.Namespaces = DefaultNamespaceSpec()
	}

	return spec
}

func createResourcesSpec(memoryBytes *uint64, pidsLimit *uint64, cpuQuota *CPUQuota) ResourceSpec {
	spec := ResourceSpec{}

	if memoryBytes != nil {
		spec.MemoryBytes = memoryBytes
	} else {
		spec.MemoryBytes = testutil.Ptr(uint64(1000))
	}

	if pidsLimit != nil {
		spec.PidsLimit = pidsLimit
	} else {
		spec.PidsLimit = testutil.Ptr(uint64(1000))
	}

	if cpuQuota != nil {
		spec.CPUQuota = cpuQuota
	} else {
		spec.CPUQuota = createCPUQuota(nil, nil)
	}

	return spec
}

func createCPUQuota(quota *uint64, period *uint64) *CPUQuota {
	cpuQuota := CPUQuota{}

	if quota != nil {
		cpuQuota.Quota = *quota
	} else {
		cpuQuota.Quota = 50
	}

	if period != nil {
		cpuQuota.Period = *period
	} else {
		cpuQuota.Period = 10
	}

	return &cpuQuota
}

func collectErrorFields(validationErrors *domainErrors.ValidationErrors) map[string]struct{} {
	if validationErrors == nil {
		return map[string]struct{}{}
	}

	actualFieldNames := make(map[string]struct{}, len(validationErrors.Errors))
	for _, validationError := range validationErrors.Errors {
		actualFieldNames[validationError.Field] = struct{}{}
	}
	return actualFieldNames
}

func collectErrorValues(validationErrors *domainErrors.ValidationErrors) map[string]struct{} {
	if validationErrors == nil {
		return map[string]struct{}{}
	}

	actualValues := make(map[string]struct{}, len(validationErrors.Errors))
	for _, validationError := range validationErrors.Errors {
		actualValues[validationError.Value] = struct{}{}
	}
	return actualValues
}
