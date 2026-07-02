package grounding

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"
)

// FileContext holds the per-file facts passed to the file explanation prompt.
// Raw source content is intentionally excluded to prevent prompt injection.
type FileContext struct {
	File      string
	Language  string
	LineCount int
	Symbols   []SymbolEntry
}

// folderRoleEntry is a key-value pair for deterministic template rendering
// of FolderRoles (which is a map in GroundingContext).
type folderRoleEntry struct {
	Folder string
	Role   string
}

// projectOverviewData is the internal data struct passed to the project
// overview template. FolderRoles is pre-sorted to satisfy the G-5 guardrail.
type projectOverviewData struct {
	Root        string
	FileCount   int
	Stack       []TechEntry
	EntryPoints []string
	FolderRoles []folderRoleEntry
	TopSymbols  []SymbolEntry
}

// fileExplainPrompt is the per-file explanation prompt template.
// Three zones are kept strictly separate:
//   - [SYSTEM FACTS]: immutable verified data from SQLite
//   - [TASK]:         what the LLM must do, with explicit prohibitions
//   - [OUTPUT SCHEMA]: the exact JSON structure required
const fileExplainPrompt = `[SYSTEM FACTS — IMMUTABLE — DO NOT MODIFY OR ADD TO THIS LIST]
File: {{.File}}
Language: {{.Language}}
Size: {{.LineCount}} lines
Verified symbols in this file (from static analysis):
{{- range .Symbols}}
  - {{.Kind}} '{{.Name}}' at line {{.StartLine}}{{if .Signature}} | signature: {{.Signature}}{{end}}
{{- end}}

[TASK]
You are explaining source code to a developer.
Describe ONLY the verified symbols listed above.
Do NOT mention any symbol, function, class, file, or module that is not in the SYSTEM FACTS list.
For each symbol, write exactly one sentence describing its purpose based on its name and signature.
Do NOT hallucinate dependencies, return types, or behaviors not evident from the name/signature.

[OUTPUT SCHEMA]
Respond ONLY with valid JSON matching this structure (no markdown, no commentary):
{"file_summary":"<one sentence summary of the file purpose>","symbol_explanations":[{"name":"<symbol name from FACTS>","explanation":"<one sentence>"}]}`

// projectOverviewPrompt is the whole-project explanation prompt template.
// FolderRoles is rendered from a pre-sorted []folderRoleEntry, not a map,
// to guarantee deterministic output order (G-5).
const projectOverviewPrompt = `[SYSTEM FACTS — IMMUTABLE]
Project root: {{.Root}}
Total files: {{.FileCount}}
Detected tech stack:
{{- range .Stack}}
  - {{.Name}} ({{.Category}}, evidence: {{.Evidence}})
{{- end}}
Entry points:
{{- range .EntryPoints}}
  - {{.}}
{{- end}}
Top-level folders:
{{- range .FolderRoles}}
  - {{.Folder}}: {{.Role}}
{{- end}}
Key symbols (verified):
{{- range .TopSymbols}}
  - [{{.Kind}}] {{.Name}} in {{.File}}:{{.StartLine}}
{{- end}}

[TASK]
Write a 3-5 sentence plain English description of this project suitable for a new developer.
Reference ONLY the entry points, stack technologies, and symbol names listed in SYSTEM FACTS.
Do not mention any file, library, or function not in the list above.

[OUTPUT SCHEMA]
Respond ONLY with valid JSON (no markdown, no commentary):
{"project_description":"<3-5 sentence description>"}`

var (
	fileExplainTmpl     = template.Must(template.New("fileExplain").Parse(fileExplainPrompt))
	projectOverviewTmpl = template.Must(template.New("projectOverview").Parse(projectOverviewPrompt))
)

// FormatFileExplainPrompt renders the per-file explanation prompt using the
// verified symbols in fc. Raw source content is never included.
func FormatFileExplainPrompt(fc FileContext) (string, error) {
	var buf bytes.Buffer
	if err := fileExplainTmpl.Execute(&buf, fc); err != nil {
		return "", fmt.Errorf("grounding: render file explain prompt: %w", err)
	}
	return buf.String(), nil
}

// FormatProjectOverviewPrompt renders the project overview prompt using the
// verified facts in gc. FolderRoles are sorted for deterministic output (G-5).
func FormatProjectOverviewPrompt(gc GroundingContext) (string, error) {
	folders := make([]folderRoleEntry, 0, len(gc.FolderRoles))
	for folder, role := range gc.FolderRoles {
		folders = append(folders, folderRoleEntry{Folder: folder, Role: role})
	}
	sort.Slice(folders, func(i, j int) bool {
		return folders[i].Folder < folders[j].Folder
	})

	data := projectOverviewData{
		Root:        gc.Root,
		FileCount:   gc.FileCount,
		Stack:       gc.Stack,
		EntryPoints: gc.EntryPoints,
		FolderRoles: folders,
		TopSymbols:  gc.TopSymbols,
	}

	var buf bytes.Buffer
	if err := projectOverviewTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("grounding: render project overview prompt: %w", err)
	}
	return buf.String(), nil
}
