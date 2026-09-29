// Package config implements the content compiler support library (IMP-003):
// it parses the 24 launch catalogs in docs/07_content into a
// CandidateSnapshot, expands finite-derived tables deterministically, and
// emits compile diagnostics per docs/06_data/content_authoring_contract.md.
package config

import (
	"fmt"
	"sort"
)

// Diagnostic codes are canonical in content_authoring_contract.md §6.
const (
	CodeCatalogFileNotFound    = "CATALOG_FILE_NOT_FOUND"
	CodeTableSyntaxError       = "TABLE_SYNTAX_ERROR"
	CodeDuplicatePrimaryKey    = "DUPLICATE_PRIMARY_KEY"
	CodeUnresolvedReference    = "UNRESOLVED_REFERENCE"
	CodeValueOutOfBounds       = "VALUE_OUT_OF_BOUNDS"
	CodeBalanceGuardrailFailed = "BALANCE_GUARDRAIL_FAILED"
	CodeIntegrationCheckFailed = "INTEGRATION_CHECK_FAILED"
)

// Diagnostic is one compile-time problem, located at file:line:col.
type Diagnostic struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Col    int    `json:"col"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("[%s:%d:%d] [%s] %s", d.File, d.Line, d.Col, d.Code, d.Detail)
}

// Diagnostics is an ordered list of compile errors.
type Diagnostics []Diagnostic

// Add appends a diagnostic.
func (ds *Diagnostics) Add(file string, line, col int, code, detail string) {
	*ds = append(*ds, Diagnostic{File: file, Line: line, Col: col, Code: code, Detail: detail})
}

// HasErrors reports whether the candidate must be rejected: any diagnostic at
// all rejects the whole candidate revision (contract invariant).
func (ds Diagnostics) HasErrors() bool { return len(ds) > 0 }

// Sort orders diagnostics deterministically by (file, line, col, code,
// detail) so identical compiles emit byte-identical diagnostic output.
func (ds *Diagnostics) Sort() {
	sort.SliceStable(*ds, func(i, j int) bool {
		a, b := (*ds)[i], (*ds)[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Col != b.Col {
			return a.Col < b.Col
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Detail < b.Detail
	})
}
