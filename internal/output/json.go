// Package output renders an ebs.Result to a writer in either JSON (default,
// machine-friendly) or a human-readable table.
package output

import (
	"encoding/json"
	"io"

	"github.com/jun-uen0/aws-orphan-finder/internal/ebs"
)

// WriteJSON emits r as indented JSON. HTML escaping is disabled so that
// characters like &, <, > survive in tag values.
func WriteJSON(w io.Writer, r ebs.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}
