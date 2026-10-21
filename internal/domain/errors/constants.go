package domainErrors

const (
	ValidationRequired       = "value is required"
	ValidationNotEmpty       = "value must not be empty"
	ValidationMustBePositive = "value must be positive"
	ValidationMustBeAbsolute = "value must be an absolute path"
	ValidationMustBeTrue     = "value must be true"
	ValidationMustBeFalse    = "value must be false"
	ValidationInvalidEnvName = "environment variable name must start with a letter or underscore and contain only letters, digits, and underscores"
	ValidationNotUnique      = "value must be unique"
)
