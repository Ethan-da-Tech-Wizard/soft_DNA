// Package cli implements the minimal CodeLens command-line interface.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"codelens-local/internal/diagram"
	"codelens-local/internal/grounding"
	searchindex "codelens-local/internal/index"
	"codelens-local/internal/llm"
	"codelens-local/internal/scanner"
	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"
	"codelens-local/internal/symbols"
	"codelens-local/internal/validate"
)

const (
	exitOK       = 0
	exitUsage    = 1
	exitBadPath  = 2
	exitInternal = 3
)

const reportName = "CODEBASE_REPORT.md"
const answerName = "ANSWER.md"
const dbName = "codelens.db"

type indexStats struct {
	Path       string
	Files      int
	SearchRows int
	Symbols    int
}

// Run executes the CLI and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	fs := flag.NewFlagSet("codelens", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: codelens [flags] <path>")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Offline codebase explainer. Writes CODEBASE_REPORT.md, architecture.mmd, codelens.db, and optional ANSWER.md.")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Examples:")
		fmt.Fprintln(stderr, "  codelens ./my-project")
		fmt.Fprintln(stderr, "  codelens --out ./report ./my-project")
		fmt.Fprintln(stderr, "  codelens --ask \"where does the CLI start?\" ./my-project")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Flags:")
		fs.PrintDefaults()
	}

	outDir := fs.String("out", "./codelens-out", "output directory")
	maxFileSize := fs.Int64("max-file-size", 2*1024*1024, "maximum file size to read")
	question := fs.String("ask", "", "local FTS question to answer with file citations")
	showVersion := fs.Bool("version", false, "print version")
	var ignores repeatableFlag
	fs.Var(&ignores, "ignore", "additional ignore pattern")

	llmEnabled := fs.Bool("llm", false, "Enable hybrid LLM codebase summarization")
	llmBinary := fs.String("llm-binary", "", "Path to llama-cli executable")
	llmModel := fs.String("llm-model", "", "Path to GGUF model file")
	llmThreads := fs.Int("llm-threads", runtime.NumCPU(), "Inference thread count")
	llmTimeout := fs.Duration("llm-timeout", 60*time.Second, "Subprocess context timeout")
	llmMaxFiles := fs.Int("llm-max-files", 10, "Max count of files to explain")

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	if *showVersion {
		fmt.Fprintf(stdout, "codelens %s\n", version)
		return exitOK
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return exitUsage
	}

	root := fs.Arg(0)
	if err := validateRoot(root); err != nil {
		fmt.Fprintf(stderr, "invalid path: %v\n", err)
		return exitBadPath
	}

	scanIgnores := outputAwareIgnores(root, *outDir, ignores)
	files, err := scanner.Scan(root, scanner.Options{
		MaxFileSize: *maxFileSize,
		Ignore:      scanIgnores,
	})
	if err != nil {
		fmt.Fprintf(stderr, "scan failed: %v\n", err)
		return exitInternal
	}

	stack := stackdetect.Detect(files, root)
	folders, entries := structure.Analyze(files)
	symbolList := symbols.ExtractAll(root, files)

	stats, err := buildIndex(root, *outDir, files, symbolList)
	if err != nil {
		fmt.Fprintf(stderr, "index failed: %v\n", err)
		return exitInternal
	}

	diagramPath, err := diagram.WriteMermaid(*outDir, stack, folders, entries)
	if err != nil {
		fmt.Fprintf(stderr, "diagram failed: %v\n", err)
		return exitInternal
	}

	var llmRes LLMResult
	if *llmEnabled {
		if *llmBinary == "" || *llmModel == "" {
			fmt.Fprintln(stderr, "warning: --llm is enabled but --llm-binary or --llm-model is missing; skipping LLM pass")
		} else {
			dbPath := filepath.Join(*outDir, dbName)
			var err error
			llmRes, err = runLLMPipeline(root, dbPath, *llmBinary, *llmModel, *llmThreads, *llmMaxFiles, *llmTimeout, files)
			if err != nil {
				fmt.Fprintf(stderr, "warning: LLM pass failed: %v; continuing with deterministic report\n", err)
			}
		}
	}

	if err := writeReport(root, *outDir, files, stack, folders, entries, symbolList, stats, diagramPath, llmRes); err != nil {
		fmt.Fprintf(stderr, "failed to write report: %v\n", err)
		return exitInternal
	}

	if strings.TrimSpace(*question) != "" {
		if err := writeAnswer(*outDir, *question, entries); err != nil {
			fmt.Fprintf(stderr, "failed to write answer: %v\n", err)
			return exitInternal
		}
		fmt.Fprintf(stdout, "wrote %s\n", filepath.Join(*outDir, answerName))
	}

	fmt.Fprintf(stdout, "wrote %s\n", filepath.Join(*outDir, reportName))
	return exitOK
}

func validateRoot(path string) error {
	if path == "" {
		return errors.New("path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func writeReport(
	root, outDir string,
	files []scanner.FileInfo,
	stack stackdetect.Report,
	folders []structure.Folder,
	entries []structure.EntryPoint,
	symbolList []symbols.Symbol,
	stats indexStats,
	diagramPath string,
	llmRes LLMResult,
) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, `# CodeLens Local Report

Target: %s

Status: M5 local retrieval Q&A. Model-based chat is not implemented.

## 1. Project Summary

Scanned %d files. Detected stack: %s.

Index: %s (%d files, %d symbols, %d search rows).
Diagram: %s.

## 2. Detected Tech Stack

`, filepath.Clean(root), len(files), escapeMarkdown(stack.Summary), escapeMarkdown(stats.Path), stats.Files, stats.Symbols, stats.SearchRows, escapeMarkdown(diagramPath))

	if len(stack.Tech) == 0 {
		b.WriteString("No manifest-backed stack evidence found.\n")
	} else {
		b.WriteString("| Technology | Category | Evidence |\n")
		b.WriteString("|---|---|---|\n")
		for _, tech := range stack.Tech {
			fmt.Fprintf(&b, "| %s | %s | %s |\n",
				escapeMarkdown(tech.Name),
				escapeMarkdown(tech.Category),
				escapeMarkdown(tech.EvidenceFile),
			)
		}
	}

	b.WriteString("\n## 3. Folder Structure\n\n")
	if len(folders) == 0 {
		b.WriteString("No source files found.\n")
	} else {
		b.WriteString("| Folder | Role | Files |\n")
		b.WriteString("|---|---|---:|\n")
		for _, folder := range folders {
			fmt.Fprintf(&b, "| %s | %s | %d |\n",
				escapeMarkdown(folder.Path),
				escapeMarkdown(folder.Role),
				folder.FileCount,
			)
		}
	}

	b.WriteString("\n## 4. Important Files\n\n")
	important := importantFiles(files, entries, stack)
	if len(important) == 0 {
		b.WriteString("No important files identified yet.\n")
	} else {
		for _, path := range important {
			fmt.Fprintf(&b, "- %s\n", escapeMarkdown(path))
		}
	}

	b.WriteString("\n## 5. Entry Points\n\n")
	if len(entries) == 0 {
		b.WriteString("No entry points identified by convention.\n")
	} else {
		b.WriteString("| Path | Reason |\n")
		b.WriteString("|---|---|\n")
		for _, entry := range entries {
			fmt.Fprintf(&b, "| %s | %s |\n",
				escapeMarkdown(entry.Path),
				escapeMarkdown(entry.Reason),
			)
		}
	}

	b.WriteString(`
## 6. Functions & Classes

`)
	writeSymbolsSection(&b, symbolList, "function", "method", "class", "type", "table", "view", "procedure")

	b.WriteString("\n## 7. Imports/Dependencies\n\n")
	writeSymbolsSection(&b, symbolList, "import")

	b.WriteString("\n## 8. Setup Guess\n\n")
	writeSetupGuess(&b, stack)

	b.WriteString("\n## 9. Plain-English Explanation\n\n")
	fmt.Fprintf(&b, "This project appears to use %s. Start with the entry points above, then read the important files and top-level folders in order.\n", escapeMarkdown(stack.Summary))

	b.WriteString("\n## 10. Risky/Confusing Areas\n\n")
	writeRiskSection(&b, files)

	b.WriteString("\n## 11. AI-Assisted Summaries\n\n")
	if !llmRes.Enabled {
		b.WriteString("*AI summaries disabled or unavailable. Run with --llm flag to generate.*\n")
	} else {
		b.WriteString("*Generated with local LLM (offline).*\n\n")
		if llmRes.ProjectOverview != "" {
			fmt.Fprintf(&b, "### Project Overview\n\n%s\n\n", llmRes.ProjectOverview)
		} else {
			b.WriteString("### Project Overview\n\n*No overview generated.*\n\n")
		}
		b.WriteString("### File Annotations\n\n")
		if len(llmRes.FileSummaries) == 0 {
			b.WriteString("No file annotations generated.\n")
		} else {
			var fileKeys []string
			for f := range llmRes.FileSummaries {
				fileKeys = append(fileKeys, f)
			}
			sort.Strings(fileKeys)

			for _, f := range fileKeys {
				fe := llmRes.FileSummaries[f]
				fmt.Fprintf(&b, "#### %s\n\n%s\n\n", escapeMarkdown(f), fe.FileSummary)
				if len(fe.SymbolExplanations) > 0 {
					b.WriteString("| Symbol | Explanation |\n")
					b.WriteString("|---|---|\n")
					for _, se := range fe.SymbolExplanations {
						fmt.Fprintf(&b, "| %s | %s |\n", escapeMarkdown(se.Name), escapeMarkdown(se.Explanation))
					}
					b.WriteString("\n")
				}
			}
		}
		if len(llmRes.Stalled) > 0 {
			stalledSorted := make([]string, len(llmRes.Stalled))
			copy(stalledSorted, llmRes.Stalled)
			sort.Strings(stalledSorted)
			fmt.Fprintf(&b, "\n*Skipped after repeated validation failures: %s*\n", strings.Join(stalledSorted, ", "))
		}
	}

	b.WriteString(`
## Scanner Inventory

| Path | Language | Lines | Bytes | Status |
|---|---:|---:|---:|---|
`)

	for _, file := range files {
		status := "ok"
		if file.Skipped {
			status = "skipped: " + file.SkipReason
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %s |\n",
			escapeMarkdown(file.Path),
			escapeMarkdown(file.Language),
			file.LineCount,
			file.SizeBytes,
			escapeMarkdown(status),
		)
	}

	return os.WriteFile(filepath.Join(outDir, reportName), []byte(b.String()), 0o644)
}

func buildIndex(root, outDir string, files []scanner.FileInfo, symbolList []symbols.Symbol) (indexStats, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return indexStats{}, err
	}

	dbPath := filepath.Join(outDir, dbName)
	store, err := searchindex.Open(dbPath)
	if err != nil {
		return indexStats{}, err
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		return indexStats{}, err
	}
	if err := store.RebuildWithSymbols(root, files, symbolList); err != nil {
		return indexStats{}, err
	}

	fileRows, searchRows, err := store.Stats()
	if err != nil {
		return indexStats{}, err
	}
	symbolRows, err := store.SymbolCount()
	if err != nil {
		return indexStats{}, err
	}
	return indexStats{Path: dbPath, Files: fileRows, Symbols: symbolRows, SearchRows: searchRows}, nil
}

func writeAnswer(outDir, question string, entries []structure.EntryPoint) error {
	dbPath := filepath.Join(outDir, dbName)
	store, err := searchindex.Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	entryHits := entryPointHits(question, entries)
	limit := 5 - len(entryHits)
	if limit < 0 {
		limit = 0
	}

	hits, err := store.Search(question, limit)
	if err != nil {
		return err
	}
	hits = mergeHits(entryHits, hits)

	var b strings.Builder
	fmt.Fprintf(&b, "# CodeLens Local Answer\n\nQuestion: %s\n\n", escapeMarkdown(strings.TrimSpace(question)))
	b.WriteString("Mode: local FTS retrieval only. No model, network call, or target-code execution was used.\n\n")
	b.WriteString("## Answer\n\n")
	if len(hits) == 0 {
		b.WriteString("No indexed matches were found for this question.\n")
	} else {
		b.WriteString("The most relevant indexed files are:\n\n")
		for _, hit := range hits {
			fmt.Fprintf(&b, "- %s\n", escapeMarkdown(hit.Path))
		}
	}

	b.WriteString("\n## Sources\n\n")
	if len(hits) == 0 {
		b.WriteString("No sources.\n")
	} else {
		for _, hit := range hits {
			fmt.Fprintf(&b, "- %s\n", escapeMarkdown(hit.Path))
		}
	}

	return os.WriteFile(filepath.Join(outDir, answerName), []byte(b.String()), 0o644)
}

func outputAwareIgnores(root, outDir string, userIgnores []string) []string {
	ignores := append([]string{}, userIgnores...)
	rel, ok := relInside(root, outDir)
	if !ok || rel == "." {
		return ignores
	}
	ignores = append(ignores, filepath.ToSlash(rel)+"/")
	return ignores
}

func relInside(root, path string) (string, bool) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

func entryPointHits(question string, entries []structure.EntryPoint) []searchindex.Hit {
	question = strings.ToLower(question)
	if !strings.Contains(question, "start") && !strings.Contains(question, "entry") && !strings.Contains(question, "main") {
		return nil
	}

	hits := make([]searchindex.Hit, 0, len(entries))
	for _, entry := range entries {
		hits = append(hits, searchindex.Hit{Path: entry.Path})
	}
	return hits
}

func mergeHits(priority, fallback []searchindex.Hit) []searchindex.Hit {
	seen := make(map[string]struct{})
	var out []searchindex.Hit
	add := func(hit searchindex.Hit) {
		if hit.Path == "" {
			return
		}
		if _, ok := seen[hit.Path]; ok {
			return
		}
		seen[hit.Path] = struct{}{}
		out = append(out, hit)
	}
	for _, hit := range priority {
		add(hit)
	}
	for _, hit := range fallback {
		add(hit)
	}
	if len(out) > 5 {
		return out[:5]
	}
	return out
}

func writeSymbolsSection(b *strings.Builder, symbolList []symbols.Symbol, kinds ...string) {
	allowed := make(map[string]struct{}, len(kinds))
	for _, kind := range kinds {
		allowed[kind] = struct{}{}
	}

	var selected []symbols.Symbol
	for _, symbol := range symbolList {
		if _, ok := allowed[symbol.Kind]; ok {
			selected = append(selected, symbol)
		}
	}
	if len(selected) == 0 {
		b.WriteString("No symbols identified.\n")
		return
	}

	b.WriteString("| File | Kind | Name | Line | Signature |\n")
	b.WriteString("|---|---|---|---:|---|\n")
	for _, symbol := range selected {
		fmt.Fprintf(b, "| %s | %s | %s | %d | %s |\n",
			escapeMarkdown(symbol.File),
			escapeMarkdown(symbol.Kind),
			escapeMarkdown(symbol.Name),
			symbol.StartLine,
			escapeMarkdown(symbol.Signature),
		)
	}
}

func writeSetupGuess(b *strings.Builder, stack stackdetect.Report) {
	if len(stack.Tech) == 0 {
		b.WriteString("No setup command guessed from manifests.\n")
		return
	}

	commands := setupCommands(stack)
	for _, command := range commands {
		fmt.Fprintf(b, "- `%s`\n", command)
	}
}

func setupCommands(stack stackdetect.Report) []string {
	evidence := make(map[string]bool, len(stack.Tech))
	for _, t := range stack.Tech {
		evidence[t.EvidenceFile] = true
	}

	seen := make(map[string]struct{})
	var out []string
	add := func(command string) {
		if _, ok := seen[command]; ok {
			return
		}
		seen[command] = struct{}{}
		out = append(out, command)
	}

	for _, tech := range stack.Tech {
		switch tech.EvidenceFile {
		case "requirements.txt":
			add("python -m pip install -r requirements.txt")
			add("python main.py")
		case "pyproject.toml":
			switch {
			case evidence["poetry.lock"]:
				add("poetry install")
			case evidence["Pipfile"]:
				add("pipenv install")
			default:
				add("pip install .")
			}
		case "Cargo.toml":
			add("cargo build")
			add("cargo run")
		case "schema.sql":
			add("sqlite3 db.sqlite < schema.sql")
		case "Package.swift":
			add("swift build")
			add("swift run")
		case "DESCRIPTION":
			add(`Rscript -e "install.packages(c(...))"`)
			add("Rscript main.R")
		case "Gemfile":
			add("bundle install")
			add("bundle exec ruby app.rb")
		case "Rakefile":
			add("rake")
		case "go.mod":
			add("go run ./...")
		case "package.json":
			switch {
			case evidence["yarn.lock"]:
				add("yarn install")
				add("yarn dev")
			case evidence["pnpm-lock.yaml"]:
				add("pnpm install")
				add("pnpm dev")
			case evidence["bun.lockb"]:
				add("bun install")
				add("bun run dev")
			default:
				add("npm install")
				add("npm run dev")
			}
		case "CMakeLists.txt":
			add("cmake -B build")
			add("cmake --build build")
		case "Makefile":
			add("make")
		case "configure.ac":
			add("./configure && make")
		}
	}
	sort.Strings(out)
	return out
}

func writeRiskSection(b *strings.Builder, files []scanner.FileInfo) {
	var risks []string
	for _, file := range files {
		if file.Skipped {
			risks = append(risks, fmt.Sprintf("%s was skipped (%s).", file.Path, file.SkipReason))
		}
	}
	if len(risks) == 0 {
		b.WriteString("No scanner-level risks identified.\n")
		return
	}
	sort.Strings(risks)
	for _, risk := range risks {
		fmt.Fprintf(b, "- %s\n", escapeMarkdown(risk))
	}
}

func importantFiles(files []scanner.FileInfo, entries []structure.EntryPoint, stack stackdetect.Report) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(path string) {
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}

	for _, tech := range stack.Tech {
		add(tech.EvidenceFile)
	}
	for _, entry := range entries {
		add(entry.Path)
	}
	for _, file := range files {
		switch filepath.Base(file.Path) {
		case "README.md", "Makefile", "Dockerfile":
			add(file.Path)
		}
	}

	sort.Strings(out)
	return out
}

func escapeMarkdown(value string) string {
	return strings.ReplaceAll(value, "|", `\|`)
}

type repeatableFlag []string

func (r *repeatableFlag) String() string {
	return strings.Join(*r, ",")
}

func (r *repeatableFlag) Set(value string) error {
	*r = append(*r, value)
	return nil
}

// LLMResult holds validated outputs from the offline hybrid LLM summarization pass.
type LLMResult struct {
	Enabled         bool
	ProjectOverview string
	FileSummaries   map[string]validate.FileExplanation
	Stalled         []string
}

// llmRun invokes the LLM subprocess; a package variable so tests can swap in
// a mock runner without spawning a real llama.cpp process.
var llmRun = llm.Run

// maxLLMSymbols caps how many verified symbols are included in a file-explain
// prompt at normal throttle. At repair throttle (after a validation failure)
// the cap is halved — the "retry with a 50% chunk" rule.
const maxLLMSymbols = 5

// runLLMPipeline orchestrates facts grounding, LLM subprocess execution, and post-validation.
func runLLMPipeline(root, dbPath, binaryPath, modelPath string, threads, maxFiles int, timeout time.Duration, files []scanner.FileInfo) (LLMResult, error) {
	gCtx, err := grounding.Build(dbPath)
	if err != nil {
		return LLMResult{}, fmt.Errorf("grounding build: %w", err)
	}

	res := LLMResult{
		Enabled:       true,
		FileSummaries: make(map[string]validate.FileExplanation),
	}

	// Build map for quick FileInfo lookup by path
	infoMap := make(map[string]scanner.FileInfo, len(files))
	for _, f := range files {
		infoMap[f.Path] = f
	}

	// Extract symbols grouped by file and get a sorted list of unique file paths
	fileSymbols := make(map[string][]grounding.SymbolEntry)
	for _, sym := range gCtx.TopSymbols {
		fileSymbols[sym.File] = append(fileSymbols[sym.File], sym)
	}
	var fileList []string
	for f := range fileSymbols {
		fileList = append(fileList, f)
	}
	sort.Strings(fileList)

	if maxFiles > 0 && len(fileList) > maxFiles {
		fileList = fileList[:maxFiles]
	}

	diag := llm.NewLoopState(1+len(fileList), os.Stderr)

	// 1. Project Overview Summarization (Segment 0)
	projPrompt, err := grounding.FormatProjectOverviewPrompt(gCtx)
	if err == nil {
		projGrammar, projCleanup, gErr := llm.WriteGrammar("project_overview.gbnf")
		if gErr == nil {
			defer projCleanup()
			cfg := llm.Config{
				BinaryPath:     binaryPath,
				ModelPath:      modelPath,
				GrammarPath:    projGrammar,
				Threads:        threads,
				MaxTokens:      512,
				Temperature:    0.1,
				Timeout:        timeout,
				MaxOutputBytes: 8192,
			}
			for {
				diag.BeginTurn()
				out, runErr := llmRun(context.Background(), cfg, projPrompt)

				var pe validate.ProjectExplanation
				valid := false
				if runErr != nil {
					fmt.Fprintf(os.Stderr, "warning: LLM project overview run failed: %v\n", runErr)
				} else {
					var valErr error
					pe, valErr = validate.ValidateProjectExplanation(out)
					if valErr != nil {
						fmt.Fprintf(os.Stderr, "warning: LLM project overview validation failed: %v (raw: %q)\n", valErr, string(out))
					} else {
						valid = true
					}
				}
				diag.EndTurn(valid)

				if valid {
					res.ProjectOverview = pe.ProjectDescription
					break
				}
				if diag.Stalled() {
					fmt.Fprintf(os.Stderr, "warning: LLM project overview stalled after %d failed turns; skipping\n", diag.Stall)
					diag.ForceAdvance()
					break
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "warning: LLM project overview grammar write failed: %v\n", gErr)
			diag.BeginTurn()
			diag.EndTurn(false)
			diag.ForceAdvance()
		}
	} else {
		fmt.Fprintf(os.Stderr, "warning: LLM project overview prompt format failed: %v\n", err)
		diag.BeginTurn()
		diag.EndTurn(false)
		diag.ForceAdvance()
	}

	// 2. Per-File Summarization (Segments 1 to N)
	if len(fileList) > 0 {
		fileGrammar, fileCleanup, gErr := llm.WriteGrammar("file_explain.gbnf")
		if gErr == nil {
			defer fileCleanup()
			cfg := llm.Config{
				BinaryPath:     binaryPath,
				ModelPath:      modelPath,
				GrammarPath:    fileGrammar,
				Threads:        threads,
				MaxTokens:      512,
				Temperature:    0.1,
				Timeout:        timeout,
				MaxOutputBytes: 8192,
			}
			res.FileSummaries, res.Stalled = summarizeFiles(root, fileList, infoMap, fileSymbols, cfg, diag, os.Stderr)
		} else {
			fmt.Fprintf(os.Stderr, "warning: LLM file explain grammar write failed: %v\n", gErr)
		}
	}

	return res, nil
}

// summarizeFiles runs the boundary-safe summarization loop (BRSL) over
// fileList: each file is one segment; a segment only counts as covered once
// its LLM output passes validation. Failed turns freeze the pointer, drop the
// throttle to llm.RhoRepair (which halves the symbol chunk fed to the prompt),
// and retry the same file. After more than llm.StallLimit consecutive failures
// the segment is declared an atomic block stall: a cutoff summary is forced
// and the loop moves on so a single bad file can never wedge the whole pass.
func summarizeFiles(
	root string,
	fileList []string,
	infoMap map[string]scanner.FileInfo,
	fileSymbols map[string][]grounding.SymbolEntry,
	cfg llm.Config,
	diag *llm.LoopState,
	warnOut io.Writer,
) (map[string]validate.FileExplanation, []string) {
	summaries := make(map[string]validate.FileExplanation)
	var stalled []string

	for _, f := range fileList {
		info, ok := infoMap[f]
		if !ok {
			continue
		}

		for {
			symCap := maxLLMSymbols
			if diag.Rho < llm.RhoNormal {
				symCap = maxLLMSymbols / 2
			}
			syms := fileSymbols[f]
			if len(syms) > symCap {
				syms = syms[:symCap]
			}

			// T3: Read file content and extract bounded excerpts starting at each symbol's StartLine
			var excerpts []grounding.SymbolExcerpt
			maxLinesPerSymbol := 12
			maxLinesPerFile := 60
			maxBytesPerFile := 2048
			if diag.Rho < llm.RhoNormal {
				maxLinesPerSymbol = 6
				maxLinesPerFile = 30
				maxBytesPerFile = 1024
			}

			rel, relOk := relInside(root, filepath.Join(root, filepath.FromSlash(f)))
			if relOk {
				targetPath := filepath.Join(root, filepath.FromSlash(rel))
				data, readErr := os.ReadFile(targetPath)
				if readErr == nil {
					rawLines := strings.Split(string(data), "\n")
					totalLinesAdded := 0
					totalBytesAdded := 0

					for _, s := range syms {
						if totalLinesAdded >= maxLinesPerFile || totalBytesAdded >= maxBytesPerFile {
							break
						}

						startIdx := s.StartLine - 1
						if startIdx < 0 {
							startIdx = 0
						}
						if startIdx >= len(rawLines) {
							continue
						}

						var excerptLines []string
						symbolLinesCount := 0
						symbolBytesCount := 0

						for i := startIdx; i < len(rawLines); i++ {
							if symbolLinesCount >= maxLinesPerSymbol || (totalLinesAdded+symbolLinesCount) >= maxLinesPerFile {
								break
							}

							line := rawLines[i]
							// Injection defense: strip the literal delimiter ">>>"
							line = strings.ReplaceAll(line, ">>>", "")
							lineWithNL := line + "\n"

							if (totalBytesAdded + symbolBytesCount + len(lineWithNL)) > maxBytesPerFile {
								break
							}

							excerptLines = append(excerptLines, lineWithNL)
							symbolLinesCount++
							symbolBytesCount += len(lineWithNL)
						}

						if len(excerptLines) > 0 {
							excerpts = append(excerpts, grounding.SymbolExcerpt{
								Name:  s.Name,
								Lines: excerptLines,
							})
							totalLinesAdded += symbolLinesCount
							totalBytesAdded += symbolBytesCount
						}
					}
				}
			}

			fc := grounding.FileContext{
				File:      f,
				Language:  info.Language,
				LineCount: info.LineCount,
				Symbols:   syms,
				Excerpts:  excerpts,
			}
			prompt, pErr := grounding.FormatFileExplainPrompt(fc)
			if pErr != nil {
				// Deterministic failure — retrying cannot help; skip the file.
				fmt.Fprintf(warnOut, "warning: LLM file explain prompt format failed for %s: %v\n", f, pErr)
				break
			}

			diag.BeginTurn()
			raw, runErr := llmRun(context.Background(), cfg, prompt)

			var fe validate.FileExplanation
			valid := false
			if runErr != nil {
				fmt.Fprintf(warnOut, "warning: LLM file explain run failed for %s: %v\n", f, runErr)
			} else {
				// Build list of expected/allowed symbol names in this file for validation
				var allowedNames []string
				for _, s := range syms {
					allowedNames = append(allowedNames, s.Name)
				}
				var errs []validate.ValidationError
				fe, errs = validate.ValidateFileExplanation(raw, allowedNames)
				if len(errs) > 0 {
					fmt.Fprintf(warnOut, "warning: LLM file explain validation warnings for %s: %v (raw: %q)\n", f, errs, string(raw))
				}
				valid = len(errs) == 0 || fe.FileSummary != ""
			}
			diag.EndTurn(valid)

			if valid {
				summaries[f] = fe
				break
			}
			if diag.Stalled() {
				fmt.Fprintf(warnOut, "warning: atomic block stall on %s after %d failed turns; forcing cutoff summary\n", f, diag.Stall)
				if fe.FileSummary != "" {
					summaries[f] = fe
				} else {
					stalled = append(stalled, f)
				}
				diag.ForceAdvance()
				break
			}
		}
	}

	return summaries, stalled
}
