# Editor and Metadata Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Upgrade the current baseline implementations for autocomplete, syntax highlighting, Tibero/Cubrid metadata parity, and test coverage into stronger, more reliable production behavior.

**Architecture:** Keep the existing Bubble Tea/TUI structure and avoid a ground-up editor rewrite. Improve query-editor behavior in layers: first strengthen completion data and UI flow, then improve visible editor highlighting without forking the entire editor unless absolutely necessary, then harden metadata retrieval for ODBC drivers, and finally add regression tests around the highest-risk paths.

**Tech Stack:** Go, Bubble Tea, Bubbles textarea/textinput, Lip Gloss, database/sql, modernc SQLite, ODBC-tagged drivers, existing internal packages under `internal/`.

---

## Requirements Summary

- Autocomplete should move beyond table-name-only suggestions.
- Syntax highlighting should be more visible in the actual editor experience, not just a preview strip.
- Tibero/Cubrid metadata support should be more trustworthy and less “best effort”.
- Test coverage should grow around the newly added features and risky integration points.

## Acceptance Criteria

- Query autocomplete suggests table names and column names, including `table.column`-style completions where possible.
- Query editor visibly highlights SQL keywords in the editing surface or the effective rendered editor content, not only in a separate preview.
- Tibero and Cubrid metadata methods compile and pass existing tagged tests, with added parser/helper tests where applicable.
- `go test ./...` passes.
- `make test-odbc` passes.
- README/AGENTS/CLAUDE remain consistent with the final behavior.

## Risks and Mitigations

- `textarea` does not natively support true token-span styling.
  Mitigation: first improve rendered editor highlighting by post-processing the visible editor content; only fork `textarea` if still insufficient.
- Loading all columns for autocomplete can be slow on large schemas.
  Mitigation: add lazy-loading or capped caching, starting with table list + on-demand column expansion.
- ODBC metadata queries may differ across environments.
  Mitigation: isolate parser/helpers, keep compile/tag tests green, and avoid overfitting to one runtime shape.
- TUI behavior regressions are easy to introduce.
  Mitigation: add helper/unit tests around completion parsing, SQL highlight rendering, and screen-level logic where possible.

## Verification Steps

- Run: `go test ./...`
- Run: `make test-odbc`
- Run: `go test ./internal/tui`
- Run: `go test ./internal/db`

### Task 1: Autocomplete Upgrade

**Files:**
- Modify: `internal/tui/query_completion.go`
- Modify: `internal/tui/query_completion_test.go`
- Modify: `internal/tui/screen_query.go`
- Modify: `internal/tui/keys.go`
- Modify: `internal/tui/screen_help.go`

**Step 1: Write the failing test**

Add tests for:
- column-aware completion prefix parsing
- `table.column` suggestion filtering
- empty-prefix behavior with cached suggestions

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui`
Expected: FAIL in `query_completion_test.go` for missing or incomplete column-aware behavior

**Step 3: Write minimal implementation**

Implement:
- richer completion item model or structured suggestion handling
- table-name plus column-name suggestion sources
- lazy or capped metadata loading in `screen_query.go`
- insertion logic that supports suffix completion for `table.column`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/query_completion.go internal/tui/query_completion_test.go internal/tui/screen_query.go internal/tui/keys.go internal/tui/screen_help.go
git commit -m "feat: improve query autocomplete"
```

### Task 2: Syntax Highlighting Upgrade

**Files:**
- Modify: `internal/tui/sql_highlight.go`
- Create: `internal/tui/sql_highlight_render_test.go`
- Modify: `internal/tui/screen_query.go`
- Modify: `internal/tui/theme.go`

**Step 1: Write the failing test**

Add tests for:
- keyword highlighting in rendered editor text
- preservation of non-keyword text
- multi-line preview/render behavior

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui`
Expected: FAIL in SQL highlight rendering tests

**Step 3: Write minimal implementation**

Implement:
- stronger SQL keyword styling helpers
- rendered-editor highlighting pass that affects the visible editor output
- theme styles for keyword emphasis without breaking cursor visibility

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/sql_highlight.go internal/tui/sql_highlight_render_test.go internal/tui/screen_query.go internal/tui/theme.go
git commit -m "feat: improve query syntax highlighting"
```

### Task 3: Tibero/Cubrid Metadata Hardening

**Files:**
- Modify: `internal/db/tibero.go`
- Modify: `internal/db/cubrid.go`
- Modify: `internal/db/cubrid_test.go`
- Create: `internal/db/tibero_metadata_test.go`

**Step 1: Write the failing test**

Add tests for:
- Cubrid FK parser edge cases
- Tibero helper/query-shaping logic where practical
- metadata helper correctness for ordering and grouping

**Step 2: Run test to verify it fails**

Run: `make test-odbc`
Expected: FAIL in tagged metadata helper/parser tests

**Step 3: Write minimal implementation**

Implement:
- stronger Cubrid FK parsing and index/PK grouping
- Tibero metadata query/result shaping consistency
- helper extraction where repeated grouping/order logic is present

**Step 4: Run test to verify it passes**

Run: `make test-odbc`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/db/tibero.go internal/db/cubrid.go internal/db/cubrid_test.go internal/db/tibero_metadata_test.go
git commit -m "feat: harden odbc metadata support"
```

### Task 4: Test Coverage Expansion

**Files:**
- Create: `internal/tui/screen_query_flow_test.go`
- Modify: `internal/history/history_test.go`
- Modify: `internal/db/driver_test.go`
- Create: `internal/sshtunnel/tunnel_test.go`

**Step 1: Write the failing test**

Add tests for:
- query history/export/autocomplete screen helpers
- pooling helper edge cases
- SSH tunnel auth method validation/path selection

**Step 2: Run test to verify it fails**

Run: `go test ./...`
Expected: FAIL in new helper/flow tests

**Step 3: Write minimal implementation**

Implement only the helper adjustments needed to make tests pass.

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/screen_query_flow_test.go internal/history/history_test.go internal/db/driver_test.go internal/sshtunnel/tunnel_test.go
git commit -m "test: expand coverage for query and tunnel flows"
```

### Task 5: Documentation Sync

**Files:**
- Modify: `README.md`
- Modify: `README.en.md`
- Modify: `AGENTS.md`
- Modify: `CLAUDE.md`

**Step 1: Write the failing test**

Manual doc audit checklist:
- feature list matches code
- no feature is documented as “planned” if implemented
- remaining gaps are accurately described

**Step 2: Run test to verify it fails**

Run: manual diff review against implemented behavior
Expected: mismatches found before doc update

**Step 3: Write minimal implementation**

Update all project docs to reflect the hardened autocomplete/highlighting/metadata/test story.

**Step 4: Run test to verify it passes**

Run: manual doc audit again
Expected: all key behaviors match the codebase

**Step 5: Commit**

```bash
git add README.md README.en.md AGENTS.md CLAUDE.md
git commit -m "docs: sync hardened editor and metadata behavior"
```
