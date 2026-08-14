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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "upgrade":
			os.Exit(runUpgrade(os.Args[2:]))
		case "-help", "--help", "help":
			os.Exit(help())
		case "-version", "--version", "version":
			os.Exit(printVersion())
		}
	}

	outputFile := flag.String("o", "", "output file (stdout if not provided)")
	format := flag.String("format", "json", "output format: json, csv, or pp (Portfolio Performance)")
	lang := flag.String("lang", "en", "language for -format pp headers/labels: en or de")
	includeSource := flag.Bool("include-source", false, "add source filename to each transaction")
	includeMetadata := flag.Bool("include-metadata", false, "wrap output with depot metadata (json format only)")
	quiet := flag.Bool("quiet", false, "hide skipped/problematic files; emit only valid JSON")
	verbose := flag.Bool("verbose", false, "print progress to stderr: how many files parsed")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		os.Exit(usage())
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
	transactions, metadata, errs, metaErr := processPDFs(pdfFiles, *includeSource)
	// Only fatal when the metadata was actually asked for. Without the flag a
	// mixed-depot directory is a fine thing to parse; with it, there is no
	// honest depot block to write, and writing none silently would be read as
	// "these documents carried no depot".
	if metaErr != nil && *includeMetadata {
		fmt.Fprintf(os.Stderr, "error: %v\n", metaErr)
		os.Exit(1)
	}
	if !*quiet {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "skipped %v\n", e)
		}
	}
	if len(transactions) == 0 {
		fmt.Fprintf(os.Stderr, "no transactions extracted from %d file(s)\n", len(pdfFiles))
		os.Exit(1)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "parsed %d transaction(s) from %d of %d file(s)\n",
			len(transactions), len(pdfFiles)-len(errs), len(pdfFiles))
	}

	if err := writeOutput(*format, *outputFile, *lang, transactions, metadata, *includeMetadata); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	// A partly failed batch still writes its output, because the documents that
	// did parse are worth having, but it must not report success. This runs
	// unattended, where the exit status is the only thing a scheduler looks at,
	// so exiting 0 here would turn a skipped statement into a silent data gap.
	// -quiet hides the per-file lines to keep stdout clean; it does not hide
	// the failure itself.
	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "parsed %d of %d file(s); %d skipped\n",
			len(pdfFiles)-len(errs), len(pdfFiles), len(errs))
		os.Exit(1)
	}

	os.Exit(0)
}

// writeOutput dispatches to the requested format. "pp" always writes two
// files (a Portfolio Transactions CSV and an Account Transactions CSV)
// because Portfolio Performance's CSV import handles those as two separate
// wizards; it therefore requires outFile so the two derived filenames are
// deterministic.
func writeOutput(format, outFile, lang string, transactions []*schema.Transaction, metadata *schema.DocumentMetadata, includeMetadata bool) error {
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
		if !export.ValidLang(lang) {
			return fmt.Errorf("unknown lang %q (want en or de)", lang)
		}
		base := strings.TrimSuffix(outFile, ".csv")
		if err := writeTo(base+"-portfolio.csv", func(w io.Writer) error {
			return export.WritePortfolioTransactions(w, transactions, lang)
		}); err != nil {
			return err
		}
		return writeTo(base+"-accounts.csv", func(w io.Writer) error {
			return export.WriteAccountTransactions(w, transactions, lang)
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
// bad file never aborts the batch. It returns the parsed transactions, the
// depot the batch belongs to, one error per failed file, and separately the
// error that arises when the batch belongs to more than one depot — a file
// that fails to parse is a per-file error, but a second depot is a property
// of the whole run and must not inflate the skipped-file count.
func processPDFs(pdfFiles []string, includeSource bool) ([]*schema.Transaction, *schema.DocumentMetadata, []error, error) {
	var transactions []*schema.Transaction
	var metadata *schema.DocumentMetadata
	var metaErr error
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

		// Capture metadata from the first file that has any, and refuse to let
		// a later file's depot be papered over by it. The output is one flat
		// transaction list with no per-transaction depot, so a single metadata
		// block is only truthful while every document in the batch shares it.
		// Taking the first depot and stamping it over the rest — as this did —
		// silently reattributes a second account's trades to the first holder.
		if doc.DepotNumber == "" && doc.DepotHolder == "" && doc.AccountNumber == "" {
			continue
		}
		md := &schema.DocumentMetadata{
			DepotNumber:   doc.DepotNumber,
			DepotHolder:   doc.DepotHolder,
			AccountNumber: doc.AccountNumber,
		}
		switch {
		case metadata == nil && metaErr == nil:
			metadata = md
		case metadata != nil && *metadata != *md:
			metaErr = fmt.Errorf(
				"batch spans more than one depot: %s belongs to depot %s (%s), earlier files to depot %s (%s); parse each depot separately to get metadata",
				pdfPath, md.DepotNumber, md.DepotHolder, metadata.DepotNumber, metadata.DepotHolder)
			metadata = nil
		}
	}

	return transactions, metadata, errs, metaErr
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

func printVersion() int {
	v := version
	if v == "" {
		v = "dev"
	}
	fmt.Printf("flatex-pdf-cli %s\n", v)
	return 0
}

func usage() int {
	fmt.Fprintf(os.Stderr, `usage:
  %[1]s [options] <file.pdf | directory>
  %[1]s upgrade [-check] [-y]
  %[1]s -version

Run '%[1]s -help' for a full description and examples.
`, os.Args[0])
	return 2
}

func help() int {
	fmt.Printf(`flatex-pdf-cli - extract structured JSON from German flatexDEGIRO broker PDFs

Parses trade confirmations, dividend/interest statements, crypto settlements,
and order documents into structured transactions. Works on a single PDF or
recursively over a directory of PDFs.

USAGE
  %[1]s [options] <file.pdf | directory>
  %[1]s upgrade [-check] [-y]
  %[1]s -version

OPTIONS
  -o FILE              output file (stdout if not provided; for -format pp, base name for the two output files)
  -format FORMAT       output format: json (default), csv, or pp (Portfolio Performance)
  -lang LANG           language for -format pp headers/labels: en (default) or de
  -include-source      add source filename to each transaction
  -include-metadata    wrap output with depot metadata (json format only)
  -quiet               hide skipped/problematic files; emit only valid JSON
  -version             show version and exit
  -help                show this help and exit

UPGRADE FLAGS
  -check               report whether a newer release exists, without installing it;
                       exit code: 0 up to date, 1 upgrade available, 2 on error
  -y                   skip the "Upgrade to vX.Y.Z? [y/N]" confirmation prompt

EXAMPLES
  # single PDF to stdout
  %[1]s path/to/statement.pdf

  # whole directory, save to file
  %[1]s -o output.json path/to/documents/

  # Portfolio Performance import CSVs
  %[1]s -format pp -o output.csv path/to/documents/

  # include depot metadata and source filenames with each transaction
  %[1]s -include-source -include-metadata path/to/documents/

  # check for / install updates
  %[1]s upgrade -check
  %[1]s upgrade
`, os.Args[0])
	return 0
}
