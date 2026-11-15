package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ToggyO/cocker/internal/config/errors"
)

// DecodeJSON reads exactly one JSON object and rejects unknown fields.
// It preserves optional values without normalization or domain validation.
func DecodeJSON(r io.Reader) (Input, error) {
	// A pointer distinguishes a JSON object from a top-level null.
	var input *Input

	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&input)
	if errors.Is(err, io.EOF) {
		return Input{}, fmt.Errorf("decode config: %w", configErrors.DecodingErrorObjectRequired)
	}

	if err != nil {
		return Input{}, fmt.Errorf("decode config: %w", err)
	}

	if input == nil {
		return Input{}, fmt.Errorf("decode config: %w", configErrors.DecodingErrorObjectRequired)
	}

	// Only whitespace may follow the first object.
	var dummy json.RawMessage
	err = decoder.Decode(&dummy)

	if errors.Is(err, io.EOF) {
		return *input, nil
	}

	if err == nil {
		return Input{}, fmt.Errorf("decode config: %w", configErrors.DecodingErrorMultipleObjectNotAllowed)
	}

	return Input{}, fmt.Errorf("decode config: %w", err)
}
