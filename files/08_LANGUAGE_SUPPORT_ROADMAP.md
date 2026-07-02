# CodeLens Local — Language Support Expansion Roadmap

This document outlines the blueprints, conventions, and rules for extending CodeLens Local to support additional target languages. 

Each language is structured under three main categories:
1. **Manifest files** (for stack detection and setup command guessing).
2. **Entry point conventions** (for locating starting directories and files).
3. **Symbol and import extraction** (lightweight regex-based rules matching the KISS constraints).

---

## 🐍 Python (Existing + Extensions)

- **Manifests:** `requirements.txt`, `pyproject.toml`, `setup.py`, `Pipfile`
  - *Setup guess:* `pip install -r requirements.txt`, `poetry install`, or `pipenv install`
- **Entry points:** `main.py`, `app.py`, `run.py`, `src/main.py`
- **Symbol Extraction:**
  - **Imports:** `^(?:from\s+([A-Za-z0-9_./]+)\s+import\s+(.+)|import\s+(.+))`
  - **Functions:** `^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\s*` -> kind `function` (or `method` if indented under a class)
  - **Classes:** `^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)` -> kind `class`

---

## 🇨 C

- **Manifests:** `Makefile`, `configure.ac`, `Makefile.am`
  - *Setup guess:* `make` or `./configure && make`
- **Entry points:** `main.c`, `src/main.c`
- **Symbol Extraction:**
  - **Imports:** `^\s*#include\s+["<]([^">]+)[">]` -> kind `import`
  - **Functions:** `^\s*(?:[A-Za-z0-9_*\s]+)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*(?:{|;)?` -> kind `function`
  - **Structs/Unions:** `^\s*(typedef\s+)?(struct|union|enum)\s+([A-Za-z_][A-Za-z0-9_]*)` -> kind `type`

---

## 🦀 Rust

- **Manifests:** `Cargo.toml`
  - *Setup guess:* `cargo build` and `cargo run`
- **Entry points:** `src/main.rs`, `src/lib.rs`
- **Symbol Extraction:**
  - **Imports:** `^\s*use\s+([A-Za-z0-9_::*{}]+);` -> kind `import`
  - **Functions:** `^\s*(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)` -> kind `function` (or `method` if inside an `impl` block)
  - **Types:** `^\s*(?:pub\s+)?(struct|enum|union|trait)\s+([A-Za-z_][A-Za-z0-9_]*)` -> kind `type`

---

## 🐹 Go (Existing)

- **Manifests:** `go.mod`
  - *Setup guess:* `go build ./...` or `go run ./...`
- **Entry points:** `main.go`, `cmd/*/main.go`
- **Symbol Extraction (AST-based):**
  - **Imports:** Standard Go AST import specs -> kind `import`
  - **Functions:** Func decls -> kind `function` (or `method` if has receivers)
  - **Types:** Struct/interface specs -> kind `type`

---

## 💎 Ruby

- **Manifests:** `Gemfile`, `Rakefile`
  - *Setup guess:* `bundle install` and `bundle exec ruby app.rb`
- **Entry points:** `main.rb`, `app.rb`, `config/environment.rb`
- **Symbol Extraction:**
  - **Imports:** `^\s*(?:require|require_relative|load)\s+["']([^"']+)["']` -> kind `import`
  - **Methods:** `^\s*def\s+([A-Za-z0-9_!?.]+)\s*` -> kind `method`
  - **Classes/Modules:** `^\s*(class|module)\s+([A-Za-z0-9_:]+)` -> kind `class`

---

## 📊 R

- **Manifests:** `DESCRIPTION`, `NAMESPACE`
  - *Setup guess:* `Rscript -e "install.packages(c(...))"`
- **Entry points:** `main.R`, `run.R`
- **Symbol Extraction:**
  - **Imports:** `^\s*(?:library|require)\s*\(([^)]+)\)` -> kind `import`
  - **Functions:** `^\s*([A-Za-z0-9_.]+)\s*(?:<-|=)\s*function\s*\(([^)]*)\)` -> kind `function`

---

## 🦅 Swift

- **Manifests:** `Package.swift`
  - *Setup guess:* `swift build` and `swift run`
- **Entry points:** `main.swift`, `App.swift`
- **Symbol Extraction:**
  - **Imports:** `^\s*import\s+([A-Za-z0-9_]+)` -> kind `import`
  - **Functions:** `^\s*(?:public|private|internal|fileprivate)?\s*(?:static|class)?\s*func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)` -> kind `function` (or `method` if inside struct/class)
  - **Types:** `^\s*(class|struct|enum|protocol|struct)\s+([A-Za-z_][A-Za-z0-9_]*)` -> kind `type`

---

## 🗄️ SQL

- **Manifests:** `schema.sql`, `migrations/`, `init.sql`
  - *Setup guess:* `sqlite3 db.sqlite < schema.sql` or similar DB tool commands
- **Entry points:** `schema.sql`, `init.sql`
- **Symbol Extraction:**
  - **Tables:** `(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z0-9_."`]+)` -> kind `table`
  - **Views:** `(?i)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?VIEW\s+([A-Za-z0-9_."`]+)` -> kind `view`
  - **Procedures:** `(?i)^\s*CREATE\s+PROCEDURE\s+([A-Za-z0-9_."`]+)` -> kind `procedure`
