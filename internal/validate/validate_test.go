package validate

import (
	"strings"
	"testing"
)

// ---- sanitizeText unit tests ----

func TestSanitizeTextStripsAbsolutePaths(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"calls /usr/bin/python to run", "calls to run"},
		{"reads /home/user/project/main.go directly", "reads directly"},
		{"on Windows \\Users\\admin\\file.txt is used", "on Windows is used"},
		{"no path here", "no path here"},
	}
	for _, tc := range cases {
		got := sanitizeText(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeText(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitizeTextStripsURLs(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"see https://example.com/docs for more", "see for more"},
		{"fetches data from http://api.example.com/v1/data", "fetches data from"},
		{"no URL here", "no URL here"},
	}
	for _, tc := range cases {
		got := sanitizeText(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeText(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitizeTextStripsBacktickedCode(t *testing.T) {
	// Backtick (ASCII 0x60) written as \x60 so the literal is preserved
	// regardless of how this file is processed by tooling.
	bt := "\x60"
	cases := []struct {
		input string
		want  string
	}{
		{"calls the " + bt + "Scan" + bt + " function to scan", "calls the function to scan"},
		{"uses " + bt + "foo" + bt + " and " + bt + "bar" + bt + " internally", "uses and internally"},
		{"no backticks here", "no backticks here"},
	}
	for _, tc := range cases {
		got := sanitizeText(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeText(%q) = %q, want %q", tc.input, got, tc.want)
		}
		// Backtick characters must be absent from the output.
		if strings.Contains(got, bt) {
			t.Errorf("backtick survived sanitization: %q", got)
		}
	}
}

func TestSanitizeTextStripsMultiplePatterns(t *testing.T) {
	// A string containing all three injection types at once.
	// Backtick written as \x60 to survive tooling that may strip literal backticks.
	bt := "\x60"
	input := "loads " + bt + "cfg.Load" + bt + " from /etc/config via https://cdn.example.com/cfg"
	got := sanitizeText(input)
	if strings.Contains(got, bt) {
		t.Errorf("backtick content not stripped: %q", got)
	}
	if strings.Contains(got, "/etc/config") {
		t.Errorf("path not stripped: %q", got)
	}
	if strings.Contains(got, "https://") {
		t.Errorf("URL not stripped: %q", got)
	}
	// After stripping all three, only prose words should remain (no double spaces).
	if strings.Contains(got, "  ") {
		t.Errorf("double space left after sanitization: %q", got)
	}
}

func TestSanitizeTextTrimsWhitespace(t *testing.T) {
	got := sanitizeText("  hello world  ")
	if got != "hello world" {
		t.Errorf("sanitizeText = %q, want %q", got, "hello world")
	}
}

func TestSanitizeTextEmptyString(t *testing.T) {
	if got := sanitizeText(""); got != "" {
		t.Errorf("sanitizeText(\"\") = %q, want empty", got)
	}
}

// ---- truncateRunes unit tests ----

func TestTruncateRunesBelowLimit(t *testing.T) {
	s, truncated := truncateRunes("hello", 10)
	if truncated {
		t.Error("truncated = true, want false for string below limit")
	}
	if s != "hello" {
		t.Errorf("s = %q, want %q", s, "hello")
	}
}

func TestTruncateRunesAtExactLimit(t *testing.T) {
	s, truncated := truncateRunes("hello", 5)
	if truncated {
		t.Error("truncated = true, want false for string at exact limit")
	}
	if s != "hello" {
		t.Errorf("s = %q, want %q", s, "hello")
	}
}

func TestTruncateRunesExceedsLimit(t *testing.T) {
	s, truncated := truncateRunes("hello world", 5)
	if !truncated {
		t.Error("truncated = false, want true for string above limit")
	}
	if s != "hello"+ellipsis {
		t.Errorf("s = %q, want %q", s, "hello"+ellipsis)
	}
}

func TestTruncateRunesHandlesMultibyteRunes(t *testing.T) {
	// "日本語" is 3 runes; truncating to 2 should yield "日本…" not corrupt bytes.
	s, truncated := truncateRunes("日本語テスト", 2)
	if !truncated {
		t.Error("truncated = false, want true")
	}
	if s != "日本"+ellipsis {
		t.Errorf("s = %q, want %q", s, "日本"+ellipsis)
	}
}

// ---- ValidateFileExplanation tests ----

func TestValidateFileExplanationValidJSON(t *testing.T) {
	raw := []byte(`{
		"file_summary": "Scans the file system for source files.",
		"symbol_explanations": [
			{"name": "Scan", "explanation": "Walks the directory tree."},
			{"name": "FileInfo", "explanation": "Holds metadata for one file."}
		]
	}`)
	allowed := []string{"Scan", "FileInfo"}
	fe, errs := ValidateFileExplanation(raw, allowed)
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if fe.FileSummary != "Scans the file system for source files." {
		t.Errorf("FileSummary = %q", fe.FileSummary)
	}
	if len(fe.SymbolExplanations) != 2 {
		t.Fatalf("len(SymbolExplanations) = %d, want 2", len(fe.SymbolExplanations))
	}
	if fe.SymbolExplanations[0].Name != "Scan" {
		t.Errorf("SymbolExplanations[0].Name = %q", fe.SymbolExplanations[0].Name)
	}
}

func TestValidateFileExplanationInvalidJSON(t *testing.T) {
	fe, errs := ValidateFileExplanation([]byte(`not json`), nil)
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	if errs[0].Field != "json" {
		t.Errorf("errs[0].Field = %q, want 'json'", errs[0].Field)
	}
	if !strings.Contains(errs[0].Reason, "failed to parse JSON") {
		t.Errorf("errs[0].Reason = %q", errs[0].Reason)
	}
	// FileExplanation must be empty on parse failure.
	if fe.FileSummary != "" || len(fe.SymbolExplanations) != 0 {
		t.Errorf("expected empty FileExplanation on parse failure, got %+v", fe)
	}
}

func TestValidateFileExplanationEmptyJSON(t *testing.T) {
	_, errs := ValidateFileExplanation([]byte{}, nil)
	if len(errs) != 1 || errs[0].Field != "json" {
		t.Errorf("empty bytes: expected json parse error, got %v", errs)
	}
}

func TestValidateFileExplanationAllowlistFiltersUnknownSymbols(t *testing.T) {
	raw := []byte(`{
		"file_summary": "A scanner.",
		"symbol_explanations": [
			{"name": "Scan",        "explanation": "Walks tree."},
			{"name": "HallucinatedFunc", "explanation": "Does something."},
			{"name": "FileInfo",    "explanation": "Holds metadata."}
		]
	}`)
	allowed := []string{"Scan", "FileInfo"}
	fe, errs := ValidateFileExplanation(raw, allowed)

	// Exactly one error for the unknown symbol.
	unknownErrs := filterField(errs, "symbol_explanations.HallucinatedFunc")
	if len(unknownErrs) != 1 {
		t.Errorf("expected 1 allowlist error, got %v", errs)
	}
	if !strings.Contains(unknownErrs[0].Reason, "not in allowed list") {
		t.Errorf("reason = %q", unknownErrs[0].Reason)
	}

	// Only the two allowed symbols must survive.
	if len(fe.SymbolExplanations) != 2 {
		t.Fatalf("len(SymbolExplanations) = %d, want 2", len(fe.SymbolExplanations))
	}
	for _, sym := range fe.SymbolExplanations {
		if sym.Name == "HallucinatedFunc" {
			t.Errorf("hallucinated symbol was not discarded")
		}
	}
}

func TestValidateFileExplanationAllowlistEmptyList(t *testing.T) {
	// No allowed symbols → every symbol explanation is discarded.
	raw := []byte(`{
		"file_summary": "Something.",
		"symbol_explanations": [
			{"name": "Foo", "explanation": "Does foo."},
			{"name": "Bar", "explanation": "Does bar."}
		]
	}`)
	fe, errs := ValidateFileExplanation(raw, nil)
	if len(fe.SymbolExplanations) != 0 {
		t.Errorf("expected all symbols discarded, got %v", fe.SymbolExplanations)
	}
	if len(errs) != 2 {
		t.Errorf("expected 2 allowlist errors, got %v", errs)
	}
}

func TestValidateFileExplanationSanitizesSymbolExplanations(t *testing.T) {
	raw := []byte(`{
		"file_summary": "A scanner.",
		"symbol_explanations": [
			{"name": "Scan", "explanation": "Loads config from /etc/scanner.conf via https://cdn.example.com/conf."}
		]
	}`)
	fe, errs := ValidateFileExplanation(raw, []string{"Scan"})
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	exp := fe.SymbolExplanations[0].Explanation
	if strings.Contains(exp, "/etc/") {
		t.Errorf("path not stripped from explanation: %q", exp)
	}
	if strings.Contains(exp, "https://") {
		t.Errorf("URL not stripped from explanation: %q", exp)
	}
}

func TestValidateFileExplanationSanitizesFileSummary(t *testing.T) {
	// ` is the JSON-safe encoding of backtick (U+0060).
	raw := []byte("{\"file_summary\": \"Reads /var/log/app.log and calls `parse` to process.\",\"symbol_explanations\":[]}")
	fe, _ := ValidateFileExplanation(raw, nil)
	if strings.Contains(fe.FileSummary, "/var/log/app.log") {
		t.Errorf("path not stripped from FileSummary: %q", fe.FileSummary)
	}
	if strings.Contains(fe.FileSummary, "\x60") {
		t.Errorf("backtick not stripped from FileSummary: %q", fe.FileSummary)
	}
}

func TestValidateFileExplanationTruncatesFileSummary(t *testing.T) {
	// Build a summary that exceeds 500 runes.
	long := strings.Repeat("a", 600)
	raw := []byte(`{"file_summary":"` + long + `","symbol_explanations":[]}`)
	fe, errs := ValidateFileExplanation(raw, nil)

	truncErrs := filterField(errs, "file_summary")
	if len(truncErrs) != 1 {
		t.Errorf("expected 1 truncation error, got %v", errs)
	}
	if !strings.Contains(truncErrs[0].Reason, "truncated") {
		t.Errorf("reason = %q", truncErrs[0].Reason)
	}
	runes := []rune(fe.FileSummary)
	// 500 content runes + 1 ellipsis rune = 501 total.
	if len(runes) != maxFileSummaryLen+1 {
		t.Errorf("len(runes) = %d, want %d", len(runes), maxFileSummaryLen+1)
	}
	if !strings.HasSuffix(fe.FileSummary, ellipsis) {
		t.Errorf("FileSummary does not end with ellipsis: %q…", fe.FileSummary[:20])
	}
}

func TestValidateFileExplanationNoTruncationNeeded(t *testing.T) {
	// Exactly 500 runes — must not trigger truncation.
	exactly500 := strings.Repeat("b", 500)
	raw := []byte(`{"file_summary":"` + exactly500 + `","symbol_explanations":[]}`)
	fe, errs := ValidateFileExplanation(raw, nil)
	if len(filterField(errs, "file_summary")) != 0 {
		t.Errorf("unexpected truncation error for 500-rune summary")
	}
	if fe.FileSummary != exactly500 {
		t.Errorf("FileSummary was modified unexpectedly")
	}
}

// ---- ValidateProjectExplanation tests ----

func TestValidateProjectExplanationValidJSON(t *testing.T) {
	raw := []byte(`{"project_description": "This project scans Go source code."}`)
	pe, err := ValidateProjectExplanation(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pe.ProjectDescription != "This project scans Go source code." {
		t.Errorf("ProjectDescription = %q", pe.ProjectDescription)
	}
}

func TestValidateProjectExplanationInvalidJSON(t *testing.T) {
	_, err := ValidateProjectExplanation([]byte(`{bad json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "validate: parse project explanation") {
		t.Errorf("err = %q", err.Error())
	}
}

func TestValidateProjectExplanationSanitizesDescription(t *testing.T) {
	raw := []byte(`{"project_description": "Fetches config from https://cdn.example.com and reads /etc/app.conf using ."}`)
	pe, err := ValidateProjectExplanation(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(pe.ProjectDescription, "https://") {
		t.Errorf("URL not stripped: %q", pe.ProjectDescription)
	}
	if strings.Contains(pe.ProjectDescription, "/etc/") {
		t.Errorf("path not stripped: %q", pe.ProjectDescription)
	}
	if strings.Contains(pe.ProjectDescription, "`") {
		t.Errorf("backtick not stripped: %q", pe.ProjectDescription)
	}
}

func TestValidateProjectExplanationTruncatesSilently(t *testing.T) {
	// 1100-rune description must be silently truncated; no error returned.
	long := strings.Repeat("x", 1100)
	raw := []byte(`{"project_description":"` + long + `"}`)
	pe, err := ValidateProjectExplanation(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	runes := []rune(pe.ProjectDescription)
	// 1000 content runes + 1 ellipsis rune = 1001 total.
	if len(runes) != maxProjectDescriptionLen+1 {
		t.Errorf("len(runes) = %d, want %d", len(runes), maxProjectDescriptionLen+1)
	}
	if !strings.HasSuffix(pe.ProjectDescription, ellipsis) {
		t.Errorf("description does not end with ellipsis")
	}
}

func TestValidateProjectExplanationNoTruncationAt1000(t *testing.T) {
	exactly1000 := strings.Repeat("y", 1000)
	raw := []byte(`{"project_description":"` + exactly1000 + `"}`)
	pe, err := ValidateProjectExplanation(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pe.ProjectDescription != exactly1000 {
		t.Errorf("description was modified unexpectedly (len=%d)", len([]rune(pe.ProjectDescription)))
	}
}

func TestValidateProjectExplanationEmptyDescription(t *testing.T) {
	raw := []byte(`{"project_description": ""}`)
	pe, err := ValidateProjectExplanation(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pe.ProjectDescription != "" {
		t.Errorf("ProjectDescription = %q, want empty", pe.ProjectDescription)
	}
}

// ---- ValidationError.Error() test ----

func TestValidationErrorString(t *testing.T) {
	e := ValidationError{Field: "file_summary", Reason: "too long"}
	got := e.Error()
	if got != "validate: file_summary: too long" {
		t.Errorf("Error() = %q", got)
	}
}

// ---- helpers ----

// filterField returns the subset of errs whose Field matches target.
func filterField(errs []ValidationError, target string) []ValidationError {
	var out []ValidationError
	for _, e := range errs {
		if e.Field == target {
			out = append(out, e)
		}
	}
	return out
}

func TestValidateAskAnswer(t *testing.T) {
	raw := []byte(`{"answer": "The CLI starts in the Run function. See https://example.com for docs."}trailing noise`)
	aa, err := ValidateAskAnswer(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(aa.Answer, "The CLI starts in the Run function.") {
		t.Fatalf("answer lost prose: %q", aa.Answer)
	}
	if strings.Contains(aa.Answer, "https://") {
		t.Fatalf("URL survived sanitization: %q", aa.Answer)
	}
}

func TestValidateAskAnswerRejectsInvalidJSON(t *testing.T) {
	if _, err := ValidateAskAnswer([]byte("not json")); err == nil {
		t.Fatal("expected parse error for invalid JSON")
	}
}

func TestValidateAskAnswerTruncatesLongAnswers(t *testing.T) {
	long := strings.Repeat("a", 2000)
	aa, err := ValidateAskAnswer([]byte(`{"answer": "` + long + `"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len([]rune(aa.Answer)) > 1501 {
		t.Fatalf("answer not truncated: %d runes", len([]rune(aa.Answer)))
	}
}
