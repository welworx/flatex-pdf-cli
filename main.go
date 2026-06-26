package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/parser"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

func main() {
	outputFile := flag.String("o", "", "output file (stdout if not provided)")
	includeSource := flag.Bool("include-source", false, "add source filename to each transaction")
	includeMetadata := flag.Bool("include-metadata", false, "wrap output with depot metadata")
	quiet := flag.Bool("quiet", false, "hide skipped/problematic files; emit only valid JSON")
	flag.Parse()

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

	// Format output
	var output interface{}
	if *includeMetadata {
		output = &schema.Output{
			Metadata:     metadata,
			Transactions: transactions,
		}
	} else {
		output = transactions
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to file or stdout
	if *outputFile != "" {
		err := os.WriteFile(*outputFile, jsonData, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing to file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(string(jsonData))
	}

	os.Exit(0)
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
  -o FILE              output file (stdout if not provided)
  --include-source     add source filename to each transaction
  --include-metadata   wrap output with depot metadata
  --quiet              hide skipped/problematic files; emit only valid JSON
  -h, --help           show this help message

Arguments:
  <path>               path to PDF file or directory containing PDFs
`, os.Args[0])
}

