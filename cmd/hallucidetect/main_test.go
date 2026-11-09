package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecute_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execute([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), Version) {
		t.Fatalf("expected version %s in output, got %s", Version, stdout.String())
	}
}

func TestExecute_HelpAndUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"empty args", []string{}},
		{"help flag", []string{"help"}},
		{"-h flag", []string{"-h"}},
		{"--help flag", []string{"--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := execute(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("expected exit code 0, got %d", code)
			}
			if !strings.Contains(stdout.String(), "USAGE:") {
				t.Fatalf("expected USAGE in output, got: %s", stdout.String())
			}
		})
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execute([]string{"nonexistent_command"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Unknown command") {
		t.Fatalf("expected 'Unknown command' error, got %s", stderr.String())
	}
}

func TestExecute_CheckMissingArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"missing both", []string{"check"}},
		{"missing answer", []string{"check", "-s", "some source"}},
		{"missing source", []string{"check", "-a", "some answer"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := execute(tt.args, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("expected exit code 1, got %d", code)
			}
			if !strings.Contains(stderr.String(), "must be specified") {
				t.Fatalf("expected 'must be specified' error, got %s", stderr.String())
			}
		})
	}
}

func TestExecute_CheckInlineJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{
		"check",
		"-s", "The company was established in 1998 in California.",
		"-a", "The company was established in 1998 in California.",
		"-f", "json",
	}

	code := execute(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"verdict"`) || !strings.Contains(out, "RELIABLE") {
		t.Fatalf("expected JSON output with RELIABLE verdict, got: %s", out)
	}
}

func TestExecute_CheckFailOnHallucination(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{
		"check",
		"-s", "Company revenues totaled $10 million in 2023.",
		"-a", "Company revenues totaled $999 million in 2023.",
		"--fail-on-hallucination",
	}

	code := execute(args, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 due to --fail-on-hallucination, got %d", code)
	}
}

func TestExecute_CheckWriteToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "report.json")

	var stdout, stderr bytes.Buffer
	args := []string{
		"check",
		"-s", "Python was designed by Guido van Rossum.",
		"-a", "Python was designed by Guido van Rossum.",
		"-f", "json",
		"-o", outPath,
	}

	code := execute(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Report successfully written to:") {
		t.Fatalf("expected confirmation message, got: %s", stdout.String())
	}
}

func TestExecute_Benchmark(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execute([]string{"benchmark"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Benchmark Complete:") {
		t.Fatalf("expected 'Benchmark Complete:' in output, got: %s", stdout.String())
	}
}
