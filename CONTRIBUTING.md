# Contributing to OneConfig

First off, thank you for considering contributing to OneConfig! It's people like you that make OneConfig a universal configuration engine.

## How to Contribute

### Adding a New Detector

OneConfig uses a highly extensible detector engine located in `internal/generate/`. If you want to add support for a new language, framework, or package manager, you can do so by creating or modifying a detector!

1. Look in `internal/generate/detectors.go` and `internal/generate/detectors_extended.go`.
2. Check if your ecosystem (e.g. Node.js, Python, Rust, Go, Java, Ruby) already has a detector.
3. If it does, simply extend the logic (for example, adding support for a new lockfile like `bun.lock` or a new framework like `Gradio`).
4. If it's a completely new ecosystem, implement the `Detector` interface:
   ```go
   type Detector interface {
       Detect(projectDir string, result *ScanResult) error
   }
   ```
5. Register your new detector in `internal/generate/scanner.go`.

### Development Workflow

1. Fork and clone the repository.
2. Make your changes in a new branch.
3. Run the test suite:
   ```bash
   go test ./...
   ```
4. Format your code:
   ```bash
   gofmt -w .
   ```
5. Build and test your changes locally:
   ```bash
   go build -o oneconfig .
   ./oneconfig generate
   ```
6. Submit a Pull Request with a clear description of the feature or bug fix.

### Reporting Bugs

If you find a project that `oneconfig generate` fails to detect or configures incorrectly, please open an issue! Be sure to include:
- A link to the repository (if public).
- The `oneconfig generate --stdout` output.
- What you expected it to generate instead.

Thanks for helping us make OneConfig amazing!
