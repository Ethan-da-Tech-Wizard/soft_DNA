// Program: soft_DNA - symbols.go
// Programmer: Ethan-da-Tech-Wizard (written with Claude, an AI assistant)
// Last Modified: 2026-11-18
// Purpose: Extracts functions, methods, classes/types and imports from source files in ten languages (Go via its parser, the others via line-based patterns), in parallel and with optional reuse of cached results.

// Package symbols extracts simple source symbols without executing target code.
package symbols

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
	// Line patterns for each supported language, grouped by language prefix (py, js, cpp, c, rs, rb, r, sw, sql); each captures the symbol name and, where useful, its parameters.
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
const maxJoinLines = 3

// unbalanced reports whether s opens more parentheses than it closes.
func unbalanced(s string) bool {
	return strings.Count(s, "(") > strings.Count(s, ")")
}

// joinUnclosed joins up to maxJoinLines following lines onto lines[i] while
// the parameter list remains unclosed, collapsing runs of whitespace.
// It returns the joined logical line and the index of the last line consumed.
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
func ExtractAll(root string, files []scanner.FileInfo) []Symbol {
	// Use one worker per CPU core (at least one) to extract files in parallel.
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

	// Both channels are buffered to the number of files, so producers and workers never block on each other.
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
	close(results)

	var out []Symbol
	for res := range results {
		out = append(out, res.symbols...)
	}

	// Results arrive in any order, so sort by file, line, kind and name to keep the output deterministic.
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
func ExtractAllIncremental(root string, files []scanner.FileInfo, cachedHashes map[string]string, cachedSymbols map[string][]Symbol) []Symbol {
	// Use one worker per CPU core (at least one) to extract files in parallel.
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

	// Both channels are buffered to the number of files, so producers and workers never block on each other.
	tasks := make(chan task, len(files))
	results := make(chan result, len(files))

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				// Reuse the cached symbols when the file's content hash is unchanged; otherwise extract again.
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

	// Results arrive in any order, so sort by file, line, kind and name to keep the output deterministic.
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
func Extract(root string, file scanner.FileInfo) []Symbol {
	if file.Skipped || file.IsBinary {
		return nil
	}

	path := filepath.Join(root, filepath.FromSlash(file.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Pick the extractor that matches the language detected by the scanner.
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

// extractGo parses Go source with the standard parser and returns its types, imports, functions and methods with line ranges; a file that does not parse yields no symbols.
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

// goSignature builds a short signature string from a function name and its parameter names (types are not included).
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

// extractPython scans Python lines for imports, classes and defs, joining multi-line signatures.
func extractPython(path string, data []byte) []Symbol {
	var out []Symbol
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		startLine := i + 1
		trimmed := strings.TrimSpace(lines[i])
		// Decorators never block the def/class on the following line.
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
		if pyOpenerRe.MatchString(trimmed) && unbalanced(trimmed) {
			trimmed, i = joinUnclosed(lines, i)
		}
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

// extractJS scans JavaScript and TypeScript lines for imports, functions, classes and arrow-function constants.
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

// extractRuby scans Ruby lines for require/load imports, methods, classes and modules.
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

// extractSQL scans SQL lines for CREATE TABLE, VIEW and PROCEDURE statements.
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

// extractSwift scans Swift lines for imports, functions and class, struct, enum or protocol types.
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

// extractR scans R lines for library/require imports and function definitions.
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

// extractRust scans Rust lines for use imports, functions and struct, enum, union or trait types.
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

// extractC scans C lines for includes, struct/union/enum types and function definitions or prototypes.
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
			if name == "if" || name == "while" || name == "for" || name == "switch" || name == "return" {
				continue
			}
			out = append(out, Symbol{Name: name, Kind: "function", Signature: trimmed, File: path, StartLine: line, EndLine: line})
		}
	}
	return out
}

// extractCPP scans C++ lines for includes, classes, structs and functions.
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
