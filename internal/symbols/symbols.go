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
	"sort"
	"strings"

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
	pyImportRe = regexp.MustCompile(`^(?:from\s+([A-Za-z0-9_./]+)\s+import\s+(.+)|import\s+(.+))`)
	pyDefRe    = regexp.MustCompile(`^(class|def)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(\(.*)?`)
	jsImportRe = regexp.MustCompile(`^import\s+(.+)`)
	jsFuncRe   = regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(([^)]*)\)`)
	jsClassRe  = regexp.MustCompile(`^(?:export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsArrowRe  = regexp.MustCompile(`^(?:export\s+)?const\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][A-Za-z0-9_$]*)\s*=>`)
)

// ExtractAll extracts symbols for supported files.
func ExtractAll(root string, files []scanner.FileInfo) []Symbol {
	var out []Symbol
	for _, file := range files {
		out = append(out, Extract(root, file)...)
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
func Extract(root string, file scanner.FileInfo) []Symbol {
	if file.Skipped || file.IsBinary {
		return nil
	}

	path := filepath.Join(root, filepath.FromSlash(file.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	switch file.Language {
	case "Go":
		return extractGo(file.Path, data)
	case "Python":
		return extractPython(file.Path, data)
	case "JavaScript", "TypeScript":
		return extractJS(file.Path, data)
	default:
		return nil
	}
}

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

func extractPython(path string, data []byte) []Symbol {
	var out []Symbol
	scanner := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for scanner.Scan() {
		line++
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if match := pyImportRe.FindStringSubmatch(trimmed); match != nil {
			name := match[1]
			if name == "" {
				name = match[3]
			}
			out = append(out, Symbol{Name: name, Kind: "import", File: path, StartLine: line, EndLine: line})
			continue
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
				StartLine: line,
				EndLine:   line,
			})
		}
	}
	return out
}

func extractJS(path string, data []byte) []Symbol {
	var out []Symbol
	scanner := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for scanner.Scan() {
		line++
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if match := jsImportRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: strings.TrimSpace(match[1]), Kind: "import", File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := jsFuncRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "function", Signature: trimmed, File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := jsClassRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "class", Signature: trimmed, File: path, StartLine: line, EndLine: line})
			continue
		}
		if match := jsArrowRe.FindStringSubmatch(trimmed); match != nil {
			out = append(out, Symbol{Name: match[1], Kind: "function", Signature: trimmed, File: path, StartLine: line, EndLine: line})
		}
	}
	return out
}
