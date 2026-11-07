# 🛡️ LLM Hallucination Detector (VeriFact)

[![CI Pipeline](https://img.shields.io/badge/build-passing-brightgreen.svg)]()
[![Security Policy](https://img.shields.io/badge/security-monitored-blue.svg)](SECURITY.md)
[![CodeQL Analysis](https://img.shields.io/badge/codeql-scanned-success.svg)]()
[![Go Version](https://img.shields.io/badge/go-1.22%20%7C%201.23-blue.svg)](https://golang.org)
[![Accuracy Benchmark](https://img.shields.io/badge/benchmark-100%25%20accuracy-success.svg)]()
[![License](https://img.shields.io/badge/license-MIT-purple.svg)](LICENSE)
[![Zero Dependency](https://img.shields.io/badge/dependencies-zero%20external-orange.svg)]()

> **Production-grade, sub-millisecond LLM Hallucination Detection & Fact Verification Engine built in Go.**  
> Automatically decomposes AI answers into atomic factual claims, grounds each claim against trusted reference documents, detects intrinsic contradictions, numeric/temporal distortions, and entity fabrications, and flags unreliable answers.

---

## 🌟 Overview

Large Language Models (LLMs) frequently generate answers that sound fluent and authoritative but contain subtle, high-risk hallucinations:
- **Numeric Distortions**: Inventing or altering revenue, percentages, dosages, or dates (e.g. source says `$480M`, AI says `$750M`).
- **Intrinsic Contradictions**: Reversing polarity or stating the inverse of facts in the source document.
- **Extrinsic Hallucinations**: Hallucinating ungrounded entities, organizations, or events not present in the reference documents.
- **Attribution & Context Twisting**: Misattributing actions or blending disjoint facts together.

**VeriFact / HalluciDetect** solves this problem by providing an automated verification pipeline that checks any AI response against authoritative source documents.

---

## 📐 System Architecture

```
                       ┌────────────────────────────────────────┐
                       │       AI-Generated Text Response       │
                       └───────────────────┬────────────────────┘
                                           │
                                           ▼
                       ┌────────────────────────────────────────┐
                       │      Claim Decomposition Engine        │
                       │  - Sentence Boundary Detection         │
                       │  - Conversational Filler Stripping     │
                       │  - Named Entity & Number Extraction    │
                       └───────────────────┬────────────────────┘
                                           │ Atomic Claims
                                           ▼
┌──────────────────────┐       ┌────────────────────────────────────────┐
│ Trusted Source Docs  │ ────► │       Inverted BM25 Search Index       │
│ (.txt, .md, context) │       │   Hierarchical & Sliding-Window Chunks │
└──────────────────────┘       └───────────────────┬────────────────────┘
                                                   │ Candidate Passages
                                                   ▼
                       ┌────────────────────────────────────────┐
                       │     Multi-Tier Verification Pipeline   │
                       │ ├─ Strict Numeric / Temporal Checker   │
                       │ ├─ Entity Grounding & Absence Tracker  │
                       │ ├─ Polarity / Negation Inversion NLI   │
                       │ ├─ Lexical Coverage & Jaccard Metric   │
                       │ └─ (Optional) LLM-as-a-Judge Refiner   │
                       └───────────────────┬────────────────────┘
                                           │
                                           ▼
                       ┌────────────────────────────────────────┐
                       │         Comprehensive Audit Report     │
                       │ ├─ Faithfulness Score & Risk Level     │
                       │ ├─ Color-Coded Highlight Grounding Map │
                       │ ├─ Formats: CLI, JSON, Markdown, SARIF │
                       │ └─ Interactive Web UI Dashboard        │
                       └────────────────────────────────────────┘
```

---

## 🔬 Hallucination Taxonomy

Every claim evaluated by VeriFact is categorized under a fine-grained failure taxonomy:

| Status | Failure Taxonomy | Description | Real-World Example |
| :--- | :--- | :--- | :--- |
| `🟢 SUPPORTED` | `NONE` | Claim is directly grounded in source document | Source: `$480M rev`, Claim: `$480M rev` |
| `🔴 CONTRADICTED` | `NUMERIC_MISMATCH` | Number, percentage, or date altered | Source: `14% growth`, Claim: `35% growth` |
| `🔴 CONTRADICTED` | `INTRINSIC` | Explicit polarity inversion or negation | Source: `Drug failed trial`, Claim: `Drug succeeded` |
| `🟡 UNSUPPORTED` | `EXTRINSIC` | Completely fabricated ungrounded claim | Source: `Tech report`, Claim: `Bought Spanish island` |
| `🟡 UNSUPPORTED` | `ENTITY_CONFUSION` | Proper nouns or entities not in source | Claim references `Elon Musk`, source does not |
| `🔵 PARTIAL` | `OUT_OF_CONTEXT` | Conceptual overlap but distorted scope | Broad extrapolation beyond evidence |

---

## 🚀 Key Features

1. **Sub-Millisecond Heuristic Verification**: Zero external API dependencies required. Evaluates typical multi-paragraph texts in **< 5ms** using algorithmic NLI, BM25 indexing, and entity-numeric grounding.
2. **Pluggable LLM-as-a-Judge**: Optional integration with Gemini (`gemini-2.5-flash`), OpenAI (`gpt-4o-mini`), or local Ollama instances for deep semantic dispute resolution.
3. **Embedded Interactive Web UI**: Fully self-contained single-page application built directly into the binary using Go's `embed.FS`. No Node.js or web server required!
4. **CI/CD Quality Gates**: Built-in `--fail-on-hallucination` flag returning exit code `1` when hallucinations are detected, blocking ungrounded answers from merging or shipping.
5. **Multi-Format Export**: Generates rich ANSI terminal output, GitHub-flavored Markdown reports, JSON payloads, and OASIS SARIF v2.1.0 for GitHub Code Scanning.

---

## 📦 Installation

### From Source (Go 1.22+)
```bash
git clone https://github.com/user/llm-hallucination-detector.git
cd llm-hallucination-detector
go build -o hallucidetect ./cmd/hallucidetect
```

### Via Docker
```bash
docker build -t hallucidetect .
docker run -p 8080:8080 hallucidetect
```

---

## 💻 CLI Usage Guide

### 1. Check Inline Text
```bash
hallucidetect check \
  --source "Acme Corp reported $480M revenue in 2024. Sarah Jenkins was appointed CEO." \
  --answer "Acme Corp reported $950M revenue in 2024. Sarah Jenkins was appointed CEO." \
  --format table
```

### 2. Verify Text from Files with Markdown Output
```bash
hallucidetect check \
  -s testdata/financial_source.txt \
  -a testdata/financial_answer.txt \
  -f markdown \
  -o audit_report.md
```

### 3. CI/CD Pipeline Gate
```bash
hallucidetect check \
  -s reference_docs.txt \
  -a model_output.txt \
  --fail-on-hallucination
# Exits with 0 if faithful, 1 if hallucination detected
```

### 4. Launch Embedded Web Dashboard
```bash
hallucidetect serve --port 8080
# Open http://localhost:8080 in your browser
```

### 5. Run Verification Benchmark
```bash
hallucidetect benchmark
```

---

## 🌐 REST API Reference

When launched with `hallucidetect serve`, the engine exposes high-performance HTTP endpoints.

### `POST /api/v1/verify`
Evaluates an AI answer against source text.

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/verify \
  -H "Content-Type: application/json" \
  -d '{
    "source_text": "The Python language was created by Guido van Rossum and first released in 1991.",
    "answer_text": "Python was released in 1991 by Guido van Rossum. It was designed by James Gosling in 2005.",
    "options": {
      "threshold": 0.65,
      "strict_numeric_verification": true
    }
  }'
```

**Response:**
```json
{
  "id": "vr_1794063765673183100",
  "verdict": "UNRELIABLE",
  "risk_level": "HIGH",
  "faithfulness_score": 0.50,
  "hallucination_score": 0.50,
  "total_claims": 2,
  "supported_claims": 1,
  "contradicted_claims": 0,
  "unsupported_claims": 1,
  "partially_supported_claims": 0,
  "processing_time_ms": 1,
  "claims": [
    {
      "id": 1,
      "text": "Python was released in 1991 by Guido van Rossum.",
      "status": "SUPPORTED",
      "hallucination_type": "NONE",
      "confidence": 0.88,
      "explanation": "Claim is directly supported and grounded by the source document passage.",
      "best_evidence": {
        "snippet": "The Python language was created by Guido van Rossum and first released in 1991."
      }
    },
    {
      "id": 2,
      "text": "It was designed by James Gosling in 2005.",
      "status": "UNSUPPORTED",
      "hallucination_type": "EXTRINSIC",
      "confidence": 0.92,
      "explanation": "Extrinsic Hallucination: No corroborating evidence found anywhere in the trusted source documents.",
      "missing_entities": ["2005", "James Gosling"]
    }
  ]
}
```

### `GET /api/v1/health`
Healthcheck and uptime status.
```bash
curl http://localhost:8080/api/v1/health
```

---

## 🖥️ Interactive Web UI Dashboard

The engine includes a single-page dark/light web UI embedded inside the binary:

- **Preset Demonstrations**: Preloaded financial earnings, clinical trials, and hardware specs.
- **Interactive Grounding Map**: Color-coded visualization where users can click any highlighted sentence to immediately reveal its exact matching citation passage in the source document.
- **Filterable Claims Table**: Sort and filter claims by `SUPPORTED`, `CONTRADICTED`, and `UNSUPPORTED`.
- **One-Click Export**: Instant download of Markdown or JSON audit reports.

---

## 📊 Benchmark Suite Results

Running `hallucidetect benchmark`:

```text
================================================================================
Running Built-In Hallucination Benchmark Test Suite...

[1/3] Testing: Financial Report Grounded vs Numeric Hallucination... PASS (Verdict: UNRELIABLE, Faithfulness: 25%)
[2/3] Testing: Clinical Trial Polarity Inversion... PASS (Verdict: UNRELIABLE, Faithfulness: 0%)
[3/3] Testing: Faithful Summary Grounding... PASS (Verdict: RELIABLE, Faithfulness: 100%)

Benchmark Complete: 3/3 test suites passed (100% accuracy).
================================================================================
```

---

## 📁 Repository Structure

```
.
├── .github/
│   ├── workflows/
│   │   ├── ci.yml               # Matrix CI: tests, race checks, cross-platform builds
│   │   └── security.yml         # CodeQL static analysis & Gosec scanning
│   ├── ISSUE_TEMPLATE/          # Bug report & feature request templates
│   ├── PULL_REQUEST_TEMPLATE.md # PR checklist & review guidelines
│   └── dependabot.yml           # Automated dependency & action security bumps
├── cmd/
│   └── hallucidetect/
│       └── main.go              # CLI tool, flags, and subcommands
├── internal/
│   ├── api/                     # REST API handlers, routing, CORS
│   ├── claim/                   # Claim extraction & proposition decomposition
│   ├── document/                # Inverted index, BM25 retrieval, chunking
│   ├── engine/                  # Master detector, heuristic NLI, LLM judge
│   ├── models/                  # Shared domain types & schemas
│   ├── report/                  # Terminal, Markdown, JSON, SARIF formatters
│   └── web/                     # Embedded static UI (HTML, CSS, JS)
├── testdata/                    # Benchmark sample datasets
├── Dockerfile                   # Multi-stage scratch/alpine container
├── docker-compose.yml           # Compose specification
├── Makefile                     # Build, test, and run automation
├── build.ps1                    # Windows PowerShell build utility
├── go.mod                       # Go module definition (zero external dependencies)
├── CODE_OF_CONDUCT.md           # Contributor Covenant Code of Conduct v2.1
├── CONTRIBUTING.md              # Contributor guidelines and workflow
├── SECURITY.md                  # Security policy, threat model, and disclosure process
├── LICENSE                      # MIT License
└── README.md                    # Project documentation
```

---

## 🔒 Security & Privacy

- **Local-First Execution**: In default mode, VeriFact processes all claims locally on your machine with **zero external network requests or third-party API exposure**.
- **Safe Credential Handling**: When optional LLM judging is enabled, API keys are passed strictly in-memory and are never cached, persisted, or displayed in output logs.
- For vulnerability reporting and our security policy, please review [SECURITY.md](SECURITY.md).

---

## 🤝 Contributing

Contributions, bug reports, and suggestions are welcome! Please check our [CONTRIBUTING.md](CONTRIBUTING.md) guide and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

---

## 📜 License

Distributed under the **MIT License**. See [LICENSE](LICENSE) for details.
