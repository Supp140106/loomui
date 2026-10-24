# Loom

Loom is a declarative TUI framework in Go: FTXUI-style declarative API
(HBox/VBox/Text/Border/Flex), the feature set of Lip Gloss (styling) and
Bubbles (widgets), on a Bubble Tea-like runtime.

## Architecture

Dependency flows downward only. No import cycles, ever:

```
style -> cell -> layout -> element -> term -> component -> widget -> tui (re-exports)
```

- `style` — immutable, chainable styling (colors, attributes, margins, padding).
- `cell` — a single terminal cell (rune + style).
- `layout` — size negotiation, HBox/VBox, alignment, flex.
- `element` — pure, stateless renderables (Text, Border, Box, etc.).
- `term` — terminal I/O, the headless backend, raw mode.
- `component` — stateful components, event loop/update model (Bubble Tea-like).
- `widget` — concrete widgets (Bubbles-like: textinput, spinner, etc.).
- `tui` — root package. Re-exports the public API. All user code imports only `tui`.

## Rules

1. **Go 1.22+.** Keep dependencies minimal: `golang.org/x/term`,
   `github.com/rivo/uniseg`, `github.com/mattn/go-runewidth`. Nothing else
   without asking.
2. **Unicode width uses grapheme clusters**, never `len()` and never raw
   runes. Measure via `uniseg`/`runewidth` on cluster boundaries.
3. **Elements are pure and stateless.** All state lives in components.
   Rendering an element twice with the same input must produce identical output.
4. **Styles are immutable and chainable.** Methods return copies, never
   mutate the receiver.
5. **Every package needs tests.**
6. **Every element and widget needs a golden-file test** using the headless
   backend. Golden files live next to the package under `testdata/`; regenerate
   deliberately, never blindly.
7. **Public API goes through the root package `tui`** and stays small and
   idiomatic. Package internals may be exported only if `tui` needs them.
8. **No new packages or features** beyond what was asked for.

## Checks (run after every task)

```sh
go vet ./...
go test ./...
gofmt -l .   # output must be empty
```

## Conventions

- Package names are singular and lowercase; no `utils`, no `helpers`.
- No comments unless asked; no comments explaining the obvious.
- Golden files are reviewed as part of the diff. A changed golden file with no
  matching intentional behavior change is a bug.
- Do not commit changes unless explicitly requested.
