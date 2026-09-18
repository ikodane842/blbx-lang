// Package diagnostic defines the shared BLBX tooling diagnostic format.
package diagnostic

import (
	"errors"
	"fmt"
	"strings"
)

// Positions are one-based Unicode code-point positions; ends are exclusive.
type Diagnostic struct {
	Phase     string `json:"phase,omitempty"`
	Cause     error  `json:"-"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	Severity  string `json:"severity"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

func (d Diagnostic) Error() string {
	phase := d.Phase
	if phase == "" {
		phase = "syntax"
	}
	return fmt.Sprintf("%s %s: %s", phase, d.Code, d.Message)
}

func (d Diagnostic) Unwrap() error { return d.Cause }

// Runtime normalizes native and wrapped errors without duplicating prefixes.
// Syntax diagnostics originating in imported files retain their phase.
func Runtime(err error) error {
	return RuntimeCode(NativeFailure, err)
}

// RuntimeCode assigns a category to an uncategorized error, retaining any
// diagnostic already attached by the operation that actually failed.
func RuntimeCode(code string, err error) error {
	if err == nil {
		return nil
	}
	var existing Diagnostic
	if errors.As(err, &existing) {
		message := strings.ReplaceAll(err.Error(), existing.Error(), existing.Message)
		existing.Message = message
		existing.Cause = err
		return existing
	}
	return Diagnostic{Phase: "runtime", Severity: "error", Code: code, Message: err.Error(), Cause: err}
}
