package configErrors

// FieldError associates a configuration field with its underlying error.
type FieldError struct {
	Field string
	Err   error
}

// NewFieldError wraps a non-nil error with the configuration field name.
func NewFieldError(field string, err error) *FieldError {
	return &FieldError{
		Field: field,
		Err:   err,
	}
}

func (e *FieldError) Error() string {
	return e.Field + ": " + e.Err.Error()
}

func (e *FieldError) Unwrap() error {
	return e.Err
}
