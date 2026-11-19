// ==============================================================================
// Program: symbols.go
// Programmer: Ethan-da-Tech-Wizard
// Last Modified: November 18, 2026
// Purpose: Extracts declarations, functions, methods, classes, and exported interfaces across supported source languages using regex and pattern analysis without requiring full compilation toolchains.
// ==============================================================================

// Package symbols extracts simple source symbols without executing target code.
package symbols

// File role: turns source files into a flat list of `Symbol`s (functions, methods, classes, types, imports,
// SQL tables) that feed three consumers: the SQLite index (internal/index), the Markdown report
// (internal/cli) and the grounding facts given to the local LLM (internal/grounding).
//
// Approach: Go files are parsed with the real go/parser; every other language is handled with line-by-line
// regular expressions. That is deliberately approximate (no compiler needed, no code executed, works on a
// half-written file) at the cost of false positives and misses, for example nested or macro-generated
// definitions are not recognised and a symbol's EndLine is usually its start line.
//
// Concurrency: ExtractAll / ExtractAllIncremental run one worker per CPU; results are sorted afterwards, so
// output order never depends on goroutine scheduling.
//
// Tested in symbols_test.go (`go test ./internal/symbols`).
import (
	"bufio"
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"codelens-local/internal/scanner"
)

// Symbol describes a function, class/type, method, or import.
// Kind values used across the program: "function", "method", "class", "type", "import", "table", "view", "procedure".
// Line numbers are 1-based. Signature is the declaration text where the language gives one (for Go it is rebuilt from the AST and lists parameter names only).
type Symbol struct {
	Name      string
	Kind      string
	Signature string
	Doc       string
	File      string
	StartLine int
	EndLine   int
}

var (
	// One regular expression per language construct, grouped by language prefix: py (Python), js (JavaScript/TypeScript),
	// cpp/c, rs (Rust), rb (Ruby), r, sw (Swift), sql. Each is applied to a whitespace-trimmed single line, and
	// capture group 1 (or the first non-empty group) is the symbol name. Patterns are compiled once at program start.
	// The C/C++ function patterns are the loosest; callers reject lines ending in ';' and control keywords to cut false positives.
	pyImportRe   = regexp.MustCompile(`^(?:from\s+([A-Za-z0-9_./]+)\s+import\s+(.+)|import\s+(.+))`)
	pyDefRe      = regexp.MustCompile(`^(class|def)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(\(.*)?`)
	jsImportRe   = regexp.MustCompile(`^import\s+(.+)`)
	jsFuncRe     = regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(([^)]*)\)`)
	jsClassRe    = regexp.MustCompile(`^(?:export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsArrowRe    = regexp.MustCompile(`^(?:export\s+)?const\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][A-Za-z0-9_$]*)\s*=>`)
	cppIncludeRe = regexp.MustCompile(`^\s*#include\s+["<]([^">]+)[">]`)
	cppClassRe   = regexp.MustCompile(`^\s*(?:class|struct)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	cppFuncRe    = regexp.MustCompile(`^\s*(?:[A-Za-z0-9_::<>&*~]+\s+)?([A-Za-z_~][A-Za-z0-9_:]*)\s*\(([^)]*)\)\s*(?:const)?\s*(?::\s*[^{;]+)?\s*(?:{|;)?`)
	cTypeRe      = regexp.MustCompile(`^\s*(?:typedef\s+)?(?:struct|union|enum)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	cFuncRe      = regexp.MustCompile(`^\s*(?:[A-Za-z0-9_*\s]+)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*(?:{|;)?`)
	rsImportRe   = regexp.MustCompile(`^\s*use\s+([A-Za-z0-9_::*{}]+);`)
	rsFuncRe     = regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rsTypeRe     = regexp.MustCompile(`^\s*(?:pub\s+)?(?:struct|enum|union|trait)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rbImportRe   = regexp.MustCompile(`^\s*(?:require|require_relative|load)\s+["']([^"']+)["']`)
	rbMethodRe   = regexp.MustCompile(`^\s*def\s+([A-Za-z0-9_!?.]+)`)
	rbClassRe    = regexp.MustCompile(`^\s*(?:class|module)\s+([A-Za-z0-9_:]+)`)
	rImportRe    = regexp.MustCompile(`^\s*(?:library|require)\s*\(([^)]+)\)`)
	rFuncRe      = regexp.MustCompile(`^\s*([A-Za-z0-9_.]+)\s*(?:<-|=)\s*function\s*\(([^)]*)\)`)
	swImportRe   = regexp.MustCompile(`^\s*import\s+([A-Za-z0-9_]+)`)
	swFuncRe     = regexp.MustCompile(`^\s*(?:public|private|internal|fileprivate)?\s*(?:static|class)?\s*func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`)
	swTypeRe     = regexp.MustCompile(`^\s*(?:public|private|internal|fileprivate)?\s*(?:class|struct|enum|protocol)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	sqlTableRe   = regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z0-9_.]+)`)
	sqlViewRe    = regexp.MustCompile(`(?i)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?VIEW\s+([A-Za-z0-9_.]+)`)
	sqlProcRe    = regexp.MustCompile(`(?i)^\s*CREATE\s+PROCEDURE\s+([A-Za-z0-9_.]+)`)

	// Openers mark lines that may begin a multi-line function signature and
	// are safe to join with continuation lines (see joinUnclosed).
	pyOpenerRe = regexp.MustCompile(`^(?:class|def)\s`)
	jsOpenerRe = regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s`)
	rsOpenerRe = regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s`)
)

// maxJoinLines bounds how many continuation lines a multi-line signature may
// pull in; anything longer stays truncated rather than risking a runaway join.
// Three continuation lines cover typical wrapped parameter lists without letting an unbalanced "(" in a comment or string swallow the rest of the file.
const maxJoinLines = 3

// unbalanced reports whether s opens more parentheses than it closes.
// A cheap textual test, not a parser: parentheses inside string literals are counted too, which is acceptable because the result only decides whether to join more lines.
func unbalanced(s string) bool {
	return strings.Count(s, "(") > strings.Count(s, ")")
}

// joinUnclosed joins up to maxJoinLines following lines onto lines[i] while
// the parameter list remains unclosed, collapsing runs of whitespace.
// It returns the joined logical line and the index of the last line consumed.
// Used by the Python, JavaScript and Rust extractors for signatures that span several lines. The returned index
// lets the caller skip the lines that were consumed, so they are not scanned a second time as separate symbols.
func joinUnclosed(lines []string, i int) (string, int) {
	joined := strings.TrimSpace(lines[i])
	j := i
	for unbalanced(joined) && j+1 < len(lines) && j-i < maxJoinLines {
		j++
		joined += " " + strings.TrimSpace(lines[j])
	}
	return strings.Join(strings.Fields(joined), " "), j
}

// ExtractAll extracts symbols for supported files in parallel using a worker pool.
// Worker pool: one goroutine per CPU reads tasks from a channel and sends symbol slices to a results channel.
// Both channels are buffered to len(files), so neither the producer loop nor the workers can block each other.
// Complexity is O(total bytes); wall time scales with CPU count. Never returns nil for non-empty input unless nothing matched.
func ExtractAll(root string, files []scanner.FileInfo) []Symbol {
	// At least one worker, in case NumCPU reports 0 in an unusual environment.
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	type task struct {
		file  scanner.FileInfo
		index int
	}

	type result struct {
		symbols []Symbol
		index   int
	}

	// All tasks are queued up front (buffered channel) and the channel is then closed, so workers exit when the queue drains.
	tasks := make(chan task, len(files))
	results := make(chan result, len(files))

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				syms := Extract(root, t.file)
				results <- result{symbols: syms, index: t.index}
			}
		}()
	}

	for i, f := range files {
		tasks <- task{file: f, index: i}
	}
	close(tasks)

	wg.Wait()
	// Safe only after wg.Wait(): every worker has finished sending, so ranging over results below terminates.
	close(results)

	var out []Symbol
	for res := range results {
		out = append(out, res.symbols...)
	}

	// Total order (file, start line, kind, name) so the result is identical on every run regardless of which worker finished first.
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].StartLine != out[j].StartLine {
			return out[i].StartLine < out[j].StartLine
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ExtractAllIncremental extracts symbols in parallel, reusing cachedSymbols if ContentHash matches cachedHashes.
// Same as ExtractAll, but a file whose ContentHash equals the cached hash reuses its cached symbols instead of
// being re-read and re-parsed. A cache hit requires both entries to exist; any miss falls back to Extract.
func ExtractAllIncremental(root string, files []scanner.FileInfo, cachedHashes map[string]string, cachedSymbols map[string][]Symbol) []Symbol {
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	type task struct {
		file  scanner.FileInfo
		index int
	}

	type result struct {
		symbols []Symbol
		index   int
	}

	tasks := make(chan task, len(files))
	results := make(chan result, len(files))

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				// Cache hit only when the content hash is unchanged; the path alone is not enough because the file may have been edited.
				if cachedHash, ok := cachedHashes[t.file.Path]; ok && cachedHash == t.file.ContentHash {
					if syms, ok := cachedSymbols[t.file.Path]; ok {
						results <- result{symbols: syms, index: t.index}
						continue
					}
				}
				syms := Extract(root, t.file)
				results <- result{symbols: syms, index: t.index}
			}
		}()
	}

	for i, f := range files {
		tasks <- task{file: f, index: i}
	}
	close(tasks)

	wg.Wait()
	close(results)

	var out []Symbol
	for res := range results {
		out = append(out, res.symbols...)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].StartLine != out[j].StartLine {
			return out[i].StartLine < out[j].StartLine
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Extract extracts symbols from one file. Unsupported or unreadable files return no symbols.
// Dispatches on the language name set by the scanner. Skipped and binary files (and unknown languages) return nil, never an error:
// a file that cannot be analysed simply contributes no symbols.
func Extract(root string, file scanner.FileInfo) []Symbol {
	if file.Skipped || file.IsBinary {
		return nil
	}

	path := filepath.Join(root, filepath.FromSlash(file.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Language strings must match the scanner's names exactly (for example "C++", "JavaScript"). TypeScript reuses the JavaScript extractor.
	switch file.Language {
	case "Go":
		return extractGo(file.Path, data)
	case "Python":
		return extractPython(file.Path, data)
	case "JavaScript", "TypeScript":
		return extractJS(file.Path, data)
	case "Ruby":
		return extractRuby(file.Path, data)
	case "R":
		return extractR(file.Path, data)
	case "Swift":
		return extractSwift(file.Path, data)
	case "SQL":
		return extractSQL(file.Path, data)
	case "Rust":
		return extractRust(file.Path, data)
	case "C":
		return extractC(file.Path, data)
	case "C++":
		return extractCPP(file.Path, data)
	default:
		return nil
	}
}

// Uses the standard library parser, so Go symbols are accurate (real line ranges, methods vs functions).
// A file with syntax errors yields nil rather than partial results.
func extractGo(path string, data []byte) []Symbol {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, parser.ParseComments)
	if err != nil {
		return nil
	}

	var out []Symbol
	for _, decl := range file.Decls {
		switch item := decl.(type) {
		case *ast.GenDecl:
			// Only type declarations and imports are reported from GenDecl; constants and variables are not symbols here.
			if item.Tok != token.TYPE && item.Tok != token.IMPORT {
				continue
			}
			for _, spec := range item.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					out = append(out, Symbol{
						Name:      spec.Name.Name,
						Kind:      "type",
						File:      path,
						StartLine: fset.Position(spec.Pos()).Line,
						EndLine:   fset.Position(spec.End()).Line,
					})
				case *ast.ImportSpec:
					name := strings.Trim(spec.Path.Value, `"`)
					out = append(out, Symbol{
						Name:      name,
						Kind:      "import",
						File:      path,
						StartLine: fset.Position(spec.Pos()).Line,
						EndLine:   fset.Position(spec.End()).Line,
					})
				}
			}
		case *ast.FuncDecl:
			kind := "function"
			name := item.Name.Name
			// A receiver makes it a method; plain functions have none.
			if item.Recv != nil {
				kind = "method"
			}
			out = append(out, Symbol{
				Name:      name,
				Kind:      kind,
				Signature: goSignature(name, item),
				File:      path,
				StartLine: fset.Position(item.Pos()).Line,
				EndLine:   fset.Position(item.End()).Line,
			})
		}
	}
	return out
}

// Builds "func Name(a, b)" from parameter names only (no types, no result list), which keeps table cells short.
func goSignature(name string, fn *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")
	b.WriteString(name)
	if fn.Type.Params != nil {
		b.WriteString("(")
		var parts []string
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				parts = append(parts, name.Name)
			}
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString(")")
	}
	return b.String()
}

// Line-based: top-level and nested `def`/`class` are both found (indentation is stripped), so methods appear as functions.
func extractPython(path string, data []byte) []Symbol {
	var out []Symbol
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		startLine := i + 1
		trimmed := strings.TrimSpace(lines[i])
		// Decorators never block the def/class on the following line.
		// Blank lines, comments and decorators are skipped; a decorator never prevents the following def from matching because that line is examined on its own.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "@") {
			continue
		}
		if match := pyImportRe.FindStringSubmatch(trimmed); match != nil {
			name := match[1]
			if name == "" {
				name = match[3]
			}
			out = append(out, Symbol{Name: name, Kind: "import", File: path, StartLine: startLine, EndLine: startLine})
			continue
		}

		// Only a def/class line with an unclosed "(" is joined with its continuation lines, so ordinary statements are never merged.
		if pyOpenerRe.MatchString(trimmed) && unbalanced(trimmed) {
			trimmed, i = joinUnclosed(lines, i)
		}

		// EndLine is the last joined line (`i + 1`), not the end of the function body: bodies are not tracked.
		if match := pyDefRe.FindStringSubmatch(trimmed); match != nil {
			kind := "function"
			if match[1] == "class" {
				kind = "class"
			}
			out = append(out, Symbol{
				Name:      match[2],
				Kind:      kind,
				Signature: strings.TrimSuffix(trimmed, ":"),
				File:      path,
				StartLine: startLine,
				EndLine:   i + 1,
			})
		}
	}
	return out
}

// Recognises imports, `function` declarations, classes and arrow functions bound with `const`. Used for both JavaScript and TypeScript;
// TypeScript-only syntax (interfaces, type aliases, generics before the paren) is not recognised.
func extractJS(path string, data []byte) []Symbol {
	var out []Symbol
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		startLine := i + 1
		trimmed := strings.TrimSpace(lines[i])
		// Decorators never block the declaration on the following line.
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "@") {
			continue
		}
		if match := jsImportRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: strings.TrimSpace(match[1]), Kind: "import", File: path, StartLine: startLine, EndLine: startLine})
			continue
		}
		if jsOpenerRe.MatchString(trimmed) && unbalanced(trimmed) {
			trimmed, i = joinUnclosed(lines, i)
		}
		if match := jsFuncRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "function", Signature: trimmed, File: path, StartLine: startLine, EndLine: i + 1})
			continue
		}
		if match := jsClassRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "class", Signature: trimmed, File: path, StartLine: startLine, EndLine: startLine})
			continue
		}
		if match := jsArrowRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "function", Signature: trimmed, File: path, StartLine: startLine, EndLine: startLine})
		}
	}
	return out
}

// Uses bufio.Scanner (not Split) because these formats need no look-ahead. Note bufio.Scanner has a 64 KiB line limit; longer lines stop the scan early.
func extractRuby(path string, data []byte) []Symbol {
	var out []Symbol
	sc := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for sc.Scan() {
		line++
		trimmed := strings.TrimSpace(sc.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if match := rbImportRe.FindStringSubmatch(trimmed); match != nil {
			// Imports are one-line symbols, so start and end line are the same.
			out = append(out, Symbol{Name: match[1], Kind: "import", File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := rbMethodRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "method", Signature: trimmed, File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := rbClassRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "class", Signature: trimmed, File: path, StartLine: line, EndLine: line})
		}
	}
	return out
}

// Matches CREATE TABLE / VIEW / PROCEDURE (case-insensitive, optional IF NOT EXISTS / OR REPLACE) one line at a time; `--` comment lines are skipped.
func extractSQL(path string, data []byte) []Symbol {
	var out []Symbol
	sc := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for sc.Scan() {
		line++
		trimmed := strings.TrimSpace(sc.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		if match := sqlTableRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "table", Signature: trimmed, File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := sqlViewRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "view", Signature: trimmed, File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := sqlProcRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "procedure", Signature: trimmed, File: path, StartLine: line, EndLine: line})
		}
	}
	return out
}

// Types (class, struct, enum, protocol) are checked before functions so a type line is never misread as a function.
func extractSwift(path string, data []byte) []Symbol {
	var out []Symbol
	sc := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for sc.Scan() {
		line++
		trimmed := strings.TrimSpace(sc.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if match := swImportRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "import", File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := swTypeRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "type", Signature: trimmed, File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := swFuncRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "function", Signature: trimmed, File: path, StartLine: line, EndLine: line})
		}
	}
	return out
}

// R has no declaration keywords for functions: a name assigned a `function(...)` value is treated as a function definition.
func extractR(path string, data []byte) []Symbol {
	var out []Symbol
	sc := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for sc.Scan() {
		line++
		trimmed := strings.TrimSpace(sc.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if match := rImportRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: strings.TrimSpace(match[1]), Kind: "import", File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := rFuncRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "function", Signature: trimmed, File: path, StartLine: line, EndLine: line})
		}
	}
	return out
}

// Attributes such as #[derive(...)] are skipped so the item on the next line is still seen; imports are `use` statements ending in ';'.
func extractRust(path string, data []byte) []Symbol {
	var out []Symbol
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		startLine := i + 1
		trimmed := strings.TrimSpace(lines[i])
		// Attributes (#[derive(...)] etc.) never block the item below them.
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#[") {
			continue
		}
		if match := rsImportRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "import", File: path, StartLine: startLine, EndLine: startLine})
			continue
		}
		if rsOpenerRe.MatchString(trimmed) && unbalanced(trimmed) {
			trimmed, i = joinUnclosed(lines, i)
		}
		if match := rsFuncRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "function", Signature: trimmed, File: path, StartLine: startLine, EndLine: i + 1})
			continue
		}
		if match := rsTypeRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "type", Signature: trimmed, File: path, StartLine: startLine, EndLine: startLine})
		}
	}
	return out
}

// Handles #include lines, struct/union/enum types and function definitions. The function regex is loose, so lines ending in ';' (prototypes, calls) and control keywords are rejected.
func extractC(path string, data []byte) []Symbol {
	var out []Symbol
	sc := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for sc.Scan() {
		line++
		trimmed := strings.TrimSpace(sc.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if match := cppIncludeRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "import", File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := cTypeRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "type", Signature: trimmed, File: path, StartLine: line, EndLine: line})
			continue
		}
		// Lines ending in ';' are prototypes or call statements, not
		// definition bodies — rejecting them kills most false positives.
		if match := cFuncRe.FindStringSubmatch(trimmed); match != nil && !strings.HasSuffix(trimmed, ";") {
			name := match[1]
			// `if (x)`, `while (x)` and similar match the function regex, so they are filtered by name.
			if name == "if" || name == "while" || name == "for" || name == "switch" || name == "return" {
				continue
			}
			out = append(out, Symbol{Name: name, Kind: "function", Signature: trimmed, File: path, StartLine: line, EndLine: line})
		}
	}
	return out
}

// Like C, plus classes/structs. A qualified name such as `Foo::bar` becomes method "bar", which is how out-of-class method definitions are recognised.
func extractCPP(path string, data []byte) []Symbol {
	var out []Symbol
	sc := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for sc.Scan() {
		line++
		trimmed := strings.TrimSpace(sc.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if match := cppIncludeRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "import", File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := cppClassRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "class", Signature: trimmed, File: path, StartLine: line, EndLine: line})
			continue
		}
		// Lines ending in ';' are prototypes or call statements, not
		// definition bodies — rejecting them kills most false positives.
		if match := cppFuncRe.FindStringSubmatch(trimmed); match != nil && !strings.HasSuffix(trimmed, ";") {
			name := match[1]
			if name == "if" || name == "while" || name == "switch" || name == "for" || name == "catch" || name == "return" {
				continue
			}

			// Strip the class qualifier so the symbol name is the bare method name; the full qualified text stays in Signature.
			cleanName := name
			if idx := strings.LastIndex(name, "::"); idx != -1 {
				cleanName = name[idx+2:]
			}
			kind := "function"
			if strings.Contains(name, "::") {
				kind = "method"
			}
			out = append(out, Symbol{Name: cleanName, Kind: kind, Signature: trimmed, File: path, StartLine: line, EndLine: line})
		}
	}
	return out
}
