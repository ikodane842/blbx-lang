// Package diagnostic defines the shared BLBX tooling diagnostic format.
package diagnostic

import "fmt"

// Positions are one-based Unicode code-point positions; ends are exclusive.
type Diagnostic struct {
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
	return fmt.Sprintf("%s:%d:%d: %s %s: %s", d.File, d.Line, d.Column, d.Severity, d.Code, d.Message)
}
