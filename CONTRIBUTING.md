# Contributing

Contributions are welcome — bug reports, real-world sample documents (PII
removed!), and code. For issues, feature requests, or questions, open an issue
on GitHub.

## Project Layout

CLI entry point in `main.go`; PDF text extraction in `internal/extractor`,
document detection and parsing in `internal/parser`, output types in
`internal/schema`. Agent skill in `skill/`, PII-free sample PDFs in `testdata/`.
Single dependency: [gxpdf](https://github.com/coregx/gxpdf) for PDF text
extraction.

## Running Tests

```bash
go test ./...                  # all tests
go test -v ./internal/parser   # one package, verbose
```

## Test Fixtures

The fixtures in `testdata/` are real flatex PDFs with the PII redacted and
replaced in place with synthetic values, so they behave exactly like production
documents. How they were made — and why naive synthetic PDFs don't work — is
covered in [Your AI's Test Fixtures Are Lying to You. Make real-world synthetic PDF files, PII safe!](https://pub.automatetherest.com/your-ais-test-fixtures-are-lying-to-you-0bc4f4ec7604).

![PII redaction workflow](docs/assets/pii-redaction-workflow.svg)

One known gap: the redaction re-inserts text out of reading order, so the ORDER
and CRYPTO fixtures only exercise *type detection*, not full field extraction —
those parsers are verified against real documents instead. A
reading-order-preserving redaction would let them parse end-to-end.

## Code Quality

The project uses `golangci-lint` for linting (config in `.golangci.yml`):

```bash
go fmt ./...
golangci-lint run
```

Optional pre-commit hooks: `pip install pre-commit && pre-commit install` —
runs `go fmt`, `go vet`, and `go test` on every commit (config in
`.pre-commit-config.yaml`).

## Before Opening a PR

1. All tests pass: `go test ./...`
2. Code is formatted: `go fmt ./...`
3. Linter passes: `golangci-lint run`
4. Commit messages follow conventional commits format
