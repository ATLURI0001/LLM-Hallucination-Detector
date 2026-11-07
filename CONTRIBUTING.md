# Contributing to VeriFact / HalluciDetect

Thank you for your interest in contributing to **VeriFact / HalluciDetect**! We welcome contributions from the community to help make LLM output more reliable, factual, and trustworthy.

---

## 🛠️ Development Setup

### Prerequisites
- [Go 1.22+](https://golang.org/dl/)
- Git

### Getting Started
1. **Fork the repository** on GitHub.
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/ATLURI0001/llm-hallucination-detector.git
   cd llm-hallucination-detector
   ```
3. **Verify the test suite**:
   ```bash
   go test -v ./...
   ```
4. **Run benchmarks**:
   ```bash
   go run ./cmd/hallucidetect benchmark
   ```

---

## 🧪 Testing Guidelines

- Any new feature, claim extraction rule, or NLI heuristic **must include automated unit tests**.
- Place unit tests alongside the code in `*_test.go` files.
- Ensure all tests pass with the race detector enabled:
  ```bash
  go test -v -race ./...
  ```
- Run static analysis:
  ```bash
  go vet ./...
  ```

---

## 🔄 Pull Request Guidelines

1. Create a feature branch with a descriptive name:
   ```bash
   git checkout -b feat/add-date-consistency-check
   ```
2. Commit your changes following [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` New capability or heuristic
   - `fix:` Bug fix in parser or scorer
   - `docs:` Documentation improvements
   - `test:` Adding or updating tests
   - `perf:` Performance optimizations
3. Push to your fork and submit a Pull Request to `main`.
4. Ensure CI checks pass on all platforms (Linux, macOS, Windows).

---

## 📜 Code of Conduct

Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for details.
