# CodeLens Local — Sequential AI Agent Prompts

This document contains copy-pasteable task prompts for extending CodeLens Local. These prompts are organized in the exact order of implementation, from simple polish to language extensions.

---

## Prompt 1 — SHA-256 Release Checksums

```text
TASK: Add SHA-256 Release Checksums
Milestone: Post-M5 Polish
Implements: NFR-M2 (Release verification)
In scope:
- Update the Makefile release target to generate SHA-256 checksums for all compiled binaries.
- Write the checksums to dist/checksums.txt.
- Ensure the shell loop handles cross-platform checksum commands (e.g., fallback between sha256sum and shasum).

Out of scope: Writing complex Go verification code, signing binaries.

Read first:
- Makefile
- files/04_AGENT_GUIDE.md
- files/06_SECURITY_MODEL.md

Build:
1. Open the Makefile and locate the `release` target.
2. After the platform loop completes compiling all binaries into `dist/`, add a step to navigate to `dist/` and compute checksums.
3. Use a portable command sequence like:
   `cd dist && (sha256sum * > checksums.txt 2>/dev/null || shasum -a 256 * > checksums.txt 2>/dev/null || true)`
4. Ensure the clean target removes `dist` as it already does.

Tests:
- Run `make release VERSION=vtest` and assert that `dist/checksums.txt` is created containing checksums of all generated binaries.
- Run `make clean` and verify `dist` is deleted.

Done when:
`make lint`, `make test`, and `make release` compile and run without errors.
```

---

## Prompt 2 — pyproject.toml & Python Tooling Guesses

```text
TASK: Add pyproject.toml stack detection and guesses
Milestone: Post-M5 Polish
Implements: FR-D (Stack Detection), FR-C (CLI Commands)
In scope:
- Recognize pyproject.toml as Python language/tool evidence.
- Suggest modern Python setup commands based on lockfiles (poetry.lock, Pipfile).

Out of scope: Writing custom setup.py parsers, running python setup commands.

Read first:
- internal/stackdetect/stackdetect.go
- internal/cli/cli.go

Build:
1. In `stackdetect.go`, ensure `pyproject.toml` is detected. Add:
   - "Python" (language) using "pyproject.toml"
   - "pyproject" (tool) using "pyproject.toml"
2. In `cli.go`, locate `setupCommands`. Add a case for `"pyproject.toml"`.
3. Inside the case, look for companion lockfiles in the scanner stack report:
   - If `poetry.lock` is present: add `poetry install`.
   - If `Pipfile` is present: add `pipenv install`.
   - Otherwise, suggest `pip install .` as a generic Python package installer.

Tests:
- Create a temporary Python fixture containing `pyproject.toml` and `poetry.lock`.
- Assert that `Detect` reports `Python` and `pyproject`.
- Assert that `setupCommands` returns `poetry install`.

Done when:
`make lint test build` is fully green.
```

---

## Prompt 3 — C Language Support

```text
TASK: Add C Language Support
Milestone: Language Expansion
Implements: FR-S (C Scanner), FR-D (Make Stack), FR-T (C Entry Points), FR-Y (C Symbol Extraction)
In scope:
- Recognize .c extensions as "C" language.
- Detect Makefiles or configure.ac files as stack evidence.
- Recognize main.c and src/main.c as entry points.
- Extract #include imports, struct/union types, and function definitions/prototypes.

Out of scope: Complex macro expansions, linking CGo, running Makefile configurations.

Read first:
- files/08_LANGUAGE_SUPPORT_ROADMAP.md
- internal/scanner/scanner.go
- internal/stackdetect/stackdetect.go
- internal/structure/structure.go
- internal/symbols/symbols.go

Build:
1. **Scanner:** In `scanner.go:detectLanguage`, map `.c` -> `"C"`.
2. **Stack:** In `stackdetect.go`, check for `Makefile` or `configure.ac`. If found, add `"C"` (language) and `"Make"` (tool). Suggest `make` as a setup command in `cli.go`.
3. **Structure:** In `structure.go:entryReason`, check for `main.c` or `src/main.c` -> `"C main entry point convention"`.
4. **Symbols:** In `symbols.go`, implement `extractC(path string, data []byte) []Symbol`:
   - Imports: `^\s*#include\s+["<]([^">]+)[">]`
   - Structs/Unions: `^\s*(typedef\s+)?(struct|union|enum)\s+([A-Za-z_][A-Za-z0-9_]*)`
   - Functions: `^\s*(?:[A-Za-z0-9_*\s]+)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*(?:{|;)?` (exclude if name is: if, while, for, switch, return).

Tests:
- Create `testdata/sample_c` containing `Makefile`, `main.c`, and a header.
- Write unit tests verifying C language detection, stack detection, and symbol extraction of imports and functions.

Done when:
`make lint test build` passes all unit and golden tests.
```

---

## Prompt 4 — Rust Language Support

```text
TASK: Add Rust Language Support
Milestone: Language Expansion
Implements: FR-S (Rust Scanner), FR-D (Cargo Stack), FR-T (Rust Entry Points), FR-Y (Rust Symbol Extraction)
In scope:
- Recognize .rs extensions as "Rust" language.
- Detect Cargo.toml stack evidence.
- Recognize src/main.rs and src/lib.rs as entry points.
- Extract use imports, fn functions/methods, and struct/enum/trait types.

Out of scope: Parsing macro invocations, parsing complex lifetime/generic trees.

Read first:
- files/08_LANGUAGE_SUPPORT_ROADMAP.md
- internal/scanner/scanner.go
- internal/stackdetect/stackdetect.go
- internal/structure/structure.go
- internal/symbols/symbols.go

Build:
1. **Scanner:** In `scanner.go:detectLanguage`, map `.rs` -> `"Rust"`.
2. **Stack:** In `stackdetect.go`, check for `Cargo.toml`. If found, add `"Rust"` (language) and `"Cargo"` (tool). Suggest `cargo build` and `cargo run` in `cli.go`.
3. **Structure:** In `structure.go:entryReason`, check for `src/main.rs` or `src/lib.rs` -> `"Rust main/library entry point convention"`.
4. **Symbols:** In `symbols.go`, implement `extractRust(path string, data []byte) []Symbol`:
   - Imports: `^\s*use\s+([A-Za-z0-9_::*{}]+);`
   - Functions: `^\s*(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`
   - Types: `^\s*(?:pub\s+)?(struct|enum|union|trait)\s+([A-Za-z_][A-Za-z0-9_]*)`

Tests:
- Create `testdata/sample_rust` with `Cargo.toml` and `src/main.rs`.
- Assert that "Rust" stack, entry point, functions, and structs are correctly identified.

Done when:
`make lint test build` passes all tests.
```

---

## Prompt 5 — Ruby Language Support

```text
TASK: Add Ruby Language Support
Milestone: Language Expansion
Implements: FR-S (Ruby Scanner), FR-D (Bundler Stack), FR-T (Ruby Entry Points), FR-Y (Ruby Symbol Extraction)
In scope:
- Recognize .rb extensions as "Ruby" language.
- Detect Gemfile/Rakefile stack evidence.
- Recognize main.rb and app.rb as entry points.
- Extract require/load imports, def methods, and class/module definitions.

Out of scope: Parsing dynamic class evaluations, executing bundler scripts.

Read first:
- files/08_LANGUAGE_SUPPORT_ROADMAP.md
- internal/scanner/scanner.go
- internal/stackdetect/stackdetect.go
- internal/structure/structure.go
- internal/symbols/symbols.go

Build:
1. **Scanner:** In `scanner.go:detectLanguage`, map `.rb` -> `"Ruby"`.
2. **Stack:** In `stackdetect.go`, check for `Gemfile`. If found, add `"Ruby"` (language) and `"Bundler"` (tool). Suggest `bundle install` and `bundle exec ruby app.rb` in `cli.go`.
3. **Structure:** In `structure.go:entryReason`, check for `main.rb` or `app.rb` -> `"Ruby entry point convention"`.
4. **Symbols:** In `symbols.go`, implement `extractRuby(path string, data []byte) []Symbol`:
   - Imports: `^\s*(?:require|require_relative|load)\s+["']([^"']+)["']`
   - Methods: `^\s*def\s+([A-Za-z0-9_!?.]+)\s*`
   - Classes/Modules: `^\s*(class|module)\s+([A-Za-z0-9_:]+)`

Tests:
- Create `testdata/sample_ruby` with `Gemfile` and `main.rb`.
- Assert that Ruby stack, entry point, classes, and methods are correctly extracted.

Done when:
`make lint test build` passes.
```

---

## Prompt 6 — R Language Support

```text
TASK: Add R Language Support
Milestone: Language Expansion
Implements: FR-S (R Scanner), FR-D (R Stack), FR-T (R Entry Points), FR-Y (R Symbol Extraction)
In scope:
- Recognize .R and .r extensions as "R" language.
- Detect DESCRIPTION files as stack evidence.
- Recognize main.R and run.R as entry points.
- Extract library/require imports, and assignments to function(...) closures.

Out of scope: Running R scripts.

Read first:
- files/08_LANGUAGE_SUPPORT_ROADMAP.md
- internal/scanner/scanner.go
- internal/stackdetect/stackdetect.go
- internal/structure/structure.go
- internal/symbols/symbols.go

Build:
1. **Scanner:** In `scanner.go:detectLanguage`, map `.R` and `.r` -> `"R"`.
2. **Stack:** In `stackdetect.go`, check for `DESCRIPTION`. If found, add `"R"` (language) and `"Rscript"` (tool). Suggest running packages installation using Rscript in `cli.go`.
3. **Structure:** In `structure.go:entryReason`, check for `main.R` or `run.R` -> `"R script entry point convention"`.
4. **Symbols:** In `symbols.go`, implement `extractR(path string, data []byte) []Symbol`:
   - Imports: `^\s*(?:library|require)\s*\(([^)]+)\)`
   - Functions: `^\s*([A-Za-z0-9_.]+)\s*(?:<-|=)\s*function\s*\(([^)]*)\)`

Tests:
- Create `testdata/sample_r` with `DESCRIPTION` and `main.R`.
- Verify R stack, entry point, libraries, and function declarations.

Done when:
`make lint test build` passes.
```

---

## Prompt 7 — Swift Language Support

```text
TASK: Add Swift Language Support
Milestone: Language Expansion
Implements: FR-S (Swift Scanner), FR-D (SPM Stack), FR-T (Swift Entry Points), FR-Y (Swift Symbol Extraction)
In scope:
- Recognize .swift extensions as "Swift" language.
- Detect Package.swift stack evidence.
- Recognize main.swift and App.swift as entry points.
- Extract import statements, func functions, and class/struct/enum/protocol definitions.

Out of scope: Running swift compiler tools.

Read first:
- files/08_LANGUAGE_SUPPORT_ROADMAP.md
- internal/scanner/scanner.go
- internal/stackdetect/stackdetect.go
- internal/structure/structure.go
- internal/symbols/symbols.go

Build:
1. **Scanner:** In `scanner.go:detectLanguage`, map `.swift` -> `"Swift"`.
2. **Stack:** In `stackdetect.go`, check for `Package.swift`. If found, add `"Swift"` (language) and `"Swift Package Manager"` (tool). Suggest `swift build` and `swift run` in `cli.go`.
3. **Structure:** In `structure.go:entryReason`, check for `main.swift` or `App.swift` -> `"Swift entry point convention"`.
4. **Symbols:** In `symbols.go`, implement `extractSwift(path string, data []byte) []Symbol`:
   - Imports: `^\s*import\s+([A-Za-z0-9_]+)`
   - Functions: `^\s*(?:public|private|internal|fileprivate)?\s*(?:static|class)?\s*func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`
   - Types: `^\s*(class|struct|enum|protocol)\s+([A-Za-z_][A-Za-z0-9_]*)`

Tests:
- Create `testdata/sample_swift` with `Package.swift` and `main.swift`.
- Assert Swift stack, entry point, functions, and structs are correctly identified.

Done when:
`make lint test build` passes.
```

---

## Prompt 8 — SQL Language Support

```text
TASK: Add SQL Language Support
Milestone: Language Expansion
Implements: FR-S (SQL Scanner), FR-D (Database Stack), FR-T (SQL Entry Points), FR-Y (SQL Symbol Extraction)
In scope:
- Recognize .sql extensions as "SQL" language.
- Detect schema.sql or migrations/ directory as SQL evidence.
- Recognize schema.sql and init.sql as entry points.
- Extract CREATE TABLE, CREATE VIEW, and CREATE PROCEDURE definitions.

Out of scope: Executing SQL databases, parsing complex SQL dialect asts.

Read first:
- files/08_LANGUAGE_SUPPORT_ROADMAP.md
- internal/scanner/scanner.go
- internal/stackdetect/stackdetect.go
- internal/structure/structure.go
- internal/symbols/symbols.go

Build:
1. **Scanner:** In `scanner.go:detectLanguage`, map `.sql` -> `"SQL"`.
2. **Stack:** In `stackdetect.go`, check for `schema.sql` or `migrations/`. If found, add `"SQL"` (database).
3. **Structure:** In `structure.go:entryReason`, check for `schema.sql` or `init.sql` -> `"SQL initialization entry point convention"`.
4. **Symbols:** In `symbols.go`, implement `extractSQL(path string, data []byte) []Symbol`:
   - Tables: `(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z0-9_."`]+)`
   - Views: `(?i)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?VIEW\s+([A-Za-z0-9_."`]+)`
   - Procedures: `(?i)^\s*CREATE\s+PROCEDURE\s+([A-Za-z0-9_."`]+)`

Tests:
- Create `testdata/sample_sql` with `schema.sql`.
- Assert SQL database, entry point, and tables/views/procedures are correctly found.

Done when:
`make lint test build` passes.
```
