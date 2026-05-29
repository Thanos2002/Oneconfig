# Contributing to OneConfig

First off, thank you for considering contributing to OneConfig! It's people like you that make OneConfig a universal configuration engine.

## How to Contribute

### Adding a New Detector

OneConfig uses a highly extensible detector engine located in `internal/generate/`. Each technology has its own file (e.g. `detector_node.go`, `detector_python.go`, `detector_go.go`). If you want to add support for a new language, framework, or package manager, you can do so by creating or modifying a detector!

1. Browse the existing `detector_*.go` files in `internal/generate/` to see how current detectors work.
2. Check if your ecosystem (e.g. Node.js, Python, Rust, Go, Java, Ruby) already has a detector file.
3. If it does, simply extend the logic in that file (for example, adding support for a new lockfile like `bun.lock` or a new framework like `Gradio`).
4. If it's a completely new ecosystem, create a new `detector_<name>.go` file and implement the `Detector` interface (defined in `internal/generate/scanner.go`):
   ```go
   type Detector interface {
       Detect(projectDir string, result *ScanResult) error
   }
   ```
5. Register your new detector in the `NewScanner()` function in `internal/generate/scanner.go`.

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
