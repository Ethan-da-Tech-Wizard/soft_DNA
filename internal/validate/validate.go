// Package validate parses and sanitizes raw JSON output from the LLM runner
// before it reaches the report writer. It enforces three constraints:
//
//  1. Schema conformance — JSON must unmarshal into the expected struct.
//  2. Symbol allowlist  — every symbol explanation must reference a name
//     that was verified by static analysis; hallucinated names are discarded.
//  3. Text hygiene      — file paths, URLs, and backtick-quoted code are
//     stripped from prose fields, then the field is length-capped.
//
// This package contains no LLM invocation logic; it is a pure transform.
package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// FileExplanation is the parsed and validated output of a file-explain LLM call.
// Its shape must match the JSON schema enforced by internal/llm/grammars/file_explain.gbnf.
type FileExplanation struct {
	FileSummary        string              `json:"file_summary"`
	SymbolExplanations []SymbolExplanation `json:"symbol_explanations"`
}

// SymbolExplanation pairs a verified symbol name with the LLM-generated prose.
type SymbolExplanation struct {
	Name        string `json:"name"`
	Explanation string `json:"explanation"`
}

// ProjectExplanation is the parsed and validated output of a project-overview LLM call.
// Its shape must match internal/llm/grammars/project_overview.gbnf.
type ProjectExplanation struct {
	ProjectDescription string `json:"project_description"`
}

// ValidationError records a single constraint violation discovered during validation.
// Callers receive a slice so multiple problems can be reported in one pass.
type ValidationError struct {
	Field  string // JSON key path of the offending value
	Reason string // human-readable description of the violation
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validate: %s: %s", e.Field, e.Reason)
}

const (
	maxFileSummaryLen        = 500
	maxProjectDescriptionLen = 1000
	ellipsis                 = "…" // U+2026, appended when text is truncated
)

var (
	// pathRe matches Unix/Windows absolute path fragments such as /usr/bin/python
	// or \Windows\System32. Relative paths (internal/pkg/...) are not matched
	// because they originate from verified SYSTEM FACTS and are not hallucinations.
	pathRe = regexp.MustCompile(`[/\\][A-Za-z0-9_./\\]+`)

	// urlRe matches http/https URLs that should never appear in prose summaries.
	urlRe = regexp.MustCompile(`https?://[^\s]+`)

	// backtickRe matches backtick-delimited inline code fragments (ASCII 0x60).
	// These are stripped because they may contain raw source snippets that
	// were never part of the verified SYSTEM FACTS grounding context.
	backtickRe = regexp.MustCompile("\x60[^\x60]*\x60")

	// spaceRe collapses consecutive horizontal whitespace left behind after
	// path/URL/backtick removal. Newlines are not collapsed (TrimSpace handles ends).
	spaceRe = regexp.MustCompile(`[ \t]{2,}`)
)

// sanitizeText removes file paths, URLs, and backtick-quoted code from s,
// collapses any resulting consecutive spaces, then trims surrounding whitespace.
// It is applied to every prose field before length checking.
func sanitizeText(s string) string {
	s = backtickRe.ReplaceAllString(s, "")
	s = urlRe.ReplaceAllString(s, "")
	s = pathRe.ReplaceAllString(s, "")
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// truncateRunes truncates s to at most max Unicode code points, appending
// ellipsis if truncation occurred. Returns the (possibly truncated) string
// and a boolean indicating whether truncation happened.
func truncateRunes(s string, max int) (string, bool) {
	r := []rune(s)
	if len(r) <= max {
		return s, false
	}
	return string(r[:max]) + ellipsis, true
}

// ValidateFileExplanation parses raw LLM JSON output and applies all three
// validation constraints. It never returns a hard error; instead, every
// constraint violation is recorded in the returned []ValidationError so the
// caller can log them and still use whatever valid content remains.
func ValidateFileExplanation(raw []byte, allowedSymbols []string) (FileExplanation, []ValidationError) {
	var fe FileExplanation
	var errs []ValidationError

	lastCurly := bytes.LastIndexByte(raw, '}')
	if lastCurly != -1 {
		raw = raw[:lastCurly+1]
	}

	if err := json.Unmarshal(raw, &fe); err != nil {
		return FileExplanation{}, []ValidationError{{
			Field:  "json",
			Reason: "failed to parse JSON: " + err.Error(),
		}}
	}

	// Build O(1) lookup set from the verified symbol list.
	allowed := make(map[string]struct{}, len(allowedSymbols))
	for _, name := range allowedSymbols {
		allowed[name] = struct{}{}
	}

	// Filter symbol explanations: keep only names that exist in SYSTEM FACTS.
	var kept []SymbolExplanation
	for _, sym := range fe.SymbolExplanations {
		if _, ok := allowed[sym.Name]; !ok {
			errs = append(errs, ValidationError{
				Field:  "symbol_explanations." + sym.Name,
				Reason: "symbol not in allowed list — discarded",
			})
			continue
		}
		sym.Explanation = sanitizeText(sym.Explanation)
		kept = append(kept, sym)
	}
	fe.SymbolExplanations = kept

	// Sanitize and length-cap the file summary.
	fe.FileSummary = sanitizeText(fe.FileSummary)
	if truncated, wasTruncated := truncateRunes(fe.FileSummary, maxFileSummaryLen); wasTruncated {
		errs = append(errs, ValidationError{
			Field:  "file_summary",
			Reason: fmt.Sprintf("truncated to %d characters", maxFileSummaryLen),
		})
		fe.FileSummary = truncated
	}

	return fe, errs
}

// ValidateProjectExplanation parses raw LLM JSON output for a project overview
// and sanitizes the prose description. Truncation is silent (no ValidationError)
// because a long description is always recoverable by reading less of it.
// A hard error is returned only when the JSON is structurally invalid.
func ValidateProjectExplanation(raw []byte) (ProjectExplanation, error) {
	var pe ProjectExplanation
	lastCurly := bytes.LastIndexByte(raw, '}')
	if lastCurly != -1 {
		raw = raw[:lastCurly+1]
	}

	if err := json.Unmarshal(raw, &pe); err != nil {
		return ProjectExplanation{}, fmt.Errorf("validate: parse project explanation: %w", err)
	}
	pe.ProjectDescription = sanitizeText(pe.ProjectDescription)
	truncated, _ := truncateRunes(pe.ProjectDescription, maxProjectDescriptionLen)
	pe.ProjectDescription = truncated
	return pe, nil
}
