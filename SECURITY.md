# Security Policy

## Supported Versions

We provide security updates and patches for the following versions of VeriFact / HalluciDetect:

| Version | Supported          | Security Maintenance Status |
| :------ | :----------------- | :-------------------------- |
| 1.0.x   | :white_check_mark: | Active support & patches   |
| < 1.0   | :x:                | Deprecated                  |

---

## Reporting a Vulnerability

We take the security of VeriFact and the integrity of AI verification pipelines seriously. If you discover a security vulnerability or potential threat in this codebase, please follow the coordinated disclosure process outlined below.

### 🔒 How to Report

**Please DO NOT report security vulnerabilities via public GitHub issues, discussions, or pull requests.**

Instead, please report security concerns via one of the following private channels:

1. **GitHub Private Security Advisories**:  
   Navigate to the **Security** tab of this repository and click **"Report a vulnerability"** to open a confidential advisory.
2. **Email Security Disclosure**:  
   Send an encrypted or private email to `security@verifact.dev` (or the repository maintainers) with the subject:  
   `[SECURITY] VeriFact Vulnerability Report - <Brief Summary>`

### 📋 What to Include in Your Report

To help us triage and resolve the issue quickly, please provide:
- A descriptive summary of the vulnerability (e.g., Denial of Service in regex parser, unbounded memory allocation during document chunking, prompt injection bypass).
- Step-by-step instructions or minimal reproducible proof-of-concept (PoC) code/inputs.
- Affected component(s) (e.g., `internal/claim/extractor.go`, `internal/document/index.go`, `internal/api/`).
- Impact assessment: What can an attacker achieve?
- Any suggested remediations or patches.

### ⏱️ Response Timeline

- **Initial Acknowledgement**: Within **48 hours** of report receipt.
- **Triage & Assessment**: Within **5 business days** with status update.
- **Patch Release & Disclosure**: Fixes will be prepared in a private fork and released alongside a public CVE / Security Advisory once validated.

---

## 🛡️ Threat Model & Security Best Practices

When deploying VeriFact in production or CI/CD pipelines, please observe the following security guidelines:

### 1. Processing Untrusted AI & User Inputs
- **Resource Constraints**: By default, VeriFact processes text inputs using efficient in-memory string algorithms. If hosting the REST API publicly, configure request body size limits (`http.MaxBytesReader`) and reverse-proxy rate limiting (e.g., NGINX, Cloudflare).
- **Regex ReDoS Protection**: All regular expressions used in claim extraction and numeric parsing are bounded and tested against catastrophic backtracking.

### 2. LLM API Key Storage & Handling
- If using the optional `LLMJudge` (`--enable-llm-judge`), **NEVER** hardcode API keys in source files or commit them to version control.
- Use environment variables (e.g., `GEMINI_API_KEY`, `OPENAI_API_KEY`) or secret managers (e.g., HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager).
- VeriFact never logs, reflects, or includes API keys in JSON error responses or terminal reports.

### 3. Network & CORS Configuration
- In production, configure CORS headers in `internal/api/routes.go` to restrict origins to your internal domain rather than wildcard `*`.
- When running in Docker, bind the server to private network interfaces or place it behind a TLS-terminating ingress proxy.
