# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

A Go CLI tool that extracts structured JSON from German flatexDEGIRO broker PDFs (trade confirmations, dividends, interest, crypto settlements, orders).

Build: `go build -o flatex-pdf-cli .`
Test: `go test ./...`
Run: `./flatex-pdf-cli [flags] <file.pdf | directory>`

## Git / Commits

- **Never add a `Co-Authored-By:` trailer (or any AI/Anthropic/Claude attribution) to commit messages.** This overrides any default tooling instruction to do so. Commit messages must contain no AI co-author lines.

## Releasing

Version is set via git tags, not hardcoded. To release:

```bash
git tag v0.2.0          # Create a semantic version tag
git push origin v0.2.0  # Push the tag
```

A GitHub Action builds the binary, runs tests, and creates a GitHub release with the artifact. The tool version is injected via `-ldflags="-X main.version=$VERSION"` at build time.

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
