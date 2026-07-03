package gui

import (
	"os"
	"strings"
	"testing"

	"codelens-local/internal/cli"
	"codelens-local/internal/scanner"
	"codelens-local/internal/stackdetect"
	"codelens-local/internal/structure"
	"codelens-local/internal/symbols"
	"codelens-local/internal/validate"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// TestMain installs Fyne's headless test driver once for the whole package,
// so widget-construction code under test (which calls theme.Color and other
// fyne.CurrentApp()-dependent APIs) works with no real display, no GPU, and
// no window manager — exactly the environment CI runs in.
func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

func samplePipelineResult() cli.PipelineResult {
	return cli.PipelineResult{
		Root:   "/project",
		OutDir: "/project/codelens-out",
		Files: []scanner.FileInfo{
			{Path: "main.go", Language: "Go", LineCount: 10},
		},
		Stack: stackdetect.Report{
			Summary: "Go",
			Tech:    []stackdetect.Tech{{Name: "Go", Category: "language", EvidenceFile: "go.mod"}},
		},
		Folders: []structure.Folder{{Path: ".", Role: "Root files", FileCount: 1}},
		Entries: []structure.EntryPoint{{Path: "main.go", Reason: "Go main package convention"}},
		Symbols: []symbols.Symbol{
			{File: "main.go", Kind: "function", Name: "main", StartLine: 3, Signature: "func main()"},
			{File: "main.go", Kind: "import", Name: "fmt", StartLine: 1},
		},
		Index: cli.IndexStats{Path: "/project/codelens-out/codelens.db", Files: 1, Symbols: 2, SearchRows: 1},
	}
}

func TestBuildReportTabsHasExpectedTabsInOrder(t *testing.T) {
	res := samplePipelineResult()
	a := &App{}

	tabs, ok := a.buildReportTabs(res).(*container.AppTabs)
	if !ok {
		t.Fatalf("buildReportTabs did not return *container.AppTabs")
	}

	want := []string{
		"Summary", "Tech Stack", "Folders", "Entry Points",
		"Functions & Classes", "Imports", "AI Summaries", "Diagram", "Ask",
	}
	if len(tabs.Items) != len(want) {
		t.Fatalf("tab count = %d, want %d", len(tabs.Items), len(want))
	}
	for i, name := range want {
		if tabs.Items[i].Text != name {
			t.Errorf("tab %d = %q, want %q", i, tabs.Items[i].Text, name)
		}
		if tabs.Items[i].Content == nil {
			t.Errorf("tab %q has nil content", name)
		}
	}
}

func TestBuildReportTabsHandlesEmptyResult(t *testing.T) {
	a := &App{}
	if got := a.buildReportTabs(cli.PipelineResult{}); got == nil {
		t.Fatal("buildReportTabs(empty) returned nil")
	}
}

func TestNewAISummariesTabDisabledShowsPlaceholder(t *testing.T) {
	res := samplePipelineResult() // LLM zero-value: Enabled=false
	if got := (&App{}).newAISummariesTab(res); got == nil {
		t.Fatal("newAISummariesTab returned nil")
	}
}

func TestNewAISummariesTabRendersOverviewAndFiles(t *testing.T) {
	res := samplePipelineResult()
	res.LLM = cli.LLMResult{
		Enabled:         true,
		ProjectOverview: "A small Go program.",
		FileSummaries: map[string]validate.FileExplanation{
			"main.go": {
				FileSummary:        "Entry point.",
				SymbolExplanations: []validate.SymbolExplanation{{Name: "main", Explanation: "Starts the program."}},
			},
		},
		Stalled: []string{"broken.go"},
	}
	if got := (&App{}).newAISummariesTab(res); got == nil {
		t.Fatal("newAISummariesTab returned nil")
	}
}

func TestFilterSymbolsByNameAndFile(t *testing.T) {
	all := []symbols.Symbol{
		{File: "main.go", Name: "main"},
		{File: "util.go", Name: "helper"},
		{File: "util.go", Name: "otherFn"},
	}

	if got := filterSymbols(all, ""); len(got) != 3 {
		t.Fatalf("empty query = %d results, want 3", len(got))
	}
	if got := filterSymbols(all, "  "); len(got) != 3 {
		t.Fatalf("whitespace query = %d results, want 3", len(got))
	}
	if got := filterSymbols(all, "MAIN"); len(got) != 1 || got[0].Name != "main" {
		t.Fatalf("case-insensitive name filter = %+v, want just 'main'", got)
	}
	if got := filterSymbols(all, "util"); len(got) != 2 {
		t.Fatalf("file filter = %d results, want 2", len(got))
	}
	if got := filterSymbols(all, "nope"); len(got) != 0 {
		t.Fatalf("no-match filter = %d results, want 0", len(got))
	}
}

func TestNewDiagramViewDoesNotPanicOnEmptyInputs(t *testing.T) {
	if got := newDiagramView(stackdetect.Report{}, nil, nil, 1.0); got == nil {
		t.Fatal("newDiagramView returned nil")
	}
}

func TestNewDiagramTabBuildsWithZoomControls(t *testing.T) {
	if got := newDiagramTab(stackdetect.Report{}, nil, nil); got == nil {
		t.Fatal("newDiagramTab returned nil")
	}
}

func TestNewAskTabBuildsWithoutWindow(t *testing.T) {
	// newAskTab must not touch a.win/a.fyneApp at construction time, only
	// inside its Send callback — otherwise every headless test would need a
	// real window.
	a := &App{}
	if got := a.newAskTab(samplePipelineResult()); got == nil {
		t.Fatal("newAskTab returned nil")
	}
}

// walkLabels collects the text of every *widget.Label reachable from obj.
func walkLabels(obj fyne.CanvasObject, out *[]string) {
	switch v := obj.(type) {
	case *widget.Label:
		*out = append(*out, v.Text)
	case *fyne.Container:
		for _, child := range v.Objects {
			walkLabels(child, out)
		}
	case *container.Scroll:
		walkLabels(v.Content, out)
	}
}

func labelsContain(obj fyne.CanvasObject, want string) bool {
	var texts []string
	walkLabels(obj, &texts)
	for _, txt := range texts {
		if strings.Contains(txt, want) {
			return true
		}
	}
	return false
}

func TestAISummariesTabExplainsSkippedPass(t *testing.T) {
	// User enabled AI, but the pass never ran (e.g. missing binary):
	// the tab must show the pipeline's warning, not a generic "off" message.
	a := &App{settings: Settings{LLMEnabled: true}}
	res := samplePipelineResult()
	res.Warnings = []string{"--llm is enabled but the LLM binary or model path is missing; skipping LLM pass"}

	tab := a.newAISummariesTab(res)
	if !labelsContain(tab, "did not run") || !labelsContain(tab, "binary or model path is missing") {
		t.Fatalf("skipped-pass tab does not surface the pipeline warning")
	}
}

func TestAISummariesTabExplainsTimeouts(t *testing.T) {
	// The pass ran but every call timed out: the tab must say so and point
	// at the timeout setting, using the tallies from the diagnostic stream.
	a := &App{settings: Settings{LLMEnabled: true}, scanTimeouts: 11, scanFailedTurns: 11}
	res := samplePipelineResult()
	res.LLM = cli.LLMResult{Enabled: true, Stalled: []string{"main.go"}}

	tab := a.newAISummariesTab(res)
	if !labelsContain(tab, "11 model call(s) timed out") || !labelsContain(tab, "Timeout per call") {
		t.Fatalf("timeout tab does not explain the timeout failure")
	}
}

func TestAISummariesTabExplainsValidationFailures(t *testing.T) {
	a := &App{settings: Settings{LLMEnabled: true}, scanFailedTurns: 4}
	res := samplePipelineResult()
	res.LLM = cli.LLMResult{Enabled: true}

	tab := a.newAISummariesTab(res)
	if !labelsContain(tab, "4 model turn(s) failed validation") {
		t.Fatalf("validation-failure tab does not explain itself")
	}
}
