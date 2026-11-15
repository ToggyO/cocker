package configErrors

import "errors"

var (
	DecodingErrorMultipleObjectNotAllowed = errors.New("input must contain only one JSON object")
	DecodingErrorObjectRequired           = errors.New("input must be a JSON object")
)
