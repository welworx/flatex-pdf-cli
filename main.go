package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/welworx/flatex-pdf-cli/internal/export"
	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/parser"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

var version = ""

func main() {
	outputFile := flag.String("o", "", "output file (stdout if not provided)")
	format := flag.String("format", "json", "output format: json, csv, or pp (Portfolio Performance)")
	includeSource := flag.Bool("include-source", false, "add source filename to each transaction")
	includeMetadata := flag.Bool("include-metadata", false, "wrap output with depot metadata (json format only)")
	quiet := flag.Bool("quiet", false, "hide skipped/problematic files; emit only valid JSON")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		v := version
		if v == "" {
			v = "dev"
		}
		fmt.Printf("flatex-pdf-cli version %s\n", v)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	path := args[0]

	// Discover all PDFs
	pdfFiles, err := discoverPDFs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error discovering PDFs: %v\n", err)
		os.Exit(1)
	}

	if len(pdfFiles) == 0 {
		fmt.Fprintf(os.Stderr, "no PDF files found in %s\n", path)
		os.Exit(1)
	}

	// Process PDFs; a file that fails to extract or parse is reported and
	// skipped so the rest of the batch still produces output.
	transactions, metadata, errs := processPDFs(pdfFiles, *includeSource)
	if !*quiet {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "skipped %v\n", e)
		}
	}
	if len(transactions) == 0 {
		fmt.Fprintf(os.Stderr, "no transactions extracted from %d file(s)\n", len(pdfFiles))
		os.Exit(1)
	}

	if err := writeOutput(*format, *outputFile, transactions, metadata, *includeMetadata); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

// writeOutput dispatches to the requested format. "pp" always writes two
// files (a Portfolio Transactions CSV and an Account Transactions CSV)
// because Portfolio Performance's CSV import handles those as two separate
// wizards; it therefore requires outFile so the two derived filenames are
// deterministic.
func writeOutput(format, outFile string, transactions []*schema.Transaction, metadata *schema.DocumentMetadata, includeMetadata bool) error {
	switch format {
	case "json":
		return writeTo(outFile, func(w io.Writer) error {
			var output interface{}
			if includeMetadata {
				output = &schema.Output{Metadata: metadata, Transactions: transactions}
			} else {
				output = transactions
			}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			enc.SetEscapeHTML(false)
			return enc.Encode(output)
		})
	case "csv":
		return writeTo(outFile, func(w io.Writer) error {
			return export.WriteCSV(w, transactions)
		})
	case "pp":
		if outFile == "" {
			return fmt.Errorf("-format pp requires -o (writes <base>-portfolio.csv and <base>-accounts.csv)")
		}
		base := strings.TrimSuffix(outFile, ".csv")
		if err := writeTo(base+"-portfolio.csv", func(w io.Writer) error {
			return export.WritePortfolioTransactions(w, transactions)
		}); err != nil {
			return err
		}
		return writeTo(base+"-accounts.csv", func(w io.Writer) error {
			return export.WriteAccountTransactions(w, transactions)
		})
	default:
		return fmt.Errorf("unknown format %q (want json, csv, or pp)", format)
	}
}

// writeTo runs fn against stdout, or against a newly created file at path
// when path is non-empty.
func writeTo(path string, fn func(io.Writer) error) error {
	if path == "" {
		return fn(os.Stdout)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

// processPDFs parses each PDF, skipping (and reporting) any that fail so one
// bad file never aborts the batch. Returns the parsed transactions, the first
// file's metadata, and one error per failed file.
func processPDFs(pdfFiles []string, includeSource bool) ([]*schema.Transaction, *schema.DocumentMetadata, []error) {
	var transactions []*schema.Transaction
	var metadata *schema.DocumentMetadata
	var errs []error

	for _, pdfPath := range pdfFiles {
		doc, err := extractor.ExtractPDF(pdfPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", pdfPath, err))
			continue
		}

		txns, err := parser.Parse(doc)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", pdfPath, err))
			continue
		}

		for _, txn := range txns {
			if includeSource {
				txn.Source = doc.Filename
			}
			transactions = append(transactions, txn)
		}

		// Capture metadata from the first file that has any.
		if metadata == nil && (doc.DepotNumber != "" || doc.DepotHolder != "" || doc.AccountNumber != "") {
			metadata = &schema.DocumentMetadata{
				DepotNumber:   doc.DepotNumber,
				DepotHolder:   doc.DepotHolder,
				AccountNumber: doc.AccountNumber,
			}
		}
	}

	return transactions, metadata, errs
}

// discoverPDFs finds all PDF files recursively in the given path.
func discoverPDFs(path string) ([]string, error) {
	var pdfFiles []string

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(filePath) == ".pdf" {
			pdfFiles = append(pdfFiles, filePath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort for deterministic output
	sort.Strings(pdfFiles)

	return pdfFiles, nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: %s [options] <path>

Process PDF files from flatex and extract transaction data.

Options:
  -o FILE              output file (stdout if not provided; for -format pp, base name for the two output files)
  -format FORMAT       output format: json (default), csv, or pp (Portfolio Performance)
  -include-source      add source filename to each transaction
  -include-metadata    wrap output with depot metadata (json format only)
  -quiet               hide skipped/problematic files; emit only valid JSON
  -version             show version and exit

Arguments:
  <path>               path to PDF file or directory containing PDFs
`, os.Args[0])
}
