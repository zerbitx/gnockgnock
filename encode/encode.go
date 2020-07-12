package encode

import (
	"encoding/json"
	"io"
)

// JSONIndented encodes a value into a writer with a single space indentation
func JSONIndented(v interface{}, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")

	return encoder.Encode(v)
}
