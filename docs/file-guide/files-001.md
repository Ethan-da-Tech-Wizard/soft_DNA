# Files 1 of 1 — CodeLens Local

[All pages](../FILE_GUIDE.md) · [Project guide](../PROJECT_GUIDE.md)

| File | Role and useful details |
| --- | --- |
| [.github/workflows/ci.yml](../../.github/workflows/ci.yml) | GitHub Actions workflow for ci; defines automation triggers, jobs, and build/check steps. |
| [.gitignore](../../.gitignore) | Defines local data, credentials, dependencies, and generated outputs that Git should leave untracked. |
| [ARCHITECTURE_TODO.md](../../ARCHITECTURE_TODO.md) | Documentation: ARCHITECTURE_TODO.md — Architecture Explainer (planning doc). Covers 1. Purpose; 2. MVP scope; 3. Shape legend. |
| [CODEBASE_GUIDE.md](../../CODEBASE_GUIDE.md) | Documentation: CodeLens Local — Complete Codebase Guide. Covers 0. The big picture; 1. `cmd/codelens/main.go` — the entry point (13 lines); 2. `internal/cli` — orchestration, report writing, and the BRSL loop. |
| [KISS_LOOP_EQUATION.md](../../KISS_LOOP_EQUATION.md) | Documentation: The KISS Loop Equation. Covers 0. KISS itself, as an equation; 1. Definitions; 2. The update rule (one turn = one application of F). |
| [LICENSE](../../LICENSE) | Preserved license, copyright, or attribution terms; read the full text before reusing the covered component. |
| [Makefile](../../Makefile) | Tracked Makefile resource in .; 2,935 bytes. |
| [POLISH_CHECKLIST.md](../../POLISH_CHECKLIST.md) | Documentation: CodeLens Local — Polish Checklist. Covers Desktop GUI (`codelens-gui`); CLI / GUI parity; CLI / shared pipeline. |
| [PROJECT_NOTES.md](../../PROJECT_NOTES.md) | Preserved detailed setup, usage, architecture, and development notes from the earlier README. |
| [README.md](../../README.md) | Project overview, current status, entry points, setup navigation, and documentation links. |
| [ROADMAP.md](../../ROADMAP.md) | Documentation: Roadmap. Covers v1.0 Complete; v1.2 Complete (Diagnostics & Integrity Gate); v1.3 Complete (Native Desktop App). |
| [TODO_v1.2.md](../../TODO_v1.2.md) | Documentation: TODO — CodeLens Local v1.2.0 "Closing the Honest Gaps". Covers Global rules (apply to EVERY task — read first); T1 — Stop mislabeling Makefile projects as C; T2 — Move forced-cutoff placeholders out of the annotation list. |
| [TODO_v2.md](../../TODO_v2.md) | Documentation: TODO — CodeLens Local v2.0 "Working AI & A Diagram Worth Clicking". Covers Global rules (apply to EVERY task — read first); P0 — Make the AI actually produce output; Diagnosis (verified in code, do not re-derive). |
| [cmd/codelens-gui/main.go](../../cmd/codelens-gui/main.go) | Source module for main; defines `main`. |
| [cmd/codelens/main.go](../../cmd/codelens/main.go) | Source module for main; defines `main`. |
| [cmd/codelens/main_test.go](../../cmd/codelens/main_test.go) | Test implementation for main test; defines `TestVersionFlagEndToEnd`. |
| [docs/DEPENDENCIES.csv](../../docs/DEPENDENCIES.csv) | Machine-readable inventory of declared and locked dependencies, versions, reported licenses, and evidence URLs. |
| [docs/FILE_GUIDE.md](../../docs/FILE_GUIDE.md) | Index of the complete tracked-file guide, with one entry for every file and links to the paginated descriptions. |
| [docs/LICENSE_REVIEW.md](../../docs/LICENSE_REVIEW.md) | Original-code license, preserved component notices, dependency-license evidence, and unresolved distribution questions. |
| [docs/PROJECT_GUIDE.md](../../docs/PROJECT_GUIDE.md) | Maintainer orientation: purpose, important entry points, source layout, commands, and project-specific boundaries. |
| [docs/file-guide/files-001.md](../../docs/file-guide/files-001.md) | One page of the tracked-file catalog; links each listed path to its role and source-derived details. |
| [files.zip](../../files.zip) | ZIP archive for files; 26,209 bytes. Consult the project guide and component notices for its purpose and distribution limits. |
| [files/00_PROJECT_CHARTER.md](../../files/00_PROJECT_CHARTER.md) | Documentation: CodeLens Local — Project Charter. Covers 1. Vision; 2. Problem statement; 3. Goals. |
| [files/01_PRD.md](../../files/01_PRD.md) | Documentation: CodeLens Local — Product Requirements Document (PRD). Covers 1. Overview; 2. Target users; 3. User stories (v1). |
| [files/02_SRD.md](../../files/02_SRD.md) | Documentation: CodeLens Local — Software Requirements Document (SRD). Covers 1. Functional requirements; Scanner (`scanner`); Stack detector (`stackdetect`). |
| [files/03_BUILD_PLAN.md](../../files/03_BUILD_PLAN.md) | Documentation: CodeLens Local — Build Plan & Test Strategy. Covers 1. Build philosophy; 2. Phased plan (mapped to milestones); Phase 0 — Project bootstrap (→ M0). |
| [files/04_AGENT_GUIDE.md](../../files/04_AGENT_GUIDE.md) | Documentation: CodeLens Local — AI Agent Build Guide. Covers 1. Read order & ground rules; 2. Repository layout (target); 3. Guardrails (non-negotiable). |
| [files/05_ADR_LOG.md](../../files/05_ADR_LOG.md) | Documentation: CodeLens Local — ADR Log. Covers ADR format; ADR-000N — ; Context. |
| [files/06_SECURITY_MODEL.md](../../files/06_SECURITY_MODEL.md) | Documentation: CodeLens Local — Security Model. Covers 1. Assets to protect; 2. Hard rules; 3. Threats and controls. |
| [files/07_AGENT_TASK_PROMPTS.md](../../files/07_AGENT_TASK_PROMPTS.md) | Documentation: CodeLens Local — Agent Task Prompts. Covers Section 1 — Documentation Polish Prompt; Section 2 — Repo Skeleton Prompt; Section 3 — M0 Ticket Prompt. |
| [files/08_LANGUAGE_SUPPORT_ROADMAP.md](../../files/08_LANGUAGE_SUPPORT_ROADMAP.md) | Documentation: CodeLens Local — Language Support Expansion Roadmap. Covers 🐍 Python (Existing + Extensions); 🇨 C; 🦀 Rust. |
| [files/09_AGENT_SEQUENTIAL_PROMPTS.md](../../files/09_AGENT_SEQUENTIAL_PROMPTS.md) | Documentation: CodeLens Local — Sequential AI Agent Prompts. Covers Prompt 1 — SHA-256 Release Checksums; Prompt 2 — pyproject.toml & Python Tooling Guesses; Prompt 3 — C Language Support. |
| [files/README.md](../../files/README.md) | Documentation: CodeLens Local — Documentation Set. Covers Headline decisions (locked); Read in this order; How to use this with an AI agent. |
| [files/handover_report.md](../../files/handover_report.md) | Documentation: CodeLens Local — Final Handover Report. Covers 1. Milestone Completion Status; M0 — CLI Skeleton; M1 — File Scanner. |
| [go.mod](../../go.mod) | Go module identity, toolchain version, and dependency requirements; paired with go.sum for downloaded module checksums. |
| [go.sum](../../go.sum) | Tracked go resource in .; 11,509 bytes. |
| [internal/cli/cli.go](../../internal/cli/cli.go) | Source module for cli; defines `Run`, `IsInvalidRoot`, `RunPipeline`, `WriteMarkdownReport`, `Answer`, `validateRoot`, `writeReport` and 17 more symbols. |
| [internal/cli/cli_test.go](../../internal/cli/cli_test.go) | Test implementation for cli test; defines `TestRunWritesReportForDirectory`, `TestRunPopulatesFullReportSections`, `TestRunFullReportGoldenPrefix`, `TestRunWritesLocalAnswer`, `TestRunAnswerPrefersEntryPointQuestions`, `TestRunIgnoresOutputDirectoryWhenScanningRoot`, `TestRunRejectsMissingPath` and 30 more symbols. |
| [internal/diagram/diagram.go](../../internal/diagram/diagram.go) | Source module for diagram; defines `WriteMermaid`, `nodeID`, `escape`. |
| [internal/diagram/diagram_test.go](../../internal/diagram/diagram_test.go) | Test implementation for diagram test; defines `TestWriteMermaid`. |
| [internal/grounding/grounding.go](../../internal/grounding/grounding.go) | Source module for grounding; defines `Build`, `to`, `buildFromDB`, `folderRole`. |
| [internal/grounding/grounding_test.go](../../internal/grounding/grounding_test.go) | Test implementation for grounding test; defines `newMemDB`, `initMemDB`, `mustExec`, `TestBuildSuccess`, `TestBuildNoSchema`, `TestBuildFromDBHappyPath`, `detectLanguage` and 21 more symbols. |
| [internal/grounding/prompt.go](../../internal/grounding/prompt.go) | Source module for prompt; defines `passed`, `not`, `FormatAskPrompt`, `FormatFileExplainPrompt`, `FormatProjectOverviewPrompt`. |
| [internal/gui/app.go](../../internal/gui/app.go) | Source module for app; defines `NewApp`. |
| [internal/gui/architecture.go](../../internal/gui/architecture.go) | Source module for architecture; defines `archTopFolder`, `BuildArchDiagram`, `layoutArch`. |
| [internal/gui/architecture_test.go](../../internal/gui/architecture_test.go) | Test implementation for architecture test; defines `sampleResult`, `nodeShape`, `TestBuildArchDiagramShapesAndEdges`, `TestBuildArchDiagramLayeredLayout`, `TestBuildArchDiagramEmptyProject`, `TestBuildArchDiagramUsesLLMSummaryWhenPresent`, `TestConnectionsFor` and 4 more symbols. |
| [internal/gui/architectureview.go](../../internal/gui/architectureview.go) | Source module for architectureview; defines `archShapeColor`, `newArchCanvas`, `archArrow`, `archPanelContent`, `newArchShapeWidget`, `newArchTapCatcher`. |
| [internal/gui/ask.go](../../internal/gui/ask.go) | Source module for ask; defines `formatAnswer`. |
| [internal/gui/assets/gen_icon.go](../../internal/gui/assets/gen_icon.go) | Source module for gen icon; defines `roundedRectAlpha`, `lerp`, `main`. |
| [internal/gui/assets/icon.png](../../internal/gui/assets/icon.png) | Image or interface asset for icon; 128 × 128 pixels. Source/asset rights remain separate from software-license terms. |
| [internal/gui/diagramexport.go](../../internal/gui/diagramexport.go) | Source module for diagramexport; defines `ExportDiagramSVG`, `arrow`, `nodeColorHex`, `svgEscape`. |
| [internal/gui/diagramexport_test.go](../../internal/gui/diagramexport_test.go) | Test implementation for diagramexport test; defines `TestExportDiagramSVG`, `TestExportDiagramSVGEmpty`. |
| [internal/gui/diagramlayout.go](../../internal/gui/diagramlayout.go) | Source module for diagramlayout; defines `ComputeDiagramLayout`, `ComputeDiagramLayoutForScope`, `GetFolderContents`. |
| [internal/gui/diagramlayout_test.go](../../internal/gui/diagramlayout_test.go) | Test implementation for diagramlayout test; defines `TestComputeDiagramLayoutRootAlone`, `TestComputeDiagramLayoutFoldersEntriesAndStack`, `TestComputeDiagramLayoutNoDuplicateEdgesWithoutStack`, `TestComputeDiagramLayoutWrapping`, `TestComputeDiagramLayoutForScope`. |
| [internal/gui/diagramview.go](../../internal/gui/diagramview.go) | Source module for diagramview; defines `legendItem`, `newDiagramView`, `roleColor`, `NewDiagramNodeWidget`. |
| [internal/gui/logwriter.go](../../internal/gui/logwriter.go) | Source module for logwriter. |
| [internal/gui/logwriter_test.go](../../internal/gui/logwriter_test.go) | Test implementation for logwriter test; defines `TestLineWriterSplitsCompleteLines`, `TestLineWriterBuffersPartialLineAcrossWrites`. |
| [internal/gui/report.go](../../internal/gui/report.go) | Source module for report; defines `newTechStackTab`, `newFoldersTab`, `newEntryPointsTab`, `filterSymbols`, `prepareSymbolRows`, `newSymbolsTab`, `formatBytes`. |
| [internal/gui/report_test.go](../../internal/gui/report_test.go) | Test implementation for report test; defines `TestMain`, `samplePipelineResult`, `main`, `TestBuildReportTabsHasExpectedTabsInOrder`, `TestBuildReportTabsHandlesEmptyResult`, `TestNewAISummariesTabDisabledShowsPlaceholder`, `TestNewAISummariesTabRendersOverviewAndFiles` and 14 more symbols. |
| [internal/gui/resources.go](../../internal/gui/resources.go) | Source module for resources. |
| [internal/gui/resources_test.go](../../internal/gui/resources_test.go) | Test implementation for resources test; defines `TestAppIconIsValidPNG`. |
| [internal/gui/settings.go](../../internal/gui/settings.go) | Source module for settings; defines `settingsPath`, `LoadSettings`, `validateLLMConfig`, `autoDetectLlamaCli`, `ScanLocalModels`. |
| [internal/gui/settings_test.go](../../internal/gui/settings_test.go) | Test implementation for settings test; defines `TestSettingsSaveAndLoadRoundTrip`, `TestLoadSettingsDefaultsWhenMissing`, `TestLoadSettingsDefaultsOnCorruptFile`, `TestValidateLLMConfigDisabledIsAlwaysFine`, `TestValidateLLMConfigFlagsEmptyPaths`, `TestValidateLLMConfigFlagsMissingFiles`, `TestValidateLLMConfigPassesWithRealFiles` and 5 more symbols. |
| [internal/gui/settingsdialog.go](../../internal/gui/settingsdialog.go) | Source module for settingsdialog. |
| [internal/gui/tablehelpers.go](../../internal/gui/tablehelpers.go) | Source module for tablehelpers; defines `newLiveTable`, `newSimpleTable`. |
| [internal/index/store.go](../../internal/index/store.go) | Source module for store; defines `Open`, `ftsQuery`, `searchableContent`, `safeJoin`, `insideRoot`, `boolInt`, `searchableSymbols`. |
| [internal/index/store_test.go](../../internal/index/store_test.go) | Test implementation for store test; defines `TestStoreBuildsFTSIndex`, `TestSearchAcceptsNaturalLanguageQuery`, `TestStoreRebuildIsIdempotent`, `TestStoreIndexesSymbolsForSearch`, `TestStoreSchemaCanInitializeTwice`, `openTestStore`. |
| [internal/llm/diagnostics.go](../../internal/llm/diagnostics.go) | Source module for diagnostics; defines `NewLoopState`. |
| [internal/llm/grammars/ask_answer.gbnf](../../internal/llm/grammars/ask_answer.gbnf) | Tracked ask answer resource in internal/llm/grammars; 531 bytes. |
| [internal/llm/grammars/file_explain.gbnf](../../internal/llm/grammars/file_explain.gbnf) | Tracked file explain resource in internal/llm/grammars; 764 bytes. |
| [internal/llm/grammars/project_overview.gbnf](../../internal/llm/grammars/project_overview.gbnf) | Tracked project overview resource in internal/llm/grammars; 558 bytes. |
| [internal/llm/runner.go](../../internal/llm/runner.go) | Source module for runner; defines `when`, `WriteGrammar`, `Run`, `parseAnswerStream`. |
| [internal/llm/runner_test.go](../../internal/llm/runner_test.go) | Test implementation for runner test; defines `TestLimitedWriterBelowLimit`, `TestLimitedWriterExceedsLimitInOneWrite`, `TestLimitedWriterExactLimitThenOneByte`, `TestLimitedWriterAccumulatesAcrossMultipleWrites`, `TestLimitedWriterErrorSticksAfterFirstFailure`, `TestLimitedWriterZeroLimit`, `TestRunMissingBinary` and 6 more symbols. |
| [internal/scanner/ignore.go](../../internal/scanner/ignore.go) | Source module for ignore; defines `newIgnoreMatcher`, `readGitignore`, `matchPattern`. |
| [internal/scanner/scanner.go](../../internal/scanner/scanner.go) | Source module for scanner; defines `Scan`, `shouldSkipDir`, `scanRegularFile`, `scanSymlink`, `skippedInfo`, `baseInfo`, `cleanRel` and 4 more symbols. |
| [internal/scanner/scanner_test.go](../../internal/scanner/scanner_test.go) | Test implementation for scanner test; defines `TestScanInventoriesTextFilesDeterministically`, `TestScanRespectsGitignoreAndExtraIgnores`, `TestScanMarksLargeAndBinaryFilesSkipped`, `TestScanDoesNotFollowOutOfRootSymlink`, `TestSampleFixturesInventory`, `TestDetectSQLLanguage`, `TestDetectSwiftLanguage` and 8 more symbols. |
| [internal/stackdetect/stackdetect.go](../../internal/stackdetect/stackdetect.go) | Source module for stackdetect; defines `Detect`, `detectPackageJSON`, `summary`. |
| [internal/stackdetect/stackdetect_test.go](../../internal/stackdetect/stackdetect_test.go) | Test implementation for stackdetect test; defines `TestDetectPythonStack`, `TestDetectNodeStack`, `TestDetectUnknownStack`, `TestDetectRubyStack`, `TestDetectRustStack`, `TestDetectCStack`, `TestDetectPyprojectWithPoetry` and 10 more symbols. |
| [internal/structure/structure.go](../../internal/structure/structure.go) | Source module for structure; defines `Analyze`, `folders`, `topFolder`, `RoleFor`, `entryPoints`, `entryReason`. |
| [internal/structure/structure_test.go](../../internal/structure/structure_test.go) | Test implementation for structure test; defines `TestAnalyzeFindsPythonEntryPoint`, `TestAnalyzeFindsNodeEntryPoint`, `TestAnalyzeFindsGoCmdEntryPoint`, `TestEntryReasonRuby`, `TestAnalyzeFindsRubyEntryPoint`, `TestEntryReasonSQL`, `TestAnalyzeFindsSQLEntryPoint` and 10 more symbols. |
| [internal/symbols/symbols.go](../../internal/symbols/symbols.go) | Source module for symbols; defines `signature`, `unbalanced`, `joinUnclosed`, `ExtractAll`, `ExtractAllIncremental`, `Extract`, `extractGo` and 11 more symbols. |
| [internal/symbols/symbols_test.go](../../internal/symbols/symbols_test.go) | Test implementation for symbols test; defines `TestExtractPythonSymbols`, `TestExtractTypeScriptSymbols`, `TestExtractGoSymbols`, `TestMalformedFileDoesNotCrash`, `TestExtractSQLSymbols`, `TestExtractSwiftSymbols`, `TestExtractRSymbols` and 20 more symbols. |
| [internal/validate/validate.go](../../internal/validate/validate.go) | Source module for validate; defines `sanitizeText`, `truncateRunes`, `ValidateFileExplanation`, `ValidateAskAnswer`, `ValidateProjectExplanation`. |
| [internal/validate/validate_test.go](../../internal/validate/validate_test.go) | Test implementation for validate test; defines `TestSanitizeTextStripsAbsolutePaths`, `TestSanitizeTextStripsURLs`, `TestSanitizeTextStripsBacktickedCode`, `to`, `TestSanitizeTextStripsMultiplePatterns`, `TestSanitizeTextTrimsWhitespace`, `TestSanitizeTextEmptyString` and 24 more symbols. |
| [scripts/download_llama.sh](../../scripts/download_llama.sh) | Command-line automation for download llama; review prerequisites and side effects before executing. |
| [scripts/download_model.sh](../../scripts/download_model.sh) | Command-line automation for download model; review prerequisites and side effects before executing. |
| [testdata/edge_cases/.gitignore](../../testdata/edge_cases/.gitignore) | Defines local data, credentials, dependencies, and generated outputs that Git should leave untracked. |
| [testdata/edge_cases/binary.bin](../../testdata/edge_cases/binary.bin) | Tracked binary resource in testdata/edge_cases; 24 bytes. |
| [testdata/edge_cases/empty.txt](../../testdata/edge_cases/empty.txt) | Reference notes for empty. |
| [testdata/edge_cases/malformed.py](../../testdata/edge_cases/malformed.py) | Tracked malformed resource in testdata/edge_cases; 12 bytes. |
| [testdata/sample_c/Makefile](../../testdata/sample_c/Makefile) | Tracked Makefile resource in testdata/sample_c; 70 bytes. |
| [testdata/sample_c/main.c](../../testdata/sample_c/main.c) | Source module for main; defines `main`. |
| [testdata/sample_c/utils.c](../../testdata/sample_c/utils.c) | Source module for utils; defines `Point`, `add`. |
| [testdata/sample_c/utils.h](../../testdata/sample_c/utils.h) | Source module for utils; defines `Point`. |
| [testdata/sample_cpp/CMakeLists.txt](../../testdata/sample_cpp/CMakeLists.txt) | CMake build definition for testdata/sample_cpp; declares targets, dependency discovery, and compiler/build settings. |
| [testdata/sample_cpp/main.cpp](../../testdata/sample_cpp/main.cpp) | Source module for main; defines `App`, `main`. |
| [testdata/sample_cpp/tree_node.cpp](../../testdata/sample_cpp/tree_node.cpp) | Source module for tree node; defines `sum`. |
| [testdata/sample_cpp/tree_node.hpp](../../testdata/sample_cpp/tree_node.hpp) | Source module for tree node; defines `TreeNode`. |
| [testdata/sample_empty/.gitkeep](../../testdata/sample_empty/.gitkeep) | Tracked .gitkeep resource in testdata/sample_empty; 1 bytes. |
| [testdata/sample_go/cmd/app/main.go](../../testdata/sample_go/cmd/app/main.go) | Source module for main; defines `main`. |
| [testdata/sample_go/go.mod](../../testdata/sample_go/go.mod) | Go module identity, toolchain version, and dependency requirements; paired with go.sum for downloaded module checksums. |
| [testdata/sample_node/package.json](../../testdata/sample_node/package.json) | Node package manifest for package; scripts: dev; declares package dependencies and metadata. |
| [testdata/sample_node/src/main.tsx](../../testdata/sample_node/src/main.tsx) | Source module for main; contributes to the application’s interface. |
| [testdata/sample_node/tsconfig.json](../../testdata/sample_node/tsconfig.json) | Structured tsconfig data; top-level fields: `compilerOptions`. |
| [testdata/sample_python/main.py](../../testdata/sample_python/main.py) | Python implementation of main. Defines `hello`. |
| [testdata/sample_python/node_modules/ignored.js](../../testdata/sample_python/node_modules/ignored.js) | Source module for ignored. |
| [testdata/sample_python/requirements.txt](../../testdata/sample_python/requirements.txt) | Python dependency requirements and optional dependency groups; install with the project’s documented interpreter and environment. |
| [testdata/sample_r/DESCRIPTION](../../testdata/sample_r/DESCRIPTION) | Tracked DESCRIPTION resource in testdata/sample_r; 102 bytes. |
| [testdata/sample_r/main.R](../../testdata/sample_r/main.R) | Tracked main resource in testdata/sample_r; 118 bytes. |
| [testdata/sample_ruby/Gemfile](../../testdata/sample_ruby/Gemfile) | Tracked Gemfile resource in testdata/sample_ruby; 41 bytes. |
| [testdata/sample_ruby/main.rb](../../testdata/sample_ruby/main.rb) | Tracked main resource in testdata/sample_ruby; 120 bytes. |
| [testdata/sample_rust/Cargo.toml](../../testdata/sample_rust/Cargo.toml) | Configuration or build metadata for Cargo; read alongside the application or tool that loads it. |
| [testdata/sample_rust/src/main.rs](../../testdata/sample_rust/src/main.rs) | Source module for main; defines `Point`, `add`, `main`. |
| [testdata/sample_sql/schema.sql](../../testdata/sample_sql/schema.sql) | Database definition or migration for schema; creates `users`, `orders`. |
| [testdata/sample_swift/Package.swift](../../testdata/sample_swift/Package.swift) | Tracked Package resource in testdata/sample_swift; 190 bytes. |
| [testdata/sample_swift/main.swift](../../testdata/sample_swift/main.swift) | Tracked main resource in testdata/sample_swift; 344 bytes. |
| [walkthrough.md](../../walkthrough.md) | Documentation: CodeLens Local v1.0.0 — Developer Walkthrough. Covers What v1.0.0 Includes; Release Artifacts; Verifying checksums. |
