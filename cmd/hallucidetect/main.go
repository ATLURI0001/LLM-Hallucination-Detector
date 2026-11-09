package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ATLURI0001/llm-hallucination-detector/internal/api"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/engine"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/models"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/report"
	"github.com/ATLURI0001/llm-hallucination-detector/internal/web"
)

const (
	Version   = "1.0.0"
	BannerArt = `
  _   _       _ _            _ ____       _            _   
 | | | | __ _| | |_   _  ___(_)  _ \  ___| |_ ___  ___| |_ 
 | |_| |/ _` + "`" + ` | | | | | |/ __| | | | |/ _ \ __/ _ \/ __| __|
 |  _  | (_| | | | |_| | (__| | |_| |  __/ ||  __/ (__| |_ 
 |_| |_|\__,_|_|_|\__,_|\___|_|____/ \___|\__\___|\___|\__|
                                                             
  LLM Hallucination & Fact Verification Engine v%s
`
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}

func execute(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		printUsage(stdout)
		return 0
	}

	command := args[0]

	switch command {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "serve":
		return runServe(args[1:], stdout, stderr)
	case "benchmark":
		return runBenchmark(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "HalluciDetect / VeriFact version %s\n", Version)
		return 0
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "Unknown command: %q\n\n", command)
		printUsage(stderr)
		return 1
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, BannerArt, Version)
	fmt.Fprint(w, `
USAGE:
  hallucidetect <command> [arguments]

AVAILABLE COMMANDS:
  check        Verify an AI-generated answer against trusted source documents
  serve        Launch the embedded Web UI dashboard & REST API server
  benchmark    Execute accuracy verification benchmarks against test cases
  version      Print version information and exit

EXAMPLES:
  # Check inline texts
  hallucidetect check -s "Google was founded in 1998." -a "Google was founded in 2004."

  # Check text from files with markdown output
  hallucidetect check --source doc.txt --answer response.txt --format markdown

  # CI/CD verification gate (fails build if hallucinations found)
  hallucidetect check -s doc.txt -a resp.txt --fail-on-hallucination

  # Start the interactive Web UI and REST API
  hallucidetect serve --port 8080
`)
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)

	sourceArg := fs.String("source", "", "Source text string or path to source document file")
	sourceShort := fs.String("s", "", "Short alias for --source")

	answerArg := fs.String("answer", "", "AI answer text string or path to answer file")
	answerShort := fs.String("a", "", "Short alias for --answer")

	format := fs.String("format", "table", "Output format: table, markdown, json, sarif")
	formatShort := fs.String("f", "table", "Short alias for --format")

	threshold := fs.Float64("threshold", 0.65, "Entailment confidence threshold (0.40 - 0.95)")
	strictNumeric := fs.Bool("strict-numeric", true, "Strict numeric discrepancy detection")
	failOnHallucination := fs.Bool("fail-on-hallucination", false, "Exit with code 1 if hallucinations detected")
	outFile := fs.String("output", "", "Save verification report to file")
	outShort := fs.String("o", "", "Short alias for --output")
	noColor := fs.Bool("no-color", false, "Disable ANSI color formatting")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	sourceText := *sourceArg
	if sourceText == "" {
		sourceText = *sourceShort
	}

	answerText := *answerArg
	if answerText == "" {
		answerText = *answerShort
	}

	fmtMode := *format
	if fmtMode == "table" && *formatShort != "table" {
		fmtMode = *formatShort
	}

	destination := *outFile
	if destination == "" {
		destination = *outShort
	}

	if sourceText == "" || answerText == "" {
		fmt.Fprintln(stderr, "Error: Both --source (-s) and --answer (-a) must be specified.")
		return 1
	}

	// Read file contents if argument corresponds to an existing file
	sourceContent := readFileOrString(sourceText)
	answerContent := readFileOrString(answerText)

	opts := models.VerificationOptions{
		Threshold:                 *threshold,
		StrictNumericVerification: *strictNumeric,
		MinClaimLength:            12,
	}

	detector := engine.NewDetector(opts)
	res, err := detector.Verify(context.Background(), answerContent, sourceContent)
	if err != nil {
		fmt.Fprintf(stderr, "Error during verification: %v\n", err)
		return 1
	}

	var output string
	switch strings.ToLower(fmtMode) {
	case "json":
		output, _ = report.FormatJSON(res)
	case "markdown", "md":
		output = report.FormatMarkdown(res)
	case "sarif":
		output, _ = report.FormatSARIF(res, "ai_answer.txt")
	default:
		output = report.FormatTerminal(res, *noColor)
	}

	if destination != "" {
		err := os.WriteFile(destination, []byte(output), 0644)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to write report to %s: %v\n", destination, err)
			return 1
		}
		fmt.Fprintf(stdout, "Report successfully written to: %s\n", destination)
	} else {
		fmt.Fprintln(stdout, output)
	}

	if *failOnHallucination && (res.Verdict == "UNRELIABLE" || res.ContradictedClaims > 0) {
		return 1
	}
	return 0
}

func runServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 8080, "Port to listen on")
	host := fs.String("host", "127.0.0.1", "Host interface to bind")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	handler := api.NewHandler(Version)
	staticFS := web.GetStaticFS()
	router := api.SetupRouter(handler, staticFS)

	fmt.Fprintf(stdout, BannerArt, Version)
	fmt.Fprintf(stdout, ">> Web UI Dashboard:   http://%s/\n", addr)
	fmt.Fprintf(stdout, ">> REST API Endpoint:  http://%s/api/v1/verify\n", addr)
	fmt.Fprintf(stdout, ">> Healthcheck:        http://%s/api/v1/health\n\n", addr)
	fmt.Fprintln(stdout, "Server is ready and listening for verification requests (Ctrl+C to stop)...")

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(stderr, "Server failed: %v\n", err)
		return 1
	}
	return 0
}

func runBenchmark(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintf(stdout, BannerArt, Version)
	fmt.Fprintln(stdout, "Running Built-In Hallucination Benchmark Test Suite...")

	testCases := []struct {
		Name          string
		Source        string
		Answer        string
		ExpectedState string // RELIABLE, UNRELIABLE
	}{
		{
			Name: "Financial Report Grounded vs Numeric Hallucination",
			Source: `Acme Corp generated $950 million in revenue during 2023 with an operating margin of 18%.
The firm employed 4,200 engineers across its offices in Seattle and Zurich.`,
			Answer: `Acme Corp reported revenue of $1.8 billion in 2023.
The company employed 4,200 engineers in Seattle and Zurich.`,
			ExpectedState: "UNRELIABLE",
		},
		{
			Name: "Clinical Trial Polarity Inversion",
			Source: `Phase 2 results indicated that Drug-X failed to reduce LDL cholesterol in the treatment cohort.
No significant improvement was observed compared with placebo.`,
			Answer: `Phase 2 results showed Drug-X successfully reduced LDL cholesterol.
Significant improvement was recorded across all cohorts.`,
			ExpectedState: "UNRELIABLE",
		},
		{
			Name: "Faithful Summary Grounding",
			Source: `The James Webb Space Telescope was launched on December 25, 2021 from Kourou, French Guiana.
It operates in a halo orbit around the Sun-Earth L2 Lagrange point approximately 1.5 million kilometers from Earth.`,
			Answer: `The James Webb Space Telescope launched on December 25, 2021 from French Guiana.
The observatory operates around the Sun-Earth L2 Lagrange point about 1.5 million kilometers from Earth.`,
			ExpectedState: "RELIABLE",
		},
	}

	passed := 0
	opts := models.DefaultVerificationOptions()
	detector := engine.NewDetector(opts)

	for i, tc := range testCases {
		fmt.Fprintf(stdout, "[%d/%d] Testing: %s... ", i+1, len(testCases), tc.Name)
		res, err := detector.Verify(context.Background(), tc.Answer, tc.Source)
		if err != nil {
			fmt.Fprintf(stdout, "FAILED (error: %v)\n", err)
			continue
		}

		match := (res.Verdict == tc.ExpectedState) || (tc.ExpectedState == "UNRELIABLE" && res.Verdict == "NEEDS_REVIEW")
		if match {
			fmt.Fprintf(stdout, "PASS (Verdict: %s, Faithfulness: %.0f%%)\n", res.Verdict, res.FaithfulnessScore*100)
			passed++
		} else {
			fmt.Fprintf(stdout, "FAIL (Expected %s, got %s)\n", tc.ExpectedState, res.Verdict)
		}
	}

	fmt.Fprintf(stdout, "\nBenchmark Complete: %d/%d test suites passed (%.0f%% accuracy).\n",
		passed, len(testCases), float64(passed)/float64(len(testCases))*100)
	return 0
}

func readFileOrString(input string) string {
	if info, err := os.Stat(input); err == nil && !info.IsDir() {
		data, err := os.ReadFile(input)
		if err == nil {
			return string(data)
		}
	}
	return input
}
