package codegraph

import "encoding/json"

// JSON serializes the Result as indented JSON.
func (r *Result) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
